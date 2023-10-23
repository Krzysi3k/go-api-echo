package metrics

import (
	"context"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

var (
	activeRequests = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "active_requests",
		Help: "The current number of active requests",
	})

	totalRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "total_requests",
		Help: "The total number of requests",
	})
)

func init() {
	prometheus.MustRegister(activeRequests)
	prometheus.MustRegister(totalRequests)
}

func RandomHandler(rdb *redis.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := context.Background()
		value := rand.Intn(100) + 1
		activeRequests.Set(float64(value))
		defer activeRequests.Dec()

		rdb.Set(ctx, "num1", value, 0).Result()
		totalRequests.Inc()

		return c.String(http.StatusOK, strconv.Itoa(value))
	}
}
