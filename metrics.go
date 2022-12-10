package main

import (
	"context"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var cpuTemp = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "cpu_temperature_celsius",
	Help: "Current temperature of the CPU.",
})

func init() {
	prometheus.MustRegister(cpuTemp)
}

func PrometheusMetrics(ctx context.Context) echo.HandlerFunc {
	h := promhttp.Handler()
	cpuTemp.Set(21)
	return func(e echo.Context) error {
		h.ServeHTTP(e.Response().Writer, e.Request())
		return nil
	}
}
