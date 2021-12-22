FROM 213288384225.dkr.ecr.us-west-1.amazonaws.com/golang:1.16

WORKDIR /go/src/app

# ARG DB_PASS

# ENV DB_PASS=$DB_PASS

COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

EXPOSE 8080

# CMD ["NEWSMUX_ENV=prod", "DB_PASS=$DB_PASS", "go", "run", "./cmd/server/main.go", "-dev=false", "-no_auth"]
