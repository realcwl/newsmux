package utils

import (
	"flag"

	"github.com/sirupsen/logrus"
)

const (
	APIServer     = "api_server"
	FeedPublisher = "feed_publisher"
)

var IsDevelopment bool
var ServiceName string

func init() {
	// TODO(jamie): add more flags, able to overwrite envriment variables
	flag.BoolVar(&IsDevelopment, "dev", true, "set to true if the current run is for development. default value is true")
	flag.StringVar(&ServiceName, "service", APIServer, "'api_server' or 'feed_publisher'")
	flag.Parse()

	Logger.WithFields(
		logrus.Fields{"service": ServiceName, "is_development": IsDevelopment},
	).Info("flags initialized")
}
