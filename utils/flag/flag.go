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
	Collector     = "collector"
)

var (
	ServiceName *string
	// if true, no authentication will be performed for the incoming request
	ByPassAuth *bool
)

// Example: go run cmd/publisher/main.go -service=feed_publisher -dev=true
func init() {
	// TODO: flag.Parse() in a package's init() won't work with golang's testing package, move to main
	// Issue https://github.com/golang/go/issues/31859
	// Temporary init testing before flag.Parse
	testing.Init()

	ServiceName = flag.String("service", APIServer, "'api_server', 'feed_publisher', 'collector', 'panoptic'")
	ByPassAuth = flag.Bool("no_auth", false, "set to true if local development")

	flag.Parse()
}

// Wrap flag.Parse in a helper function, so that main package importing this
// package will not get "imported but not used" error.
func ParseFlags() {
	flag.Parse()
}
