package prometheus

import (
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_errors_total",
			Help: "Total HTTP 5xx responses",
		},
		[]string{"method", "path"},
	)
)

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		method := c.Request.Method
		path := normalizePath(c.Request.URL.Path)
		status := strconv.Itoa(c.Writer.Status())

		httpDuration.WithLabelValues(method, path, status).Observe(duration)
		httpRequestsTotal.WithLabelValues(method, path, status).Inc()

		if c.Writer.Status() >= 500 {
			httpErrorsTotal.WithLabelValues(method, path).Inc()
		}
	}
}

var (
	orderRe        = regexp.MustCompile(`^/api/v1/order$`)
	orderIDRe      = regexp.MustCompile(`^/api/v1/order/[^/]+$`)
	deliveryRe     = regexp.MustCompile(`^/api/v1/order/[^/]+/delivery$`)
	cancelRe       = regexp.MustCompile(`^/api/v1/order/[^/]+/cancel$`)
	payRe          = regexp.MustCompile(`^/api/v1/order/[^/]+/pay$`)
)

func normalizePath(path string) string {
	switch {
	case orderRe.MatchString(path):
		return "/api/v1/order"
	case deliveryRe.MatchString(path):
		return "/api/v1/order/{id}/delivery"
	case cancelRe.MatchString(path):
		return "/api/v1/order/{id}/cancel"
	case payRe.MatchString(path):
		return "/api/v1/order/{id}/pay"
	case orderIDRe.MatchString(path):
		return "/api/v1/order/{id}"
	}
	return path
}