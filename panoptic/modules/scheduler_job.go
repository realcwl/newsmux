package modules

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Luismorlan/newsmux/protocol"
)

// SchedulerJob defines the jobs which scheduler manages. Scheduler periodically
// transform those SchedulerJob into PanopticJobs, and send to event bus.
// It's worth noting that SchedulerJob and PanopticJob are not the same things,
// SchedulerJob defines how/when PanopticJob will be generated.
// This struct is thread-safe
type SchedulerJob struct {
	m sync.RWMutex

	// The last time this job is executed.
	lastRun time.Time

	// The next time this job should be executed.
	nextRun time.Time

	// General config about this SchedulerJob. Most notably, schedule is
	// included in this config.
	panopticConfig *protocol.PanopticConfig

	// The context of this job, which manages the lifecycle of this job.
	ctx context.Context

	// Cancel this Job and it's pending execution.
	cancel context.CancelFunc

	// How many times this job is scheduled on EventBus.
	runCount int64
}

func NewSchedulerJobs(configs *protocol.PanopticConfigs, ctx context.Context) []*SchedulerJob {
	jobs := []*SchedulerJob{}
	for _, config := range configs.Config {
		jobs = append(jobs, NewSchedulerJob(config, ctx))
	}
	return jobs
}

func NewSchedulerJob(config *protocol.PanopticConfig, ctx context.Context) *SchedulerJob {
	ctx, cancel := context.WithCancel(ctx)
	return &SchedulerJob{
		m:              sync.RWMutex{},
		lastRun:        time.Time{},
		nextRun:        time.Time{},
		panopticConfig: config,
		ctx:            ctx,
		cancel:         cancel,
		runCount:       0,
	}
}

func (j *SchedulerJob) HasRunBefore() bool {
	j.m.RLock()
	defer j.m.RUnlock()

	return !j.lastRun.IsZero()
}

func (j *SchedulerJob) IncrementRunCount() {
	j.m.Lock()
	defer j.m.Unlock()
	j.runCount += 1
}

func (j *SchedulerJob) DurationTillNextRun() time.Duration {
	duration, _ := j.CalculateInterval()
	if !j.HasRunBefore() {
		return duration
	}

	j.m.RLock()
	defer j.m.RUnlock()

	now := time.Now()
	return j.nextRun.Sub(now)
}

func (j *SchedulerJob) UpdateLastAndNextTime() error {
	duration, err := j.CalculateInterval()

	j.m.Lock()
	defer j.m.Unlock()

	j.lastRun = time.Now()
	if err != nil {
		return err
	}
	j.nextRun = j.lastRun.Add(duration)
	return nil
}

func (j *SchedulerJob) CalculateInterval() (time.Duration, error) {
	j.m.RLock()
	defer j.m.RUnlock()

	switch scheduleType := j.panopticConfig.TaskSchedule.Schedule.(type) {
	case *protocol.TaskSchedule_Routinely:
		return time.Duration(
			j.panopticConfig.TaskSchedule.GetRoutinely().DurationMilliseconds) *
			time.Millisecond, nil
	default:
		return 0, fmt.Errorf("unknown schedule type: %T", scheduleType)
	}
}
