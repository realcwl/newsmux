/*
flag Package set up cli flags shared across services

Usage:

	Flags listed in this package are shared across boundaries and service-agnostic
	For service dependent flags please define in their respective package

TODO(jamie): move to more powerful cli lib https://github.com/spf13/cobra
*/

package flag

import (
	"flag"
	"testing"
)

const (
	APIServer     = "api_server"
	FeedPublisher = "feed_publisher"
)

var (
	IsDevelopment bool
	ServiceName   string
	// if true, no authentication will be performed for the incoming request
	ByPassAuth bool
)

// Example: go run cmd/publisher/main.go -service=feed_publisher -dev=true
func init() {
	// TODO: flag.Parse() in a package's init() won't work with golang's testing package, move to main
	// Issue https://github.com/golang/go/issues/31859
	// Temporary init testing before flag.Parse
	testing.Init()

	flag.BoolVar(&IsDevelopment, "dev", true, "set to true if the current run is for development. default value is true")
	flag.StringVar(&ServiceName, "service", APIServer, "'api_server' or 'feed_publisher'")
	flag.BoolVar(&ByPassAuth, "no_auth", false, "set to true if local development")
	flag.Parse()
}
