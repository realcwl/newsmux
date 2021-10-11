package model

import (
	"encoding/json"

	"github.com/Luismorlan/newsmux/protocol"
	"google.golang.org/protobuf/proto"
)

type DataCollectorRequest struct {
	// Serialized PanopticJob sending to Lambda
	SerializedJob []byte
}

type DataCollectorResponse struct {
	// Serialized PanopticJob returning from Lambda
	SerializedJob []byte
}

func PanopticJobToDataCollectorRequest(job *protocol.PanopticJob) (*DataCollectorRequest, error) {
	marshal, err := proto.Marshal(job)
	if err != nil {
		return nil, err
	}

	req := &DataCollectorRequest{
		SerializedJob: marshal,
	}
	return req, nil
}

func DataCollectorResponseToPanopticJob(res *DataCollectorResponse) (*protocol.PanopticJob, error) {
	resJob := &protocol.PanopticJob{}
	err := proto.Unmarshal(res.SerializedJob, resJob)
	return resJob, err
}

func PanopticJobToLambdaPayload(job *protocol.PanopticJob) ([]byte, error) {
	req, err := PanopticJobToDataCollectorRequest(job)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func LambdaPayloadToPanopticJob(payload []byte) (*protocol.PanopticJob, error) {
	var res DataCollectorResponse
	err := json.Unmarshal(payload, &res)
	if err != nil {
		return nil, err
	}
	return DataCollectorResponseToPanopticJob(&res)
}
