package main

import (
	"time"

	. "github.com/Luismorlan/newsmux/publisher"
	. "github.com/Luismorlan/newsmux/utils"
	. "github.com/Luismorlan/newsmux/utils/log"
)

const (
	// TODO: Move to .env
	crawlerPublisherQueueName = "crawler-publisher-queue"
	messageProcessConcurrency = 1
)

func main() {
	reader, err := NewSQSMessageQueueReader(crawlerPublisherQueueName, 20)
	if err != nil {
		Log.Fatal("fail initialize SQS message queue reader : ", err)
	}

	// Main publish logic lives in processor
	processor := NewpublisherMessageProcessor(reader)

	for {
		processor.ReadAndProcessMessages(messageProcessConcurrency)

		// Protective delay
		time.Sleep(2 * time.Second)
	}
}
