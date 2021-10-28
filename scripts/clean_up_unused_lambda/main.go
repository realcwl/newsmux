package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/Luismorlan/newsmux/panoptic"
	. "github.com/Luismorlan/newsmux/utils/log"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

const (
	STALE_TIME = 15 * time.Minute
)

func deleteLambdaFunctions(client *lambda.Client, ctx context.Context) int {
	count := 0
	res, err := client.ListFunctions(ctx, &lambda.ListFunctionsInput{})
	if err != nil {
		panic(err)
	}
	now := time.Now()
	for _, f := range res.Functions {
		if !strings.HasPrefix(*f.FunctionName, "data_collector_") {
			continue
		}
		lastModifiedTime, err := time.Parse("2006-01-02T15:04:05-0700", *f.LastModified)
		if err != nil {
			panic(err)
		}

		if now.Sub(lastModifiedTime) > STALE_TIME {
			_, err := client.DeleteFunction(ctx, &lambda.DeleteFunctionInput{
				FunctionName: f.FunctionName,
			})

			if err != nil {
				panic(err)
			}
			count++
			fmt.Println("function deleted, name:", *f.FunctionName, "Created at:", *f.LastModified)
		}
	}
	return count
}

func main() {
	flag.Parse()
	InitLogger()

	var client *lambda.Client
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(panoptic.AWS_REGION),
	)
	if err != nil {
		panic(err)
	}
	client = lambda.NewFromConfig(cfg)

	for deleteLambdaFunctions(client, ctx) > 0 {
		fmt.Println("== still have Lambda function to clean up ==")
	}
}
