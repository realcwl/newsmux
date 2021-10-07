package modules

import (
	"testing"
	"time"

	"github.com/Luismorlan/newsmux/protocol"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/stretchr/testify/assert"
)

func TestGetLambdaLifeSpanWithRandomness(t *testing.T) {
	span := GetLambdaLifeSpanWithRandomness(1000)
	assert.GreaterOrEqual(t, span, time.Duration(750*time.Second))
	assert.LessOrEqual(t, span, time.Duration(1250*time.Second))
}

func TestLambdaFunctionAddPendingJob(t *testing.T) {
	now := "2021-10-02T23:28:38.534+0000"
	name := "test"
	f, err := NewLambdaFunction(&lambda.CreateFunctionOutput{
		FunctionName: &name,
		LastModified: &now,
	}, time.Duration(10*time.Second))

	assert.Nil(t, err)
	assert.Equal(t, f.name, name)
}

func TestFunctionAddDeleteJob(t *testing.T) {
	now := "2021-10-02T23:28:38.534+0000"
	name := "test"
	f, err := NewLambdaFunction(&lambda.CreateFunctionOutput{
		FunctionName: &name,
		LastModified: &now,
	}, time.Duration(10*time.Second))

	assert.Nil(t, err)

	assert.True(t, f.IsRemovable())
	f.AddPendingJob(&protocol.PanopticJob{JobId: "1"})
	assert.True(t, len(f.jobs) == 1)
	f.DeletePendingJob(&protocol.PanopticJob{JobId: "1"})
	assert.True(t, len(f.jobs) == 0)
}
