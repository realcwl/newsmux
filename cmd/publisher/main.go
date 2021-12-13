package main

import (
	"flag"
	"log"
	"time"

	"github.com/Luismorlan/newsmux/deduplicator"
	"github.com/Luismorlan/newsmux/protocol"
	. "github.com/Luismorlan/newsmux/publisher"
	"github.com/Luismorlan/newsmux/utils"
	. "github.com/Luismorlan/newsmux/utils"
	"github.com/Luismorlan/newsmux/utils/dotenv"
	. "github.com/Luismorlan/newsmux/utils/flag"
	. "github.com/Luismorlan/newsmux/utils/log"
	"google.golang.org/grpc"
)

const (
	crawlerPublisherQueueName    = "newsfeed_crawled_items_queue.fifo"
	devCrawlerPublisherQueueName = "crawler-publisher-queue"
	// Read batch size must be within [1, 10]
	sqsReadBatchSize                 = 10
	publishMaxBackOffSeconds float64 = 2.0
	initialBackOff           float64 = 0.1
)

var (
	serverAddr = flag.String("deduplicator_addr", "localhost:50051", "The server address in the format of host:port for deduplicator")
)

func getDeduplicatorClientAndConnection() (protocol.DeduplicatorClient, *grpc.ClientConn) {
	if !utils.IsProdEnv() {
		return deduplicator.FakeDeduplicatorClient{}, nil
	}

	opts := []grpc.DialOption{grpc.WithInsecure()}
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial deduplicator: %v", err)
	}
	client := protocol.NewDeduplicatorClient(conn)

	return client, conn
}

func getNewBackOff(backOff float64) float64 {
	if backOff == 0.0 {
		return initialBackOff
	} else if backOff*2 < publishMaxBackOffSeconds {
		return 2 * backOff
	}
	return publishMaxBackOffSeconds
}

func main() {
	ParseFlags()
	InitLogger()

	if err := dotenv.LoadDotEnvs(); err != nil {
		Log.Fatal("fail to load env : ", err)
	}

	db, err := GetDBConnection()
	if err != nil {
		Log.Fatal("fail to connect database : ", err)
	}
	PublisherDBSetup(db)

	client, conn := getDeduplicatorClientAndConnection()
	defer conn.Close()

	sqsName := crawlerPublisherQueueName
	if !utils.IsProdEnv() {
		sqsName = devCrawlerPublisherQueueName
	}
	reader, err := NewSQSMessageQueueReader(sqsName, 20)
	if err != nil {
		Log.Fatal("fail initialize SQS message queue reader : ", err)
	}

	// Main publish logic lives in processor
	processor := NewPublisherMessageProcessor(reader, db, client)

	// Exponentially backoff on
	backOff := 0.0
	for {
		successCount := processor.ReadAndProcessMessages(sqsReadBatchSize)
		if successCount == 0 {
			backOff = getNewBackOff(backOff)
		} else {
			backOff = 0.0
		}

		// Protective back off on read or process failure.
		time.Sleep(time.Duration(backOff) * time.Second)
	}
}
