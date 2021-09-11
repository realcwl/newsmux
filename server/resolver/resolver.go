package resolver

import (
	"gorm.io/gorm"
)

//go:generate go run github.com/99designs/gqlgen
//go:generate protoc --proto_path=$GOPATH/src/github.com/Luismorlan/newsmux/protocol/ --go_out=$GOPATH/src/github.com/Luismorlan/newsmux/protocol/ --go_opt=paths=source_relative $GOPATH/src/github.com/Luismorlan/newsmux/protocol/crawler_publisher_message.proto --experimental_allow_proto3_optional

const (
	DefaultSubSourceName = "default"
)

// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	DB             *gorm.DB
	SeedStateChans *SeedStateChannels
}
