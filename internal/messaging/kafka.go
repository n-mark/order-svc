package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"

	"order-svc/internal/config"
	"order-svc/internal/models"
)

type consumer struct {
	source  Source
	handler HandlerFunc
}

// KafkaImpl is the Kafka counterpart of RabbitImpl.
//
// Topology mapping: exchange -> topic (`order` for produce, `order-payment` and
// `delivery` for consume), routing key -> the `event_type` field of the payload
// (server-side bindings become client-side filters), queue -> consumer group.
type KafkaImpl struct {
	cfg       config.KafkaConfig
	writer    *kafka.Writer
	consumers []consumer
}

func NewKafkaImpl(cfg config.KafkaConfig) *KafkaImpl {
	w := &kafka.Writer{
		Addr: kafka.TCP(cfg.Brokers...),
		// Same key (order_id) -> same partition -> events of one order stay ordered.
		Balancer: &kafka.Hash{},
		// acks=all + min.insync.replicas=2 => no data loss if one broker dies.
		RequiredAcks:           kafka.RequireAll,
		AllowAutoTopicCreation: true,
		BatchTimeout:           50 * time.Millisecond,
	}
	return &KafkaImpl{cfg: cfg, writer: w}
}

func (k *KafkaImpl) GetOrderPaymentSource() Source {
	return Source{
		Name:       k.cfg.OrderPaymentTopic,
		Group:      k.cfg.OrderPaymentGroup,
		EventTypes: k.cfg.OrderPaymentEventTypes,
	}
}

func (k *KafkaImpl) GetDeliverySource() Source {
	return Source{
		Name:       k.cfg.DeliveryTopic,
		Group:      k.cfg.DeliveryGroup,
		EventTypes: k.cfg.DeliveryEventTypes,
	}
}

func (k *KafkaImpl) RegisterConsumer(s Source, h HandlerFunc) {
	k.consumers = append(k.consumers, consumer{source: s, handler: h})
}

// PublishOrderEvent publishes ORDER_PAID / ORDER_CANCELLED to the `order`
// topic, keyed by order_id so all events of one order stay ordered.
func (k *KafkaImpl) PublishOrderEvent(e models.OrderEvent) error {
	body, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal order event: %w", err)
	}

	slog.Info("publishing order event",
		"topic", k.cfg.OrderTopic, "event_type", e.EventType, "order_id", e.OrderId)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return k.writer.WriteMessages(ctx, kafka.Message{
		Topic: k.cfg.OrderTopic,
		Key:   []byte(e.OrderId.String()),
		Value: body,
	})
}

func (k *KafkaImpl) Run() {
	defer k.writer.Close()

	wg := &sync.WaitGroup{}
	for _, c := range k.consumers {
		wg.Add(1)
		go k.runConsumer(c, wg)
	}
	wg.Wait()
}

func (k *KafkaImpl) runConsumer(c consumer, wg *sync.WaitGroup) {
	defer wg.Done()

	const reconnectBackoff = 2 * time.Second

	for {
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers: k.cfg.Brokers,
			Topic:   c.source.Name,
			GroupID: c.source.Group,
			// Manual commits: commit only after the handler succeeded, so a crash
			// leads to redelivery instead of a lost message (at-least-once).
			CommitInterval: 0,
			MaxWait:        time.Second,
		})

		slog.Info("kafka consumer started",
			"topic", c.source.Name, "group", c.source.Group, "event_types", c.source.EventTypes)

		// consumeLoop only returns on a fatal read error (e.g. broker restart).
		// Recreate the reader so the consumer keeps working without a rollout
		// restart, instead of letting the goroutine die forever.
		consumeLoop(r, c)
		_ = r.Close()

		slog.Warn("kafka consumer reconnecting", "topic", c.source.Name, "group", c.source.Group)
		time.Sleep(reconnectBackoff)
	}
}

func consumeLoop(r *kafka.Reader, c consumer) {
	for {
		msg, err := r.FetchMessage(context.Background())
		if err != nil {
			slog.Error("fetch message", "topic", c.source.Name, "err", err)
			return
		}

		if !matchesEventType(msg.Value, c.source.EventTypes) {
			// Not ours (the topic carries several event types): commit so we
			// do not re-read it forever.
			if err := r.CommitMessages(context.Background(), msg); err != nil {
				slog.Error("commit filtered message", "topic", c.source.Name, "err", err)
			}
			continue
		}

		ok, err := c.handler(msg.Value)
		if err != nil {
			// No commit -> redelivery. Handlers must be idempotent.
			slog.Error("handle message", "topic", c.source.Name, "offset", msg.Offset, "err", err)
			continue
		}
		if !ok {
			slog.Debug("message skipped", "topic", c.source.Name, "offset", msg.Offset)
		}

		if err := r.CommitMessages(context.Background(), msg); err != nil {
			slog.Error("commit message", "topic", c.source.Name, "offset", msg.Offset, "err", err)
		}
	}
}

// matchesEventType is the client-side replacement for AMQP routing-key wildcards.
func matchesEventType(body []byte, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}

	var envelope struct {
		EventType string `json:"event_type"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		// Malformed payloads are handed to the handler, which reports the error.
		return true
	}

	for _, a := range allowed {
		if a == envelope.EventType {
			return true
		}
	}
	return false
}
