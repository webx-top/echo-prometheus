package main

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/webx-top/echo"
	echoPrometheus "github.com/webx-top/echo-prometheus"
	"github.com/webx-top/echo/engine/standard"
	"github.com/webx-top/echo/middleware"
)

func main() {
	e := echo.New()
	e.Use(middleware.Log())
	e.Use(echoPrometheus.MetricsMiddleware())
	e.Get("/metrics", echo.WrapHandler(promhttp.Handler()))

	e.Get("/", func(c echo.Context) error {
		return c.String("Hello, World!")
	})
	e.Logger().Fatal(e.Run(standard.New(":1323")))
}
