package utils

import (
	. "github.com/Luismorlan/newsmux/utils/flag"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func init() {
	// Datadog tracer

	env := "development"
	if IsProdEnv() {
		env = "production"
	}

	tracer.Start(
		tracer.WithService(ServiceName),
		tracer.WithEnv(env),
	)
}

// Stop tracer, OK to be closed multiple times
func CloseTracer() {
	// Datadog tracer
	tracer.Stop()
}
