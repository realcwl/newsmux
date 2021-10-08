package modules

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/panoptic"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	. "github.com/Luismorlan/newsmux/utils/log"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

// Configuration of the lambda executor.
type LambdaExecutorConfig struct {
	// Number of Lambdas maintained at a given time.
	LambdaPoolSize int

	// Lambda life span in second. Any lambda function that exceed this
	// value will be cleaned up and replaced with a new one.
	LambdaLifeSpanSecond int64

	// Maintain the lambda pool every other interval.
	MaintainEverySecond int64
}

// LambdaFunction maintains the in-memory state for the LambdaFunction, which
// contains the function state, as well as jobs on the function.
type LambdaFunction struct {
	// Actual lambda name on AWS
	name string
	// Time this lambda function is created. A Lambda function is considered stale
	// if time.Now() - createdTime > LambdaLifeSpanSecond
	createdTime time.Time
	// How long should this lambda function be considered stale
	span time.Duration
	// Pending jobs on this Lambda function, keyed by JobId.
	m    sync.RWMutex
	jobs map[string]*protocol.PanopticJob
}

// Lambda Executor executes jobs on AWS lambda, it maintains a list of active
// lambda functions and deprecate old Lambda functions if they become stale.
type LambdaExecutor struct {
	config *LambdaExecutorConfig

	// The context of this LambdaExecutor
	ctx context.Context

	// State of the LambdaExecutor
	state panoptic.LambdaExecutorState

	// pool contains all active function, while stalePool contains all stale
	// function waiting to be cleaned up. Accessing to both pool should be guarded
	// by mutex m.
	m         sync.RWMutex
	pool      []*LambdaFunction
	stalePool sync.Map

	// AWS Lambda Client that actually in charge of executing lambda function
	lambdaClient *lambda.Client
}

// TODO(chenweilunster): Remove this once api is unified.
type DataCollectorRequest struct {
	SerializedJob []byte
}

// Create an uninitialized LambdaExecutor.
func NewLambdaExecutor(ctx context.Context,
	client *lambda.Client, cfg *LambdaExecutorConfig) *LambdaExecutor {
	return &LambdaExecutor{
		m:            sync.RWMutex{},
		config:       cfg,
		ctx:          ctx,
		state:        panoptic.UNINITIALIZED,
		lambdaClient: client,
		pool:         []*LambdaFunction{},
		stalePool:    sync.Map{},
	}
}

func NewLambdaFunction(out *lambda.CreateFunctionOutput, span time.Duration) (*LambdaFunction, error) {
	lastModifiedTime, err := time.Parse("2006-01-02T15:04:05-0700", *out.LastModified)

	if err != nil {
		return nil, err
	}

	return &LambdaFunction{
		name:        *out.FunctionName,
		createdTime: lastModifiedTime,
		span:        span,
		m:           sync.RWMutex{},
		jobs:        make(map[string]*protocol.PanopticJob),
	}, nil
}

// We need to poison each lambda's life span with a standard deviation so that
// all Lambda will not start and stop all at once, creating a vacuum gap where
// no Lambda exists.
func GetLambdaLifeSpanWithRandomness(LambdaLifeSpanSecond int64) time.Duration {
	deviationSec := utils.GetRandomNumberInRangeStandardDeviation(float64(LambdaLifeSpanSecond), float64(LambdaLifeSpanSecond)/4)
	return time.Duration(deviationSec * float64(time.Second))
}

func WaitForLambdaActivate(ctx context.Context, functionName string, lambdaClient *lambda.Client) error {
	waiter := lambda.NewFunctionActiveWaiter(lambdaClient)
	err := waiter.Wait(ctx, &lambda.GetFunctionConfigurationInput{
		FunctionName: &functionName,
	}, time.Duration(300*time.Second))
	if err != nil {
		return err
	}
	return nil
}

func MakeDataCollectorRpc(ctx context.Context, job *protocol.PanopticJob, functionName string, lambdaClient *lambda.Client) (*protocol.PanopticJob, error) {
	err := WaitForLambdaActivate(ctx, functionName, lambdaClient)
	if err != nil {
		return nil, err
	}

	// Invoke Lambda
	payload, err := model.PanopticJobToLambdaPayload(job)
	if err != nil {
		return nil, err
	}

	res, err := lambdaClient.Invoke(ctx, &lambda.InvokeInput{
		FunctionName: &functionName,
		Payload:      payload,
	})

	if err != nil {
		return nil, err
	}

	return model.LambdaPayloadToPanopticJob(res.Payload)
}

// A function can be removed if it contains no pending jobs
func (f *LambdaFunction) IsRemovable() bool {
	f.m.RLock()
	defer f.m.RUnlock()

	return len(f.jobs) == 0
}

// A function is stale if its created time is too long ago
func (f *LambdaFunction) IsStale() bool {
	now := time.Now()
	return now.Sub(f.createdTime) < f.span
}

// Add a pending job on the lambda function.
func (f *LambdaFunction) AddPendingJob(job *protocol.PanopticJob) {
	f.m.Lock()
	defer f.m.Unlock()

	f.jobs[job.JobId] = job
}

func (f *LambdaFunction) DeletePendingJob(job *protocol.PanopticJob) {
	f.m.Lock()
	defer f.m.Unlock()

	delete(f.jobs, job.JobId)
}

// Init initialize the Lambda pool this executor maintains, make sure they are
// created with the already uploaded image. It also spins up a garbage cleaner
// goroutine that retires stale
func (l *LambdaExecutor) Init() error {
	if l.config == nil {
		return errors.New("cannot initialize LambdaExecutor with empty config")
	}
	err := l.IntializeLambdaPool()
	if err != nil {
		return err
	}

	l.MaintainLambdaPool()

	l.state = panoptic.RUNNABLE
	return nil
}

// Create and add a new Lambda function to Lambda pool.
func (l *LambdaExecutor) AddLambdaFunction() (string, error) {
	funcName := utils.GetRandomDataCollectorFunctionName()
	role := panoptic.LAMBDA_AWS_ROLE
	imageUri := panoptic.DATA_COLLECTOR_IMAGE

	res, err := l.lambdaClient.CreateFunction(l.ctx, &lambda.CreateFunctionInput{
		FunctionName: &funcName,
		Role:         &role,
		Code: &types.FunctionCode{
			ImageUri: &imageUri,
		},
		PackageType: types.PackageTypeImage,
	})

	if err != nil {
		return "", err
	}

	lambdaFunction, err := NewLambdaFunction(res, GetLambdaLifeSpanWithRandomness(l.config.LambdaLifeSpanSecond))

	if err != nil {
		return "", err
	}

	l.m.Lock()
	defer l.m.Unlock()
	l.pool = append(l.pool, lambdaFunction)

	return lambdaFunction.name, nil
}

// Add multiple lambda functions. If one function fails to create, revert all
// creations.
func (l *LambdaExecutor) AddLambdaFunctions(count int) ([]string, error) {
	wg := sync.WaitGroup{}
	m := sync.Mutex{}
	names := []string{}
	errs := []error{}
	i := 0
	for i < count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			name, err := l.AddLambdaFunction()

			m.Lock()
			defer m.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			names = append(names, name)
		}()
		i++
	}
	wg.Wait()

	if len(errs) != 0 {
		// Clean up created lambda functions.
		for _, name := range names {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				l.DeleteLambdaFunction(name)
			}(name)
		}

		wg.Wait()
		return nil, errors.New("fail to initialize Lambda Pool due to failure")
	}

	return names, nil
}

// Delete lambda function by name, return error if there's any
func (l *LambdaExecutor) DeleteLambdaFunction(name string) error {
	_, err := l.lambdaClient.DeleteFunction(l.ctx, &lambda.DeleteFunctionInput{
		FunctionName: &name,
	})

	if err != nil {
		return err
	}

	Log.Infof("removed lambda function %s", name)

	return nil
}

// A blocking call which initialzie Lambda Pool
func (l *LambdaExecutor) IntializeLambdaPool() error {
	// Do no initialize an already initialized pool.
	if l.state != panoptic.UNINITIALIZED {
		return nil
	}

	names, err := l.AddLambdaFunctions(l.config.LambdaPoolSize)
	if err != nil {
		return err
	}

	Log.Infof("initialized lambda pool, names: %s\n", strings.Join(names, ", "))

	return nil
}

// Maintain the Lambda Pool state in a separate goroutine.
// This goroutine should do the following 3 things:
// 1. Move lambda function from active pool to stalePool.
// 2. Delete stalePool function where its state is removable.
// 3. Create new lambda function and make sure it always has enough lambda
// functions in pool.
func (l *LambdaExecutor) MaintainLambdaPool() {
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(l.config.MaintainEverySecond * int64(time.Second))):
				Log.Infof("start lambda pool maintainance")
				l.MarkStaleFunctions()
				if err := l.FillLambdaPool(); err != nil {
					Log.Errorf("fail to fill Lambda Pool, %s", err)
				}
				l.DeleteRemovableFunctions()
				continue
			}
		}
	}(l.ctx)
}

// Move all stale functions from active pool to stale pool.
func (l *LambdaExecutor) MarkStaleFunctions() {
	l.m.Lock()
	defer l.m.Unlock()

	i := 0
	for i < len(l.pool) {
		f := l.pool[i]
		if f.IsStale() {
			i++
			continue
		}
		l.pool = append(l.pool[:i], l.pool[i+1:]...)
		l.stalePool.Store(f.name, f)
	}
}

// Fill lambda pool until full
func (l *LambdaExecutor) FillLambdaPool() error {
	l.m.Lock()
	count := l.config.LambdaPoolSize - len(l.pool)
	l.m.Unlock()

	names, err := l.AddLambdaFunctions(count)
	if err != nil {
		return err
	}

	for _, name := range names {
		Log.Infof("refill lambda function %s\n", name)
	}

	return nil
}

func (l *LambdaExecutor) DeleteRemovableFunctions() {
	l.m.Lock()
	defer l.m.Unlock()

	l.stalePool.Range(func(key, value interface{}) bool {
		name := key.(string)
		f := value.(*LambdaFunction)
		if f.IsRemovable() {
			l.stalePool.Delete(name)
			go l.DeleteLambdaFunction(name)
		}
		return true
	})
}

// Return a random function for execution. Returns nil if no active lambda.
func (l *LambdaExecutor) GetRandomActiveFunction(job *protocol.PanopticJob) *LambdaFunction {
	l.m.Lock()
	defer l.m.Unlock()

	if len(l.pool) == 0 {
		return nil
	}

	idx := rand.Intn(len(l.pool))

	f := l.pool[idx]

	// We must register this job when we fetch it, otherwise there might be a case
	// that when we return this Lambda function, routinely maintainance kicks in
	// and this function is garbage collected.
	f.AddPendingJob(job)
	return f
}

// LambdaExecutor is a blocking call that executes a single PanopticJob on AWS
// Lambda. It returns the input job with additional metadata describing the
// execution result.
func (l *LambdaExecutor) Execute(ctx context.Context, job *protocol.PanopticJob) (*protocol.PanopticJob, error) {
	// Get a active Lambda function with gracefully retry.
	var f *LambdaFunction
	for {
		f = l.GetRandomActiveFunction(job)
		if f != nil {
			break
		}
		time.Sleep(1 * time.Second)
	}

	defer f.DeletePendingJob(job)
	return MakeDataCollectorRpc(ctx, job, f.name, l.lambdaClient)
}
