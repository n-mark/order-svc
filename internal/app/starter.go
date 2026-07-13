package app

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"billing-svc/internal/config"
	"billing-svc/internal/handlers/ver1"
	"billing-svc/internal/messaging"
	"billing-svc/internal/server"
	"billing-svc/internal/service"
	"billing-svc/internal/store"
)

func RunCommonServer() {
	cfg := config.InitConfig()

	// Root context cancelled on SIGINT / SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- Postgres ---
	pgCfg := config.GetPGConfig()
	pool, err := connectPG(ctx, pgCfg.DSN())
	if err != nil {
		slog.Error("failed to connect to postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	pgStore := store.NewPgStore(pool)

	// --- Broker ---
	b, err := messaging.InitBroker(*cfg)
	if err != nil {
		slog.Error("failed to init broker", "err", err)
		os.Exit(1)
	}

	orderSvc := service.NewOrderService(pgStore, b)
	h := ver1.NewOrderHandler(*orderSvc)

	b.RegisterConsumer(b.GetBillingPaymentDataSourceName(), h.UpdateOrderAfterBillingResponse)

	// HTTP and AMQP run in parallel; the AMQP Run() blocks until ctx is done
	// (when it returns we shut down HTTP via the same ctx).
	go server.RunRouter(ctx, h)

	slog.Info("starting broker loop")
	b.Run()
	slog.Info("broker loop exited")
}

func connectPG(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(dialCtx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(dialCtx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
