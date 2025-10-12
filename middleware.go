package echoprometheus

import (
	"reflect"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/webx-top/echo"
	"github.com/webx-top/echo/engine"
)

// Config responsible to configure middleware
type Config struct {
	Skipper             echo.Skipper
	Namespace           string
	Buckets             []float64
	Subsystem           string
	NormalizeHTTPStatus bool
	OnlyRoutePath       bool
}

const (
	httpRequestsCount    = "requests_total"
	httpRequestsDuration = "request_duration_seconds"
	httpRequestSize      = "request_size_bytes"
	httpResponseSize     = "response_size_bytes"
	notFoundPath         = "/not-found"
)

// DefaultConfig has the default instrumentation config
var DefaultConfig = Config{
	Skipper: func(c echo.Context) bool {
		return c.Request().URL().Path() == c.Echo().Prefix()+`/metrics`
	},
	Namespace: "echo",
	Subsystem: "http",
	Buckets: []float64{
		0.0005,
		0.001, // 1ms
		0.002,
		0.005,
		0.01, // 10ms
		0.02,
		0.05,
		0.1, // 100 ms
		0.2,
		0.5,
		1.0, // 1s
		2.0,
		5.0,
		10.0, // 10s
		15.0,
		20.0,
		30.0,
	},
	NormalizeHTTPStatus: true,
	OnlyRoutePath:       true,
}

func normalizeHTTPStatus(status int) string {
	if status < 200 {
		return "1xx"
	} else if status < 300 {
		return "2xx"
	} else if status < 400 {
		return "3xx"
	} else if status < 500 {
		return "4xx"
	}
	return "5xx"
}

var notFoundHandlerPointer = reflect.ValueOf(echo.NotFoundHandler).Pointer()

func isNotFoundHandler(handler echo.Handler) bool {
	return reflect.ValueOf(handler).Pointer() == notFoundHandlerPointer
}

// NewConfig returns a new config with default values
func NewConfig() Config {
	return DefaultConfig
}

// MetricsMiddleware returns an echo middleware with default config for instrumentation.
func MetricsMiddleware() echo.MiddlewareFuncd {
	return MetricsMiddlewareWithConfig(DefaultConfig)
}

// MetricsMiddlewareWithConfig returns an echo middleware for instrumentation.
func MetricsMiddlewareWithConfig(config Config) echo.MiddlewareFuncd {
	httpRequests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: config.Namespace,
		Subsystem: config.Subsystem,
		Name:      httpRequestsCount,
		Help:      "Number of HTTP operations",
	}, []string{"status", "method", "handler"})
	prometheus.DefaultRegisterer.Unregister(httpRequests)
	prometheus.DefaultRegisterer.MustRegister(httpRequests)

	httpReqSize := prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace: config.Namespace,
		Subsystem: config.Subsystem,
		Name:      httpRequestSize,
		Help:      "Number of bytes in the HTTP request",
	})
	prometheus.DefaultRegisterer.Unregister(httpReqSize)
	prometheus.DefaultRegisterer.MustRegister(httpReqSize)

	httpRespSize := prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace: config.Namespace,
		Subsystem: config.Subsystem,
		Name:      httpResponseSize,
		Help:      "Number of bytes in the HTTP response",
	})
	prometheus.DefaultRegisterer.Unregister(httpRespSize)
	prometheus.DefaultRegisterer.MustRegister(httpRespSize)

	httpDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: config.Namespace,
		Subsystem: config.Subsystem,
		Name:      httpRequestsDuration,
		Help:      "Spend time by processing a route",
		Buckets:   config.Buckets,
	}, []string{"method", "handler"})
	prometheus.DefaultRegisterer.Unregister(httpDuration)
	prometheus.DefaultRegisterer.MustRegister(httpDuration)

	skipper := config.Skipper
	if skipper == nil {
		skipper = echo.DefaultSkipper
	}

	return func(next echo.Handler) echo.HandlerFunc {
		return func(c echo.Context) error {
			if skipper(c) {
				return next.Handle(c)
			}
			req := c.Request()
			var path string

			// to avoid attack high cardinality of 404
			if isNotFoundHandler(next) {
				path = notFoundPath
			}

			if len(path) == 0 {
				if config.OnlyRoutePath {
					path = c.Path()
				} else {
					path = req.URL().Path()
				}
			}

			//c.Logger().Debug(path)

			if httpReqSize != nil {
				httpReqSize.Observe(float64(computeApproximateRequestSize(req)))
			}

			var timer *prometheus.Timer
			if httpDuration != nil {
				timer = prometheus.NewTimer(httpDuration.WithLabelValues(req.Method(), path))
			}

			err := next.Handle(c)

			if timer != nil {
				timer.ObserveDuration()
			}
			if httpRespSize != nil {
				httpRespSize.Observe(float64(c.Response().Size()))
			}

			if err != nil {
				c.Error(err)
			}

			if httpRequests != nil {
				var status string
				if config.NormalizeHTTPStatus {
					status = normalizeHTTPStatus(c.Response().Status())
				} else {
					status = strconv.Itoa(c.Response().Status())
				}

				httpRequests.WithLabelValues(status, req.Method(), path).Inc()
			}
			return err
		}
	}
}

func computeApproximateRequestSize(r engine.Request) int {
	s := len(r.URL().String()) + len(r.Method()) + len(r.Proto())
	for name, values := range r.Header().Std() {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}
	s += len(r.Host())

	// N.B. r.Form and r.MultipartForm are assumed to be included in r.URL.

	if r.Size() != -1 {
		s += int(r.Size())
	}
	return s
}
