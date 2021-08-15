package publisher

import (
	b64 "encoding/base64"

	. "github.com/Luismorlan/newsmux/protocol"
	. "github.com/Luismorlan/newsmux/utils"
	. "github.com/Luismorlan/newsmux/utils/log"
	"google.golang.org/protobuf/proto"
)

type CrawlerPiblisherMessageProcessor struct {
	Reader MessageQueueReader
}

// Create new processor with reader dependency injection
func NewPiblisherMessageProcessor(reader MessageQueueReader) *CrawlerPiblisherMessageProcessor {
	return &CrawlerPiblisherMessageProcessor{
		Reader: reader,
	}
}

// Use Reader to read N messages and process them in parallel
// Time out or queue name etc are defined in reader
// Reader focus on how to get message from queue
// Processor focus on how to process the message
// This function doesn't return anything, only log errors
func (processor *CrawlerPiblisherMessageProcessor) ReadAndProcessMessages(maxNumberOfMessages int64) {
	// Pull queued messages from queue
	msgs, err := processor.Reader.ReceiveMessages(maxNumberOfMessages)

	if err != nil {
		Log.Error("fail read crawler messages from queue : ", err)
		return
	}

	// Process
	// TODO: process in parallel
	for _, msg := range msgs {
		if err := processor.ProcessOneCralwerMessage(msg); err != nil {
			Log.Error("fail process one crawler message : ", err)
			continue
		}
	}
}

// Process one cralwer-publisher message
// Step1. decode into protobuf generated struct
// Step2. do publishing with new post
// Step3. if publishing succeeds, delete message in queue
func (processor *CrawlerPiblisherMessageProcessor) ProcessOneCralwerMessage(msg *MessageQueueMessage) error {

	decodedMsg, err := processor.decodeCrawlerMessage(msg)
	if err != nil {
		return err
	}

	// TODO: Do actual publishing for decodedMsg
	Log.Info(decodedMsg)

	return processor.Reader.DeleteMessage(msg)
}

// Parse message into meaningful structure CrawlerMessage
// This function assumes message passed in can be parsed, otherwise it will throw error
func (processor *CrawlerPiblisherMessageProcessor) decodeCrawlerMessage(msg *MessageQueueMessage) (*CrawlerMessage, error) {
	str, err := msg.Read()
	if err != nil {
		return nil, err
	}

	sDec, err := b64.StdEncoding.DecodeString(str)
	if err != nil {
		return nil, err
	}

	decodedMsg := &CrawlerMessage{}
	if err := proto.Unmarshal(sDec, decodedMsg); err != nil {
		return nil, err
	}

	return decodedMsg, nil
}
