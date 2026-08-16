package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

type PGConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

func (c PGConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode,
	)
}

func GetPGConfig() PGConfig {
	return PGConfig{
		Host:     getenv("PG_HOST", "postgres"),
		Port:     getenv("PG_PORT", "5432"),
		User:     getenv("PG_USER", "order"),
		Password: getenv("PG_PASSWORD", "order"),
		Database: getenv("PG_DATABASE", "order"),
		SSLMode:  getenv("PG_SSLMODE", "disable"),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// RabbitConfig describes the AMQP topology.
//
// order-svc produces to the `order` exchange (ORDER_PAID / ORDER_CANCELLED) and
// consumes two exchanges: `order-payment` (billing result) and `delivery`
// (delivery status). Each consumer has its own queue bound to the matching
// exchange.
type RabbitConfig struct {
	DSN           string
	OrderExchange string

	OrderPaymentExchange   string
	OrderPaymentConsumer   string
	OrderPaymentRoutingKey string

	DeliveryExchange   string
	DeliveryConsumer   string
	DeliveryRoutingKey string
}

// KafkaConfig mirrors RabbitConfig: exchanges become topics, routing keys
// become `event_type` values inside the payload (filtered client-side, since
// Kafka has no wildcard bindings), and the consumer queue becomes a group.
type KafkaConfig struct {
	Brokers    []string
	OrderTopic string

	OrderPaymentTopic      string
	OrderPaymentGroup      string
	OrderPaymentEventTypes []string

	DeliveryTopic      string
	DeliveryGroup      string
	DeliveryEventTypes []string
}

type RestConfig struct {
	// AdvertSvcURL is the base URL of advert-cmd-svc (used to verify delivery
	// availability at order creation).
	AdvertSvcURL string
	// DeliverySvcURL is the base URL of delivery-service (used to fetch provider
	// quotes when the user selects a delivery option).
	DeliverySvcURL string
	// BillingSvcURL is the base URL of billing-svc (used to initiate payment
	// transactions for orders).
	BillingSvcURL string
}

type Config struct {
	BrokerType string
	Rest       RestConfig
}

func InitConfig() *Config {
	brokerType := os.Getenv("BROKER_TYPE")
	return &Config{BrokerType: brokerType, Rest: initRestConfig()}
}

func GetKafkaConfig() KafkaConfig {
	return KafkaConfig{
		Brokers:    strings.Split(getenv("KAFKA_BROKERS", "kafka-1:9092,kafka-2:9092,kafka-3:9092"), ","),
		OrderTopic: getenv("KAFKA_ORDER_TOPIC", "order"),

		OrderPaymentTopic: getenv("KAFKA_ORDER_PAYMENT_TOPIC", "order-payment"),
		OrderPaymentGroup: getenv("KAFKA_ORDER_PAYMENT_GROUP", "ordersvc.order-payment"),
		OrderPaymentEventTypes: strings.Split(
			getenv("KAFKA_ORDER_PAYMENT_EVENT_TYPES", "PAYMENT_SUCCESS,PAYMENT_FAILED"), ","),

		DeliveryTopic: getenv("KAFKA_DELIVERY_TOPIC", "delivery"),
		DeliveryGroup: getenv("KAFKA_DELIVERY_GROUP", "ordersvc.delivery"),
		DeliveryEventTypes: strings.Split(
			getenv("KAFKA_DELIVERY_EVENT_TYPES", "DELIVERY_SENT,DELIVERY_DELIVERED,DELIVERY_CANCELLED"), ","),
	}
}

func initRestConfig() RestConfig {
	return RestConfig{
		AdvertSvcURL:   getenv("ADVERT_SVC_URL", "http://advert-cmd-svc:8080"),
		DeliverySvcURL: getenv("DELIVERY_SVC_URL", "http://delivery-app:8080"),
		BillingSvcURL:  getenv("BILLING_SVC_URL", "http://billing-app:8080"),
	}
}

func GetRabbitConfig() RabbitConfig {
	user := os.Getenv("RABBIT_USERNAME")
	password := os.Getenv("RABBIT_PASSWORD")
	host := os.Getenv("RABBIT_HOST")
	port := os.Getenv("RABBIT_PORT")

	u := url.URL{Scheme: "amqp",
		User: url.UserPassword(user, password),
		Host: fmt.Sprintf("%s:%s", host, port)}

	return RabbitConfig{
		DSN:           u.String(),
		OrderExchange: getenv("RABBIT_ORDER_EXCHANGE", "order"),

		OrderPaymentExchange:   getenv("RABBIT_ORDER_PAYMENT_EXCHANGE", "order-payment"),
		OrderPaymentConsumer:   getenv("RABBIT_ORDER_PAYMENT_CONSUMER", "ordersvc.consumer.for.order-payment"),
		OrderPaymentRoutingKey: getenv("RABBIT_ORDER_PAYMENT_ROUTING_KEY", "#"),

		DeliveryExchange:   getenv("RABBIT_DELIVERY_EXCHANGE", "delivery"),
		DeliveryConsumer:   getenv("RABBIT_DELIVERY_CONSUMER", "ordersvc.consumer.for.delivery"),
		DeliveryRoutingKey: getenv("RABBIT_DELIVERY_ROUTING_KEY", "#"),
	}
}
