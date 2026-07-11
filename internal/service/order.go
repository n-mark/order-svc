package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"billing-svc/internal/models"
)

type OrderService struct {
	store          Store
	responseWriter ResponseWriter
}

func NewOrderService(s Store, rw ResponseWriter) *OrderService {
	return &OrderService{store: s, responseWriter: rw}
}

func (o *OrderService) ProcessOrder(ctx context.Context, order models.Order) (bool, error) {
	slog.Info("PROCESSING ORDER...", "order", order)

	// Idempotency check.
	if dup, err := o.store.EventAlreadyProcessed(ctx, order.EventId); err != nil {
		return false, err
	} else if dup {
		slog.Info("order already processed, skipping", "event_id", order.EventId)
		return true, nil
	}

	resp, err := o.store.Save(ctx, order)
	if err != nil {
		return false, err
	}

	pub := models.Billing{
		EventId:   uuid.New(),
		EventType: "SUCCESSFULLY_PAID",
		UserId:    resp.UserId,
		OrderId:   order.OrderId,
		Status:    "paid",
		Balance:   resp.Balance,
	}
	if err := o.responseWriter.ReportOrderUpdated(pub); err != nil {
		return false, err
	}
	if err := o.store.MarkEventProcessed(ctx, order.EventId, "SUCCESSFULLY_PAID"); err != nil {
		return false, err
	}
	return true, nil
}

func (o *OrderService) UpdateOrderStatusOnBillingResponse(ctx context.Context, order models.BillingResponse) (bool, error) {
	slog.Info("PROCESSING ORDER...", "order", order)

	// Idempotency check.
	if dup, err := o.store.EventAlreadyProcessed(ctx, order.EventId); err != nil {
		return false, err
	} else if dup {
		slog.Info("order already processed, skipping", "event_id", order.EventId)
		return true, nil
	}

	resp, err := o.store.Save(ctx, order)
	if err != nil {
		return false, err
	}

	pub := models.Billing{
		EventId:   uuid.New(),
		EventType: "SUCCESSFULLY_PAID",
		UserId:    resp.UserId,
		OrderId:   order.OrderId,
		Status:    "paid",
		Balance:   resp.Balance,
	}
	if err := o.responseWriter.ReportOrderUpdated(pub); err != nil {
		return false, err
	}
	if err := o.store.MarkEventProcessed(ctx, order.EventId, "SUCCESSFULLY_PAID"); err != nil {
		return false, err
	}
	return true, nil
}
