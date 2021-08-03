package utils

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func init() {
	// Datadog tracer
	tracer.Start(
		tracer.WithService("apiserver"),
		tracer.WithEnv("development"),
	)

	Logger.WithFields(logrus.Fields{"service": "api_server", "is_development": IsDevelopment}).Info("tracer initialized")
}

// Stop tracer, OK to be closed multiple times
func CloseTracer() {
	// Datadog tracer
	tracer.Stop()
}
