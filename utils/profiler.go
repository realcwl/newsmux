package utils

import (
	. "github.com/Luismorlan/newsmux/utils/flag"
	. "github.com/Luismorlan/newsmux/utils/log"
	"gopkg.in/DataDog/dd-trace-go.v1/profiler"
)

func init() {
	// Datadog profiler

	env := "development"
	if IsDevelopment {
		env = "production"
	}

	if err := profiler.Start(
		profiler.WithService(ServiceName),
		profiler.WithEnv(env),
		profiler.WithProfileTypes(
			profiler.CPUProfile,
			profiler.HeapProfile,
			// The profiles below are disabled by
			// default to keep overhead low, but
			// can be enabled as needed.
			// profiler.BlockProfile,
			// profiler.MutexProfile,
			// profiler.GoroutineProfile,
		),
	); err != nil {
		Log.Fatal(err)
	}

	Log.Info("profiler initialized")

}

// Stop profiler, OK to be closed multiple times
func CloseProfiler() {
	// Datadog profiler
	profiler.Stop()
}
