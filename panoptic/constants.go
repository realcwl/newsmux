package panoptic

const (
	// Task emitted by executor and is in pending state.
	TopicPendingJob  = "topic.pending_job"
	TopicExecutedJob = "topic.executed_job"

	LambdaAwsRole      = "arn:aws:iam::213288384225:role/service-role/test_ddog_logging-role-8qnsddqu"
	DataCollectorImage = "213288384225.dkr.ecr.us-west-1.amazonaws.com/data_collector:latest"
	AwsRegion          = "us-west-1"

	// Datadog related
	DdogTaskStateCounter              = "task_state_counter"
	DdogTaskSuccessMessageCounter     = "task_crawled_message_counter"
	DdogTaskFailureMessageCounter     = "task_failure_message_counter"
	DdogTaskExecutionTimeDistribution = "task_execution_time_distribution"
)

type LambdaExecutorState int64

const (
	Uninitialized LambdaExecutorState = 0
	Runnable      LambdaExecutorState = 1
	Running       LambdaExecutorState = 2
)

type LambdaFunctionState int64

const (
	// Lambda function is active, can take in new job.
	Active LambdaFunctionState = 0
	// Lambda function is stale, should not accept new job. But shouldn't be
	// cleaned up because it still has pending tasks.
	Stale LambdaFunctionState = 1
	// Lambda function is both stale and has no pending job, should be removed.
	Removable LambdaFunctionState = 2
)
