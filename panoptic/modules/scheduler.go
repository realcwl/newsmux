package modules

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/encoding/prototext"
	"gorm.io/gorm"

	"github.com/Luismorlan/newsmux/app_setting"
	"github.com/Luismorlan/newsmux/panoptic"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
)

var AppSetting *app_setting.PanopticAppSetting

// A valid job batch must not contains duplicate job name.
func ValidateJobs(jobs []*SchedulerJob) error {
	seen := make(map[string]bool)
	for _, job := range jobs {
		if _, ok := seen[job.panopticConfig.Name]; ok {
			return fmt.Errorf("duplicate scheduler job name: %s", job.panopticConfig.Name)
		}
		seen[job.panopticConfig.Name] = true
	}
	return nil
}

type SchedulerConfig struct {
	// Name of the scheduler.
	Name string
}

type Scheduler struct {
	m sync.RWMutex

	// Config for this scheduler.
	Config SchedulerConfig

	// Hashing of the config's digest
	ScheduleDigest string

	// Context of this Scheduler.
	ctx context.Context

	// A list of SchedulerJobs that this scheduler is managing.
	Jobs []*SchedulerJob

	// Whether this scheduler is running.
	running bool

	// JobDoer is the actual component that executes a job. In our case it's
	// mostly emitting PanopticJob.
	Doer JobDoer

	EventBus *gochannel.GoChannel

	DB *gorm.DB
}

// Return a new instance of Scheduler.
func NewScheduler(
	panopticAppSetting *app_setting.PanopticAppSetting, config SchedulerConfig,
	e *gochannel.GoChannel, doer JobDoer, ctx context.Context) *Scheduler {
	AppSetting = panopticAppSetting

	db, err := utils.GetDBConnection()
	if err != nil {
		Logger.Log.Errorln("failed to connect to database")
	}

	scheduler := &Scheduler{
		Config:         config,
		ctx:            ctx,
		EventBus:       e,
		ScheduleDigest: "",
		Doer:           doer,
		running:        false,
		DB:             db,
	}
	return scheduler
}

func (s *Scheduler) UpdateConfigDigest(digest string) {
	s.m.Lock()
	defer s.m.Unlock()

	s.ScheduleDigest = digest
}

// For existing jobs, only job's PanopticConfig is updated. Otherwise remove
// from the job list. If the job is already in pending state, cancel it
// proactively. For all new jobs, append to the end of job lists.
func (s *Scheduler) UpsertJobs(jobs []*SchedulerJob) {
	s.m.Lock()
	defer s.m.Unlock()

	nameToJobMap := make(map[string]*SchedulerJob)

	// Index all jobs by it's config name.
	for idx := range jobs {
		nameToJobMap[jobs[idx].panopticConfig.Name] = jobs[idx]
	}

	// Existing Jobs.
	idx := 0
	for idx < len(s.Jobs) {
		existingJob := s.Jobs[idx]
		if v, ok := nameToJobMap[existingJob.panopticConfig.Name]; ok {
			// Existing job found. Update it's PanopticConfig. Also delete it from
			// nameToJobMap.
			delete(nameToJobMap, existingJob.panopticConfig.Name)
			existingJob.panopticConfig = v.panopticConfig
			idx += 1
		} else {
			// Existing job not found. Remove it from the job list.
			s.Jobs = append(s.Jobs[:idx], s.Jobs[idx+1:]...)
			existingJob.cancel()
		}
	}

	// New Jobs. Append to the end of the job list.
	for _, v := range nameToJobMap {
		s.Jobs = append(s.Jobs, v)
	}
}

// Read config either from local workspace (dev) or from Github (production)
// In addition to the config, we read from DB and add more subsources to each source in the configs
func (s *Scheduler) ReadConfig() (*protocol.PanopticConfigs, string, error) {
	configs, err := s.ReadConfigFromLocalOrGithub()
	if err != nil {
		return nil, "", err
	}

	panoptic.MergeSubsourcesFromConfigAndDb(s.DB, configs)

	digest, err := utils.TextToMd5Hash(configs.String())
	if err != nil {
		return nil, "", err
	}

	return configs, digest, nil
}

func (s *Scheduler) ReadConfigFromLocalOrGithub() (*protocol.PanopticConfigs, error) {
	configs := &protocol.PanopticConfigs{}

	if AppSetting.FORCE_REMOTE_SCHEDULE_PULL || utils.IsProdEnv() {
		Logger.Log.Infoln("read PanopticConfig from Github project: https://github.com/Luismorlan/panoptic_config")
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: os.Getenv("GITHUB_ACCESS_TOKEN")},
		)
		tc := oauth2.NewClient(s.ctx, ts)
		client := github.NewClient(tc)
		content, _, res, err := client.Repositories.GetContents(s.ctx, "Luismorlan", "panoptic_config", "config.textproto", nil)
		if err != nil {
			return nil, err
		}
		if res.StatusCode != 200 {
			return nil, fmt.Errorf("fail to get config from Github, http code %d", res.StatusCode)
		}
		decode, _ := base64.StdEncoding.DecodeString(*content.Content)
		if err := prototext.Unmarshal(decode, configs); err != nil {
			return nil, err
		}
	} else {
		Logger.Log.Infoln("read PanopticConfig from local workspace, file", AppSetting.LOCAL_PANOPTIC_CONFIG_PATH)
		in, err := ioutil.ReadFile(AppSetting.LOCAL_PANOPTIC_CONFIG_PATH)
		if err != nil {
			return nil, err
		}
		if err := prototext.Unmarshal(in, configs); err != nil {
			return nil, err
		}
	}

	return configs, nil
}

func (s *Scheduler) ParseAndUpsertJobs() ( /*reschedule*/ bool, error) {
	configs, digest, err := s.ReadConfig()
	if err != nil {
		return false, err
	}

	// If config hasn't changed, do nothing.
	if s.ScheduleDigest == digest {
		return false, nil
	}

	Logger.Log.Infof("parsed PanopticConfigs: %s", configs.String())

	jobs := NewSchedulerJobs(configs, s.ctx)
	err = ValidateJobs(jobs)
	if err != nil {
		return false, err
	}

	s.UpsertJobs(jobs)
	s.UpdateConfigDigest(digest)

	return true, nil
}

func (s *Scheduler) DoSingleJob(job *SchedulerJob) {
	err := s.Doer.Do(job)
	if err != nil {
		log.Printf(
			"Job execution failed. Name: %s, err: %v",
			job.panopticConfig.Name,
			err,
		)
	}
}

func (s *Scheduler) ScheduleSingleJob(job *SchedulerJob) {
	// Start immediately if required and never ran before.
	if !job.HasRunBefore() && job.panopticConfig.TaskSchedule.StartImmediatly {
		job.UpdateLastAndNextTime()
		// Execute the job in a non-blocking way so that we the execution time will
		// not skew the next run time.
		go s.DoSingleJob(job)
	}

	for {
		durationTillNextRun := job.DurationTillNextRun()
		select {
		// Scheduler's lifecycle is managed by engine's context, cancelling engine
		// should also shutdown scheduler.
		case <-s.ctx.Done():
			log.Printf("Job %s cancelled by scheduler.", job.panopticConfig.Name)
			return
		// In the future, a job could cancel itself under some condition (e.g. keep
		// failing, reach max run count). This can also happen when the job is
		// rescheduled due to config change, where we proactively cancel the job.
		case <-job.ctx.Done():
			log.Printf("Job %s cancelled by itself.", job.panopticConfig.Name)
			return
		case <-time.After(durationTillNextRun):
			job.UpdateLastAndNextTime()
			go s.DoSingleJob(job)
		}
	}
}

// A blocking call that returns once all jobs finished running. This function
// can be called multiple times. Each time it's called it will firstly remove
// all previous schedules, and reschedule those new jobs.
func (s *Scheduler) ScheduleJobs() {
	digest := s.ScheduleDigest
	log.Println("SchedulerJobs started with config digest: ", digest)
	var wg sync.WaitGroup

	// Critical section.
	s.m.RLock()
	for _, j := range s.Jobs {
		j.RefreshContext(s.ctx)

		wg.Add(1)
		go func(job *SchedulerJob) {
			defer wg.Done()
			s.ScheduleSingleJob(job)
		}(j)
	}
	s.m.RUnlock()

	wg.Wait()
	log.Println("SchedulerJobs ended with config digest: ", digest)
}

func (s *Scheduler) WatchConfigAndMaybeReschedule() {
	for {
		reschedule, err := s.ParseAndUpsertJobs()
		if err != nil {
			Logger.Log.Errorf("error parsing config: %s", err)
		}
		if reschedule {
			go s.ScheduleJobs()
		}

		time.Sleep(time.Duration(AppSetting.SCHEDULER_CONFIG_POLL_INTERVAL_SECOND) * time.Second)
	}
}

func (s *Scheduler) RunModule(ctx context.Context) error {
	s.WatchConfigAndMaybeReschedule()
	return nil
}

func (s *Scheduler) Name() string {
	return s.Config.Name
}

func (s *Scheduler) Shutdown() {
	// There's no need to free this lock because it doesn't really matter if we
	// are shutting down. Also it's a good practice that no additional internal
	// state change can happen.
	s.m.Lock()
	// Cancel all jobs
	for _, job := range s.Jobs {
		job.cancel()
	}
	Logger.Log.Infoln("Module ", s.Config.Name, " gracefully shutdown")
}
