SRC=$(shell find . -name "*.go")
.PHONY: fmt install_deps run_dev run_test run_prod

install_deps:
	$(info ******************** downloading dependencies ********************)
	go get -v ./...

run_devserver:
	$(info ******************** running development server ********************)
	NEWSMUX_ENV=dev go run ./cmd/server/main.go -dev=true 

run_prodserver:
	$(info ******************** running production server ********************)
	NEWSMUX_ENV=prod go run ./cmd/server/main.go -dev=false 

fmt:
	$(info ******************** checking formatting ********************)
	@test -z $(shell gofmt -l $(SRC)) || (gofmt -d $(SRC); exit 1)

test:
	NEWSMUX_ENV=test go test ./...