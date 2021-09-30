package protocol

// the command is run at current /protocol folder

// Panoptic
//go:generate protoc --go_out=. --go_opt=paths=source_relative panoptic.proto --experimental_allow_proto3_optional

// Crawler Publisher Message
//go:generate protoc --go_out=. --go_opt=paths=source_relative crawler_publisher_message.proto --experimental_allow_proto3_optional
//go:generate protoc --python_out=. crawler_publisher_message.proto --experimental_allow_proto3_optional

//go:generate protoc --proto_path=$GOPATH/src/github.com/Luismorlan/newsmux/protocol/ --go_out=$GOPATH/src/github.com/Luismorlan/newsmux/protocol/ --go_opt=paths=source_relative $GOPATH/src/github.com/Luismorlan/newsmux/protocol/crawler_publisher_message.proto --experimental_allow_proto3_optional
//go:generate protoc --python_out=. crawler_publisher_message.proto --experimental_allow_proto3_optional
