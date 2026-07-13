package messaging

import (
	"billing-svc/internal/config"
	"billing-svc/internal/models"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type HandlerFunc func(b []byte) (bool, error)

type RabbitImpl struct {
	conn          *amqp.Connection
	publisher     *amqp.Channel
	publisherLock sync.Mutex
	consumers     map[string]HandlerFunc
	cfg           config.RabbitConfig
}

func (r *RabbitImpl) GetBillingPaymentDataSourceName() string {
	return r.cfg.BillingConsumer
}

func NewRabbitImpl(cfg config.RabbitConfig) (*RabbitImpl, error) {
	conn, err := amqp.Dial(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("connect to rabbitmq: %w", err)
	}

	publisher, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open publisher channel: %w", err)
	}

	r := &RabbitImpl{conn: conn, consumers: make(map[string]HandlerFunc), cfg: cfg, publisher: publisher}

	return r, nil
}

// declareExchange ensures the topic exchange used for outbound billing
// events exists. Idempotent: declaring an existing exchange with the same
// type/params is a no-op. Fails if a non-topic exchange with the same name
// already exists.
func (r *RabbitImpl) declareExchange(ch *amqp.Channel, exchange string) error {
	if r.cfg.BillingExchange == "" {
		return nil
	}
	return ch.ExchangeDeclare(
		exchange,
		"topic",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,
	)
}

// declareQueueAndBind ensures the consumer queue exists (durable) and is
// bound to the billing exchange with the queue's incoming routing key.
// If BillingExchange is empty, only the queue is declared (default exchange).
func (r *RabbitImpl) declareQueueAndBind(ch *amqp.Channel, queue string) error {
	if _, err := ch.QueueDeclare(
		queue,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	); err != nil {
		return fmt.Errorf("declare queue %q: %w", queue, err)
	}

	if r.cfg.BillingExchange == "" || r.cfg.OrderExchange == "" {
		return nil
	}

	rks := r.routingKeyFor(queue)
	if len(rks) == 0 {
		return nil
	}

	for _, rk := range rks {
		if err := ch.QueueBind(queue, rk, r.cfg.BillingExchange, false, nil); err != nil {
			return fmt.Errorf("bind queue %q to %q with rk %q: %w", queue, r.cfg.BillingExchange, rk, err)
		}
	}

	return nil
}

func (r *RabbitImpl) routingKeyFor(queue string) []string {
	switch queue {
	case r.cfg.BillingConsumer:
		return []string{r.cfg.BillingConsumerRoutingKey}
	}
	return []string{}
}

func (r *RabbitImpl) RegisterConsumer(queueName string, h HandlerFunc) {
	r.consumers[queueName] = h
}

func (r *RabbitImpl) produceOrderEvent(routingKey, messageId string, body []byte) error {
	r.publisherLock.Lock()
	defer r.publisherLock.Unlock()

	return r.publisher.Publish(
		r.cfg.OrderExchange, routingKey, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    messageId,
			Body:         body,
		},
	)
}

func (r *RabbitImpl) ReportOrderCreated(e models.OrderCreatedEvent) error {
	body, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal order.created: %w", err)
	}
	slog.Info("publishing order.created", "order_id", e.OrderId, "rk", r.cfg.OrderCreatedRoutingKey)
	return r.produceOrderEvent(r.cfg.OrderCreatedRoutingKey, e.EventId.String(), body)
}

func (r *RabbitImpl) ReportOrderUpdated(e models.OrderUpdatedEvent) error {
	body, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal order.updated: %w", err)
	}
	slog.Info("publishing order.updated", "order_id", e.OrderId, "status", e.Status, "rk", r.cfg.OrderUpdatedRoutingKey)
	return r.produceOrderEvent(r.cfg.OrderUpdatedRoutingKey, e.EventId.String(), body)
}

func (r *RabbitImpl) Run() {
	defer r.conn.Close()
	defer r.publisher.Close()

	exchangesToDeclare := []string{r.cfg.BillingExchange, r.cfg.OrderExchange}
	for _, exchange := range exchangesToDeclare {
		if err := r.declareExchange(r.publisher, exchange); err != nil {
			slog.Error("declare topology", "op", "exchange", "err", err)
			return
		}
	}

	// Declare the entire topology up front, on the publisher channel, so
	// queues/exchange are guaranteed to exist before any consumer starts.
	for queue := range r.consumers {
		if err := r.declareQueueAndBind(r.publisher, queue); err != nil {
			slog.Error("declare topology", "queue", queue, "err", err)
			return
		}
		slog.Info("queue ready", "queue", queue, "exchange", r.cfg.BillingExchange)
	}

	wg := &sync.WaitGroup{}
	for k, v := range r.consumers {
		wg.Add(1)
		go r.runConsumer(k, v, wg)
	}
	wg.Wait()
}

func (r *RabbitImpl) runConsumer(queue string, handler HandlerFunc, wg *sync.WaitGroup) {
	defer wg.Done()
	ch, err := r.conn.Channel()
	if err != nil {
		slog.Error("create channel", "err", err)
		return
	}

	defer ch.Close()

	if err := ch.Qos(1, 0, false); err != nil {
		slog.Error("set qos", "err", err)
		return
	}

	msgs, err := ch.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		slog.Error("consume message", "err", err)
		return
	}

	for msg := range msgs {
		ok, err := handler(msg.Body)
		if err != nil {
			msg.Nack(false, false)
			continue
		}

		if !ok {
			msg.Ack(false)
			continue
		}

		msg.Ack(false)
	}
}
