package messaging

import (
	"fmt"

	"order-svc/internal/config"
	"order-svc/internal/models"
)

// Source identifies where to consume from.
//
// For RabbitMQ only `Name` matters (it is the queue; routing happens
// server-side via bindings). For Kafka `Name` is the topic, `Group` is the
// consumer group id, and `EventTypes` replaces AMQP routing-key wildcards:
// Kafka has no server-side pattern matching, so filtering by the `event_type`
// field happens on the client. An empty `EventTypes` means "accept everything".
type Source struct {
	Name       string
	Group      string
	EventTypes []string
}

type Broker interface {
	RegisterConsumer(s Source, h HandlerFunc)
	Run()
	// PublishOrderEvent publishes ORDER_PAID / ORDER_CANCELLED to the `order`
	// topic (exchange) so delivery-svc can react.
	PublishOrderEvent(e models.OrderEvent) error
	// GetOrderPaymentSource is the billing payment-result stream.
	GetOrderPaymentSource() Source
	// GetDeliverySource is the delivery status stream.
	GetDeliverySource() Source
}

func InitBroker(cfg config.Config) (Broker, error) {
	switch cfg.BrokerType {
	case "RABBITMQ":
		br, err := NewRabbitImpl(config.GetRabbitConfig())
		if err != nil {
			return nil, fmt.Errorf("can't init rabbitmq impl: %s", err)
		}
		return br, nil
	case "KAFKA":
		return NewKafkaImpl(config.GetKafkaConfig()), nil
	default:
		return nil, fmt.Errorf("unsupported BROKER_TYPE %q (supported: RABBITMQ, KAFKA)", cfg.BrokerType)
	}
}
