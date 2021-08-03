package utils

import (
	"flag"

	"github.com/sirupsen/logrus"
)

var IsDevelopment bool

func init() {
	// TODO(jamie): add more flags, able to overwrite envriment variables
	flag.BoolVar(&IsDevelopment, "dev", true, "set to true if the current run is for development. default value is true")
	flag.Parse()

	Logger.WithFields(logrus.Fields{"service": "api_server", "is_development": IsDevelopment}).Info("flags initialized")
}
