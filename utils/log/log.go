package log

import (
	"os"
	"time"

	"github.com/Luismorlan/newsmux/utils/flag"
	ddhook "github.com/bin3377/logrus-datadog-hook"
	"github.com/sirupsen/logrus"
)

const (
	dd_us_host         = "http-intake.logs.datadoghq.com"
	apiKey             = "4ff818baf9436137bfdde74914f3bdba"
	sync_frequency_sec = 30
	sync_retry         = 3
)

// global accessible logger
var logger = logrus.New()
var Log *logrus.Entry = nil

func init() {
	initLogger()
	Log = logger.WithFields(
		logrus.Fields{"service": flag.ServiceName, "is_development": flag.IsDevelopment},
	)
}

func initLogger() {
	hook := ddhook.NewHook(
		dd_us_host,
		apiKey,
		sync_frequency_sec*time.Second,
		sync_retry,
		logrus.TraceLevel,
		&logrus.JSONFormatter{},
		ddhook.Options{},
	)
	logger.Hooks.Add(hook)

	// Also send log to stderr, without json formatter for better readability
	logger.SetOutput(os.Stderr)

	// PARKING LOT:
	// Write logs both to stderr and rotational file log (which will be picked up by DataDog)
	// dd_local_log := lumberjack.Logger{
	// 	Filename:   "/Users/jamie/go/src/github.com/Luismorlan/newsmux/log.log",
	// 	MaxSize:    1,     // megabytes
	// 	MaxBackups: 3,     // number of backups to keep
	// 	MaxAge:     28,    // days
	// 	Compress:   false, // disabled by default
	// }
	// writers := io.MultiWriter(os.Stderr, dd_local_log)
	// Logger.SetOutput(io.MultiWriter(writers))
}
