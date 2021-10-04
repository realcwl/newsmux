package main

import (
	"context"
	"fmt"

	"github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/protocol"
	. "github.com/Luismorlan/newsmux/utils"
	"github.com/Luismorlan/newsmux/utils/dotenv"
	. "github.com/Luismorlan/newsmux/utils/log"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/golang/protobuf/proto"
)

func init() {
	Log.Info("data collector initialized")
}

func cleanup() {
	CloseProfiler()
	CloseTracer()
	Log.Info("data collector shutdown")
}

type DataCollectorRequest struct {
	SerializedJob []byte
}

type DataCollectorResponse struct {
	SerializedJob []byte
}

func HandleRequest(ctx context.Context, event DataCollectorRequest) (resp DataCollectorResponse, e error) {
	// parse job
	job := &protocol.PanopticJob{}
	if err := proto.Unmarshal(event.SerializedJob, job); err != nil {
		Log.Error("Failed to parse job with error:", err)
		return resp, err
	}
	Log.Info("Processing job with job id : ", job.JobId)

	// handle
	var handler collector.DataCollectServerHandler
	res, err := handler.Collect(ctx, job)
	if err != nil {
		Log.Error("Failed to execute job with error:", err)
		return resp, err
	}
	// encode job
	str, err := proto.Marshal(res)
	resp.SerializedJob = str
	return resp, nil
}

func main() {
	defer cleanup()
	if err := dotenv.LoadDotEnvs(); err != nil {
		panic(err)
	}
	lambda.Start(HandleRequest)
}
