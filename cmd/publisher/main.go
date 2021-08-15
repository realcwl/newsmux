package main

import (
	"time"

	. "github.com/Luismorlan/newsmux/publisher"
	. "github.com/Luismorlan/newsmux/utils"
	. "github.com/Luismorlan/newsmux/utils/log"
)

const (
	// TODO: Move to .env
	CRAWLER_PUBLISHER_QUEUE_NAME = "crawler-publisher-queue"
	MESSAGE_PROCESS_CONCURRENCY  = 1
)

func main() {
	reader, err := NewSQSMessageQueueReader(CRAWLER_PUBLISHER_QUEUE_NAME, 20)
	if err != nil {
		Log.Fatal("fail initialize SQS message queue reader : ", err)
	}

	// Main publish logic lives in processor
	processor := NewPiblisherMessageProcessor(reader)

	for {
		processor.ReadAndProcessMessages(MESSAGE_PROCESS_CONCURRENCY)

		// Protective delay
		time.Sleep(2 * time.Second)
	}
}
