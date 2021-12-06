package log

import (
	"os"
	"time"

	"github.com/Luismorlan/newsmux/utils/dotenv"
	"github.com/Luismorlan/newsmux/utils/flag"
	ddhook "github.com/bin3377/logrus-datadog-hook"
	"github.com/sirupsen/logrus"
)

const (
	datadogUSHost    = "http-intake.logs.datadoghq.com"
	syncFrequencySec = 30
	syncRetry        = 3
)

// global accessible logger
var (
	logger *logrus.Logger
	Log    *logrus.Entry
)

// This init function is only for testing cases, where the entry point is not
// main function. Unit test will fail with nil pointer dereference if we don't
// init here.
func init() {
	InitLogger()
}

func InitLogger() {
	logger = logrus.New()

	if os.Getenv("NEWSMUX_ENV") == dotenv.ProdEnv {
		apiKey := os.Getenv("DATADOG_API_KEY")
		hook := ddhook.NewHook(
			datadogUSHost,
			apiKey,
			syncFrequencySec*time.Second,
			syncRetry,
			logrus.InfoLevel,
			&logrus.JSONFormatter{},
			ddhook.Options{},
		)
		logger.Hooks.Add(hook)
	}

	// Also send log to stderr, without json formatter for better readability
	logger.SetOutput(os.Stderr)

	Log = logger.WithFields(
		logrus.Fields{"service": *flag.ServiceName, "is_development": os.Getenv("NEWSMUX_ENV") != dotenv.ProdEnv},
	)
}
