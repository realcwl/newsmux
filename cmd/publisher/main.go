package main

import (
	"time"

	. "github.com/Luismorlan/newsmux/publisher"
	. "github.com/Luismorlan/newsmux/utils"
	"github.com/Luismorlan/newsmux/utils/dotenv"
	. "github.com/Luismorlan/newsmux/utils/log"
)

const (
	// TODO: Move to .env
	crawlerPublisherQueueName = "newsfeed_crawled_items_queue.fifo"
	messageProcessConcurrency = 1
)

func main() {
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

	for {
		processor.ReadAndProcessMessages(messageProcessConcurrency)

		// Protective delay
		time.Sleep(2 * time.Second)
	}
}
