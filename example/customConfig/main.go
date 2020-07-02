package main

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/webx-top/echo"
	echoPrometheus "github.com/webx-top/echo-prometheus"
	"github.com/webx-top/echo/engine/standard"
)

func main() {
	e := echo.New()

	var configMetrics = echoPrometheus.NewConfig()
	configMetrics.Namespace = "namespace"
	configMetrics.NormalizeHTTPStatus = false
	configMetrics.Buckets = []float64{
		0.0005, // 0.5ms
		0.001,  // 1ms
		0.005,  // 5ms
		0.01,   // 10ms
		0.05,   // 50ms
		0.1,    // 100ms
		0.5,    // 500ms
		1,      // 1s
		2,      // 2s
	}

	e.Use(echoPrometheus.MetricsMiddlewareWithConfig(configMetrics))
	e.Get("/metrics", promhttp.Handler())

	e.Get("/", func(c echo.Context) error {
		return c.String("Hello, World!")
	})
	e.Logger().Fatal(e.Run(standard.New(":1323")))
}
