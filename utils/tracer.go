package utils

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func init() {
	// Datadog tracer

	env := "development"
	if !IsDevelopment {
		env = "production"
	}

	tracer.Start(
		tracer.WithService(ServiceName),
		tracer.WithEnv(env),
	)

	Logger.WithFields(
		logrus.Fields{"service": ServiceName, "is_development": IsDevelopment},
	).Info("tracer initialized")
}

// Stop tracer, OK to be closed multiple times
func CloseTracer() {
	// Datadog tracer
	tracer.Stop()
}
