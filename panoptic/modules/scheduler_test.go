package modules

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	TestConfig1 = `
		name: "cfg_1"
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
	TestConfig2 = `
		name: "cfg_2"
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
	TestConfig3 = `
		name: "cfg_3"
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
)

func TestUpsertJobs_AllNew(t *testing.T) {
	s := &Scheduler{
		m: sync.RWMutex{},
	}

	jobs := []*SchedulerJob{
		GetCustomizedSchedulerJob(t, TestConfig1),
		GetCustomizedSchedulerJob(t, TestConfig2),
		GetCustomizedSchedulerJob(t, TestConfig3),
	}
	s.UpsertJobs(jobs)

	assert.Equal(t, len(s.Jobs), 3)
	assert.Equal(t, s.Jobs[0].lastRun, time.Time{})
	assert.Equal(t, s.Jobs[2].panopticConfig.Name, "cfg_3")
}

func TestUpsertJobs_RemoveSome(t *testing.T) {
	s := &Scheduler{
		m: sync.RWMutex{},
		Jobs: []*SchedulerJob{
			GetCustomizedSchedulerJob(t, TestConfig1),
			GetCustomizedSchedulerJob(t, TestConfig2),
			GetCustomizedSchedulerJob(t, TestConfig3),
		},
	}

	jobs := []*SchedulerJob{
		GetCustomizedSchedulerJob(t, TestConfig1),
		GetCustomizedSchedulerJob(t, TestConfig3),
	}
	s.UpsertJobs(jobs)

	assert.Equal(t, len(s.Jobs), 2)
	assert.Equal(t, s.Jobs[0].panopticConfig.Name, "cfg_1")
	assert.Equal(t, s.Jobs[1].panopticConfig.Name, "cfg_3")
}

func TestUpsertJobs_UpdateOnlyConfig(t *testing.T) {
	s := &Scheduler{
		m: sync.RWMutex{},
		Jobs: []*SchedulerJob{
			GetCustomizedSchedulerJob(t, TestConfig1),
		},
	}

	now := time.Now()
	s.Jobs[0].lastRun = now
	s.Jobs[0].nextRun = now.Add(3 * time.Second)

	newSchedulerJob := GetCustomizedSchedulerJob(t, TestConfig1)
	newSchedulerJob.panopticConfig.
		TaskSchedule.GetRoutinely().EveryMilliseconds = 100

	jobs := []*SchedulerJob{newSchedulerJob}
	s.UpsertJobs(jobs)

	assert.Equal(t, len(s.Jobs), 1)
	assert.Equal(t, s.Jobs[0].lastRun, now)
	assert.Equal(t, s.Jobs[0].nextRun, now.Add(3*time.Second))
	assert.Equal(t, s.Jobs[0].panopticConfig.Name, "cfg_1")
	assert.Equal(t,
		s.Jobs[0].panopticConfig.GetTaskSchedule().
			GetRoutinely().EveryMilliseconds, int64(100))
}

func TestValidateJobs_DuplicateName(t *testing.T) {
	jobs := []*SchedulerJob{
		GetCustomizedSchedulerJob(t, TestConfig1),
		GetCustomizedSchedulerJob(t, TestConfig2),
		GetCustomizedSchedulerJob(t, TestConfig1),
	}
	assert.NotNil(t, ValidateJobs(jobs))
}
