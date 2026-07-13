package messaging

import (
	"billing-svc/internal/config"
	"billing-svc/internal/models"
	"fmt"
)

type Broker interface {
	RegisterConsumer(dataSourceName string, h HandlerFunc)
	Run()
	ReportOrderCreated(e models.OrderCreatedEvent) error
	ReportOrderUpdated(e models.OrderUpdatedEvent) error
	GetBillingPaymentDataSourceName() string
}

func InitBroker(cfg config.Config) (Broker, error) {
	var b Broker

	if "RABBITMQ" == cfg.BrokerType {
		br, err := NewRabbitImpl(config.GetRabbitConfig())
		if err != nil {
			return nil, fmt.Errorf("can't init rabbitmq impl: %s", err)
		}
		b = br
	}

	return b, nil
}
