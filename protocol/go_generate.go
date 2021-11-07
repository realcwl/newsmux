package protocol

// the command is run at current /protocol folder

// Panoptic
//go:generate protoc --go_out=. --go_opt=paths=source_relative panoptic_config.proto --experimental_allow_proto3_optional
//go:generate protoc --go_out=. --go_opt=paths=source_relative panoptic.proto --experimental_allow_proto3_optional

// Crawler Publisher Message
//go:generate protoc --go_out=. --go_opt=paths=source_relative crawler_publisher_message.proto --experimental_allow_proto3_optional
//go:generate protoc --python_out=. crawler_publisher_message.proto --experimental_allow_proto3_optional

// Deduplicator
//go:generate python -m grpc_tools.protoc -I. --python_out=../deduplicator --grpc_python_out=../deduplicator deduplicator.proto --experimental_allow_proto3_optional
//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative deduplicator.proto
