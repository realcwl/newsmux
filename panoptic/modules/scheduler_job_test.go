package modules

import (
	"context"
	"testing"
	"time"

	"github.com/Luismorlan/newsmux/protocol"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/testing/protocmp"
)

const DefaultPanopticConfig = `
	name: "Jinshi Kuaixun"
	data_collector_id: COLLECTOR_JINSHI
	task_params: {
		source_id: "dummy_source_id"
	}
	task_schedule: {
		start_immediatly: true
		routinely: {
			every_milliseconds: 1000
		}
	}
`

func GetDefaultSchedulerJob(t *testing.T) *SchedulerJob {
	return GetCustomizedSchedulerJob(t, DefaultPanopticConfig)
}

func GetCustomizedSchedulerJob(t *testing.T, s string) *SchedulerJob {
	config := protocol.PanopticConfig{}

	assert.Nil(t, prototext.Unmarshal([]byte(s), &config))

	ctx := context.Background()
	return NewSchedulerJob(&config, ctx)
}

func TestNewSchedulerJob(t *testing.T) {
	config := protocol.PanopticConfig{}

	assert.Nil(t, prototext.Unmarshal([]byte(DefaultPanopticConfig), &config))

	ctx := context.Background()
	actual := NewSchedulerJob(&config, ctx)

	opts := []cmp.Option{
		protocmp.Transform(),
	}
	// Config is copied to SchedulerJob
	assert.Empty(t, cmp.Diff(actual.panopticConfig, &config, opts...))

	// Both time is initialized to 0
	assert.Equal(t, actual.lastRun, time.Time{})
	assert.Equal(t, actual.nextRun, time.Time{})

	// runCount is initialized to 0
	assert.Equal(t, actual.runCount, int64(0))
}

func TestHasRunBefore(t *testing.T) {
	job := GetDefaultSchedulerJob(t)
	// runCount is initialized to 0
	assert.Equal(t, job.runCount, int64(0))
	// increment to 1
	job.IncrementRunCount()
	assert.Equal(t, job.runCount, int64(1))
	// increment to 3
	job.IncrementRunCount()
	job.IncrementRunCount()
	assert.Equal(t, job.runCount, int64(3))
}

func TestCalculateInterval(t *testing.T) {
	job := GetDefaultSchedulerJob(t)
	duration, err := job.CalculateInterval()
	assert.Nil(t, err)
	assert.Equal(t, duration, 1000*time.Millisecond)
}

func TestUpdateLastAndNextTime(t *testing.T) {
	job := GetDefaultSchedulerJob(t)
	now := time.Now()
	job.UpdateLastAndNextTime()

	// Within reasonable time.
	diff := job.lastRun.Sub(now)
	assert.True(t, diff > -2*time.Second && diff < 2*time.Second)

	interval := job.nextRun.Sub(job.lastRun)
	assert.Equal(t, interval, 1*time.Second)
}

func TestDurationTillNextRun(t *testing.T) {
	job := GetDefaultSchedulerJob(t)
	// should return interval because job is not executed
	duration := job.DurationTillNextRun()
	assert.Equal(t, duration, 1*time.Second)

	job.UpdateLastAndNextTime()
	duration = job.DurationTillNextRun()
	assert.Less(t, duration, 1*time.Second)
}

func TestMaybeSplitIntoMultipleSchedulerJobs(t *testing.T) {
	c := protocol.PanopticConfig{
		Name: "test",
		TaskParams: &protocol.TaskParams{
			SubSources: []*protocol.PanopticSubSource{
				{Name: "ss_1"},
				{Name: "ss_2"},
				{Name: "ss_3"},
				{Name: "ss_4"},
				{Name: "ss_5"},
			},
			MaxSubsourcePerTask: 2,
		},
	}

	ctx := context.TODO()

	jobs := MaybeSplitIntoMultipleSchedulerJobs(&c, ctx)
	assert.Equal(t, len(jobs), 3)
	assert.Equal(t, jobs[0].panopticConfig.TaskParams.SubSources[0].Name, "ss_1")
	assert.Equal(t, jobs[0].panopticConfig.TaskParams.SubSources[1].Name, "ss_2")
	assert.Equal(t, jobs[1].panopticConfig.TaskParams.SubSources[0].Name, "ss_3")
	assert.Equal(t, jobs[1].panopticConfig.TaskParams.SubSources[1].Name, "ss_4")
	assert.Equal(t, jobs[2].panopticConfig.TaskParams.SubSources[0].Name, "ss_5")
}

func TestMaybeSplitIntoMultipleSchedulerJobs_UnsetShouldReturnOnlyOneJob(t *testing.T) {
	c := protocol.PanopticConfig{
		Name: "test",
		TaskParams: &protocol.TaskParams{
			SubSources: []*protocol.PanopticSubSource{
				{Name: "ss_1"},
				{Name: "ss_2"},
				{Name: "ss_3"},
				{Name: "ss_4"},
				{Name: "ss_5"},
			},
		},
	}

	ctx := context.TODO()

	jobs := MaybeSplitIntoMultipleSchedulerJobs(&c, ctx)
	assert.Equal(t, len(jobs), 1)
	assert.Equal(t, jobs[0].panopticConfig.TaskParams.SubSources[0].Name, "ss_1")
	assert.Equal(t, jobs[0].panopticConfig.TaskParams.SubSources[1].Name, "ss_2")
	assert.Equal(t, jobs[0].panopticConfig.TaskParams.SubSources[2].Name, "ss_3")
	assert.Equal(t, jobs[0].panopticConfig.TaskParams.SubSources[3].Name, "ss_4")
	assert.Equal(t, jobs[0].panopticConfig.TaskParams.SubSources[4].Name, "ss_5")
}
