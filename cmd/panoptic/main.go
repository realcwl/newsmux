package main

import (
	"context"
	"log"
	"os"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/Luismorlan/newsmux/panoptic"
	"github.com/Luismorlan/newsmux/panoptic/modules"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

func CreateAndInitLambdaExecutor(ctx context.Context) *modules.LambdaExecutor {
	//
	var client *lambda.Client
	env := os.Getenv("NEWSMUX_ENV")
	if env == "prod" {
		client = lambda.New(lambda.Options{Region: panoptic.AWS_REGION})
	} else {
		cfg, err := config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(panoptic.AWS_REGION),
		)
		if err != nil {
			panic(err)
		}
		client = lambda.NewFromConfig(cfg)
	}

	executor := modules.NewLambdaExecutor(ctx, client, &modules.LambdaExecutorConfig{
		LambdaPoolSize:       1,
		LambdaLifeSpanSecond: 300,
		MaintainEverySecond:  10,
	})
	if err := executor.Init(); err != nil {
		panic(err)
	}
	return executor
}

func NewDogStatsdClient() *statsd.Client {
	statsd, err := statsd.New("127.0.0.1:8125")
	if err != nil {
		panic(err)
	}
	return statsd
}

func main() {
	eventbus := gochannel.NewGoChannel(
		gochannel.Config{
			OutputChannelBuffer:            100,
			BlockPublishUntilSubscriberAck: false,
		},
		watermill.NewStdLogger(false, false),
	)
	ctx := context.Background()

	// Initialize all engine modules here.
	modules := []panoptic.Module{
		// Reporter reports the execution metrics to datadog for monitoring purpose.
		modules.NewReporter(modules.ReporterConfig{Name: "reporter"}, NewDogStatsdClient(), eventbus),
		// Scheduler parses data collector configs, fanout into multiple tasks and
		// pushes onto EventBus.
		modules.NewScheduler(
			modules.SchedulerConfig{Name: "scheduler",
				PanopticConfigPath: "panoptic/data/testing_panoptic_config.textproto"},
			eventbus,
			modules.NewSchedulerJobDoer(eventbus),
			ctx,
		),
		// Orchestrator listens tasks on EventBus, maintains an active Lambda pool
		// and wrap Lambda result in a tasks and publish to the exporter for
		// monitoring.
		modules.NewOrchestrator(
			modules.OrchestratorConfig{Name: "orchestrator"},
			CreateAndInitLambdaExecutor(ctx),
			eventbus,
		),
	}

	engine := panoptic.NewEngine(modules, eventbus)

	// blocking call.
	engine.Run(ctx)

	log.Println("engine stopped execution.")
}
