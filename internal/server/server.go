package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"billing-svc/internal/handlers/ver1"
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
func RunRouter(ctx context.Context, h *ver1.BillingHandler) {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestLogger())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := router.Group("/api")
	v1 := api.Group("/v1")
	//v1.GET("/billing", h.GetBillingHandler)
	//v1.PUT("/billing", h.AddMoneyHandler)
	//v1.POST("/billing/withdraw", h.WithdrawMoneyHandler)
	v1.POST("/order", h.AddMoneyHandler)

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
