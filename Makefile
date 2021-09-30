SRC=$(shell find . -name "*.go")
.PHONY: fmt install_deps run_dev run_test run_prod

install_deps:
	$(info ******************** downloading dependencies ********************)
	go get -v ./...

run_devserver:
	$(info ******************** running dev api server ********************)
	NEWSMUX_ENV=dev go run ./cmd/server/main.go -dev=true -no_auth -service=api_server

run_prodserver:
	$(info ******************** running prod api server ********************)
	NEWSMUX_ENV=prod go run ./cmd/server/main.go -dev=false -no_auth -service=api_server

run_prodpublisher:
	$(info ******************** running prod publisher ********************)
	NEWSMUX_ENV=prod go run ./cmd/publisher/main.go -dev=false -service=feed_publisher

run_devpublisher:
	$(info ******************** running dev publisher ********************)
	NEWSMUX_ENV=dev go run ./cmd/publisher/main.go -dev=true -service=feed_publisher

run_prodcollector:
	$(info ******************** running prod collector ********************)
	NEWSMUX_ENV=dev go run ./cmd/collector/main.go -dev=true -service=collector

run_devcollector:
	$(info ******************** running dev collector ********************)
	NEWSMUX_ENV=prod go run ./cmd/collector/main.go -dev=false -service=collector

fmt:
	$(info ******************** checking formatting ********************)
	@test -z $(shell gofmt -l $(SRC)) || (gofmt -d $(SRC); exit 1)

test:
	NEWSMUX_ENV=test go test ./...
