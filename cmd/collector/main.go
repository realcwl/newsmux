package main

import (
	ddlambda "github.com/DataDog/datadog-lambda-go"
	"github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/model"
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

func HandleRequest(event model.DataCollectorRequest) (model.DataCollectorResponse, error) {
	res := model.DataCollectorResponse{}

	// parse job
	job := &protocol.PanopticJob{}
	Log.Info("Raw serialized job : ", event.SerializedJob)
	if err := proto.Unmarshal(event.SerializedJob, job); err != nil {
		Log.Error("Failed to parse job with error:", err)
		return res, err
	}
	Log.Info("Processing job with job id : ", job.JobId)

	// handle
	var handler collector.DataCollectJobHandler
	err := handler.Collect(job)
	if err != nil {
		Log.Error("Failed to execute job with error:", err)
		return res, err
	}
	// encode job
	bytes, err := proto.Marshal(job)
	if err != nil {
		return res, err
	}

	res.SerializedJob = bytes
	return res, nil
}

func main() {
	defer cleanup()
	if err := dotenv.LoadDotEnvs(); err != nil {
		panic(err)
	}
	Log.Info("Starting lambda handler, waiting for requests...")

	lambda.Start(ddlambda.WrapFunction(HandleRequest, nil))
}
