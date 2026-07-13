package service

import "billing-svc/internal/models"

type ResponseWriter interface {
	ReportOrderCreated(e models.OrderCreatedEvent) error
	ReportOrderUpdated(e models.OrderUpdatedEvent) error
}
