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
	apiKey           = "4ff818baf9436137bfdde74914f3bdba"
	syncFrequencySec = 30
	syncRetry        = 3
)

// global accessible logger
var (
	logger *logrus.Logger
	Log    *logrus.Entry
)

func init() {
	initLogger()
}

func initLogger() {
	logger = logrus.New()

	hook := ddhook.NewHook(
		datadogUSHost,
		apiKey,
		syncFrequencySec*time.Second,
		syncRetry,
		logrus.TraceLevel,
		&logrus.JSONFormatter{},
		ddhook.Options{},
	)
	logger.Hooks.Add(hook)

	// Also send log to stderr, without json formatter for better readability
	logger.SetOutput(os.Stderr)

	Log = logger.WithFields(
		logrus.Fields{"service": flag.ServiceName, "is_development": os.Getenv("NEWSMUX_ENV") != dotenv.ProdEnv},
	)
}
