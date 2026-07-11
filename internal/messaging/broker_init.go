package messaging

import (
	"billing-svc/internal/config"
	"billing-svc/internal/models"
	"fmt"
)

type Broker interface {
	RegisterConsumer(dataSourceName string, h HandlerFunc)
	Run()
	ReportOrderCreated(b models.Billing) error
	ReportOrderUpdated(b models.Billing) error
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
