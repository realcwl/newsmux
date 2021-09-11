FROM golang:1.16

WORKDIR /go/src/app

COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

EXPOSE 8080

RUN NEWSMUX_ENV=prod go run ./cmd/server/main.go -dev=false -no_auth 
