package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/Luismorlan/newsmux/app_setting"
	"github.com/Luismorlan/newsmux/panoptic"
	"github.com/Luismorlan/newsmux/panoptic/modules"
	"github.com/Luismorlan/newsmux/utils/dotenv"
	. "github.com/Luismorlan/newsmux/utils/flag"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

var (
	AppSettingPath *string
	// Configuration to customize binary startup.
	AppSetting app_setting.PanopticAppSetting
)

// init() will always be called on before the execution of main function.
func init() {
	AppSettingPath = flag.String("app_setting_path", "cmd/panoptic/config.yaml", "path to panoptic app setting")
	if err := dotenv.LoadDotEnvs(); err != nil {
		panic(err)
	}
}

func CreateAndInitLambdaExecutor(ctx context.Context) *modules.LambdaExecutor {
	var client *lambda.Client

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(panoptic.AWS_REGION),
	)
	if err != nil {
		panic(err)
	}
	client = lambda.NewFromConfig(cfg)

	executor := modules.NewLambdaExecutor(ctx, client, &modules.LambdaExecutorConfig{
		LambdaPoolSize:       AppSetting.LAMBDA_POOL_SIZE,
		LambdaLifeSpanSecond: AppSetting.LAMBDA_LIFE_SPAN_SECOND,
		MaintainEverySecond:  AppSetting.MAINTAIN_EVERY_SECOND,
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
	ParseFlags()

	AppSetting = app_setting.ParsePanopticAppSetting(*AppSettingPath)

	eventbus := gochannel.NewGoChannel(
		gochannel.Config{
			OutputChannelBuffer:            100,
			BlockPublishUntilSubscriberAck: false,
		},
		watermill.NewStdLogger(false, false),
	)

	rootCtx := context.Background()
	ctx, cancel := context.WithCancel(rootCtx)

	// Initialize all engine modules here.
	modules := []panoptic.Module{
		// Reporter reports the execution metrics to datadog for monitoring purpose.
		modules.NewReporter(modules.ReporterConfig{Name: "reporter"}, NewDogStatsdClient(), eventbus),
		// Scheduler parses data collector configs, fanout into multiple tasks and
		// pushes onto EventBus.
		modules.NewScheduler(
			&AppSetting,
			modules.SchedulerConfig{Name: "scheduler"},
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

	engine := panoptic.NewEngine(modules, ctx, cancel, eventbus)

	go engine.Run()

	// Wait for ctrl+c (SIGINT) to gracefully shutdown the entire process.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	engine.Shutdown()

	log.Println("engine stopped execution.")
}
