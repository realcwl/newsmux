package main

import (
	"flag"
	"time"

	. "github.com/Luismorlan/newsmux/publisher"
	. "github.com/Luismorlan/newsmux/utils"
	"github.com/Luismorlan/newsmux/utils/dotenv"
	. "github.com/Luismorlan/newsmux/utils/log"
)

const (
	// TODO: Move to .env
	crawlerPublisherQueueName = "newsfeed_crawled_items_queue.fifo"
	// Read batch size must be within [1, 10]
	sqsReadBatchSize                 = 10
	publishMaxBackOffSeconds float64 = 2.0
	initialBackOff           float64 = 0.1
)

func getNewBackOff(backOff float64) float64 {
	if backOff == 0.0 {
		return initialBackOff
	} else if backOff*2 < publishMaxBackOffSeconds {
		return 2 * backOff
	}
	return publishMaxBackOffSeconds
}

func main() {
	flag.Parse()
	if err := dotenv.LoadDotEnvs(); err != nil {
		Log.Fatal("fail to load env : ", err)
	}

	db, err := GetDBConnection()
	if err != nil {
		Log.Fatal("fail to connect database : ", err)
	}

	reader, err := NewSQSMessageQueueReader(crawlerPublisherQueueName, 20)
	if err != nil {
		Log.Fatal("fail initialize SQS message queue reader : ", err)
	}

	// Main publish logic lives in processor
	processor := NewPublisherMessageProcessor(reader, db)

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
