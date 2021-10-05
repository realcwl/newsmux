SRC=$(shell find . -name "*.go")
.PHONY: fmt install_deps run_dev run_test run_prod

IMAGE_NAME = data_collector
ECR_ARN = 213288384225.dkr.ecr.us-west-1.amazonaws.com/data_collector

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

run_collector_lambda_locally:
	$(info ******************** running dev collector ********************)
	docker build -t $(IMAGE_NAME) -f cmd/collector/Dockerfile .
	docker run --env _LAMBDA_SERVER_PORT=9000 --env AWS_LAMBDA_RUNTIME_API=localhost --env NEWSMUX_ENV=dev -p 9000:8080 $(IMAGE_NAME)

build_collector_and_push_image:
	$(info ******************** building and push collector image to ECR ********************)
	aws ecr get-login-password --region us-west-1 | docker login --username AWS --password-stdin $(ECR_ARN)
	docker build -t $(IMAGE_NAME) -f cmd/collector/Dockerfile .
	docker tag $(IMAGE_NAME):latest $(ECR_ARN):latest
	docker push $(ECR_ARN):latest

fmt:
	$(info ******************** checking formatting ********************)
	@test -z $(shell gofmt -l $(SRC)) || (gofmt -d $(SRC); exit 1)

test:
	NEWSMUX_ENV=test go test ./...
