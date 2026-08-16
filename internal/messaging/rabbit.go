package messaging

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"

	"order-svc/internal/config"
	"order-svc/internal/models"
)

type HandlerFunc func(b []byte) (bool, error)

type RabbitImpl struct {
	conn          *amqp.Connection
	publisher     *amqp.Channel
	publisherLock sync.Mutex
	consumers     map[string]HandlerFunc
	cfg           config.RabbitConfig
}

// RabbitMQ filters server-side via the queue binding, so no EventTypes here.
func (r *RabbitImpl) GetOrderPaymentSource() Source {
	return Source{Name: r.cfg.OrderPaymentConsumer}
}

func (r *RabbitImpl) GetDeliverySource() Source {
	return Source{Name: r.cfg.DeliveryConsumer}
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

// declareExchange declares a durable topic exchange (idempotent).
func (r *RabbitImpl) declareExchange(ch *amqp.Channel, exchange string) error {
	if exchange == "" {
		return nil
	}
	return ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil)
}

// declareQueueAndBind declares the consumer queue (durable) and binds it to the
// exchange it consumes from.
func (r *RabbitImpl) declareQueueAndBind(ch *amqp.Channel, queue string) error {
	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare queue %q: %w", queue, err)
	}

	exchange, rk := r.bindingFor(queue)
	if exchange == "" {
		return nil
	}
	if err := ch.QueueBind(queue, rk, exchange, false, nil); err != nil {
		return fmt.Errorf("bind queue %q to %q with rk %q: %w", queue, exchange, rk, err)
	}
	return nil
}

// bindingFor returns the exchange and routing key a consumer queue binds to.
func (r *RabbitImpl) bindingFor(queue string) (exchange, routingKey string) {
	switch queue {
	case r.cfg.OrderPaymentConsumer:
		return r.cfg.OrderPaymentExchange, r.cfg.OrderPaymentRoutingKey
	case r.cfg.DeliveryConsumer:
		return r.cfg.DeliveryExchange, r.cfg.DeliveryRoutingKey
	}
	return "", ""
}

func (r *RabbitImpl) RegisterConsumer(s Source, h HandlerFunc) {
	r.consumers[s.Name] = h
}

// PublishOrderEvent publishes an order event to the `order` exchange. The
// routing key is the event type (ORDER_PAID / ORDER_CANCELLED).
func (r *RabbitImpl) PublishOrderEvent(e models.OrderEvent) error {
	body, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal order event: %w", err)
	}

	r.publisherLock.Lock()
	defer r.publisherLock.Unlock()

	slog.Info("publishing order event",
		"exchange", r.cfg.OrderExchange, "event_type", e.EventType, "order_id", e.OrderId)

	return r.publisher.Publish(
		r.cfg.OrderExchange, e.EventType, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    e.EventId.String(),
			Body:         body,
		},
	)
}

func (r *RabbitImpl) Run() {
	defer r.conn.Close()
	defer r.publisher.Close()

	// Declare every exchange we produce to or consume from.
	for _, exchange := range []string{r.cfg.OrderExchange, r.cfg.OrderPaymentExchange, r.cfg.DeliveryExchange} {
		if err := r.declareExchange(r.publisher, exchange); err != nil {
			slog.Error("declare topology", "op", "exchange", "exchange", exchange, "err", err)
			return
		}
	}

	// Declare all consumer queues + bindings up front, on the publisher channel.
	for queue := range r.consumers {
		if err := r.declareQueueAndBind(r.publisher, queue); err != nil {
			slog.Error("declare topology", "queue", queue, "err", err)
			return
		}
		slog.Info("queue ready", "queue", queue)
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
