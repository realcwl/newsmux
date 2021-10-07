package model

import (
	"testing"

	"github.com/Luismorlan/newsmux/protocol"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestPanopticJobToDataCollectorRequest(t *testing.T) {
	expected := &protocol.PanopticJob{
		JobId: "job_id",
		Tasks: []*protocol.PanopticTask{{TaskId: "task_id"}},
	}

	req, err := PanopticJobToDataCollectorRequest(expected)
	assert.Nil(t, err)
	actual := &protocol.PanopticJob{}
	err = proto.Unmarshal(req.SerializedJob, actual)
	assert.Nil(t, err)

	assert.Equal(t, actual.JobId, "job_id")
	assert.Equal(t, actual.Tasks[0].TaskId, "task_id")
}

func TestDataCollectorResponseToPanopticJob(t *testing.T) {
	j := &protocol.PanopticJob{
		JobId: "job_id",
		Tasks: []*protocol.PanopticTask{{TaskId: "task_id"}},
	}
	req, err := PanopticJobToDataCollectorRequest(j)
	assert.Nil(t, err)
	res := &DataCollectorResponse{}
	res.SerializedJob = req.SerializedJob
	actual, err := DataCollectorResponseToPanopticJob(res)
	assert.Nil(t, err)

	assert.Equal(t, actual.JobId, "job_id")
	assert.Equal(t, actual.Tasks[0].TaskId, "task_id")
}

func TestPanopticJobToLambdaPayload(t *testing.T) {
	j := &protocol.PanopticJob{
		JobId: "job_id",
		Tasks: []*protocol.PanopticTask{{TaskId: "task_id"}},
	}
	payload, err := PanopticJobToLambdaPayload(j)
	assert.Nil(t, err)
	actual, err := LambdaPayloadToPanopticJob(payload)
	assert.Nil(t, err)

	assert.Equal(t, actual.JobId, "job_id")
	assert.Equal(t, actual.Tasks[0].TaskId, "task_id")
}
