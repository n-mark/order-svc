package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"order-svc/internal/handlers/ver1"
	"order-svc/internal/prometheus"
)

func newServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.Info("http",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"dur_ms", time.Since(start).Milliseconds(),
		)
	}
}

// RunRouter starts the HTTP API on $APP_PORT (default :8080) and blocks
// until ctx is cancelled.
func RunRouter(ctx context.Context, h *ver1.OrderHandler) {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestLogger())
	router.Use(prometheus.MetricsMiddleware())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	api := router.Group("/api")
	v1 := api.Group("/v1")
	v1.POST("/order", h.CreateOrderHandler)
	v1.GET("/order", h.ListOrdersHandler)
	v1.GET("/order/:id", h.GetOrderHandler)
	v1.PATCH("/order/:id/delivery", h.SetDeliveryHandler)
	v1.POST("/order/:id/cancel", h.CancelOrderHandler)
	v1.POST("/order/:id/pay", h.PayOrderHandler)

	srv := newServer(":"+port, router)

	go func() {
		<-ctx.Done()
		slog.Info("HTTP server shutting down")
		shutdownCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	slog.Info("HTTP server listening", "port", port)
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("HTTP server stopped", "err", err)
	}
}
