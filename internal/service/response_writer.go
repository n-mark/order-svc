package service

import "billing-svc/internal/models"

type ResponseWriter interface {
	ReportOrderCreated(b models.Billing) error
	ReportOrderUpdated(b models.Billing) error
}
