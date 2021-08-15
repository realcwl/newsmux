package publisher

import (
	b64 "encoding/base64"
	"testing"

	"github.com/Luismorlan/newsmux/protocol"
	. "github.com/Luismorlan/newsmux/utils"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

type TestMessageQueueReader struct {
	msgs []*MessageQueueMessage
}

func (reader *TestMessageQueueReader) DeleteMessage(msg *MessageQueueMessage) error {
	return nil
}

// Always return all messages
func (reader *TestMessageQueueReader) ReceiveMessages(maxNumberOfMessages int64) (msgs []*MessageQueueMessage, err error) {
	return reader.msgs, nil
}

// Pass in all the crawler messages that will be used for testing
// Reader will return queue message object
func NewTestMessageQueueReader(crawlerMsgs []*protocol.CrawlerMessage) *TestMessageQueueReader {
	var res TestMessageQueueReader
	var queueMsgs []*MessageQueueMessage

	for _, m := range crawlerMsgs {
		encodedBytes, _ := proto.Marshal(m)
		str := b64.StdEncoding.EncodeToString(encodedBytes)
		var msg MessageQueueMessage
		msg.Message = &str
		queueMsgs = append(queueMsgs, &msg)
	}
	res.msgs = queueMsgs
	return &res
}

func TestDecodeCrawlerMessage(t *testing.T) {

	origin := protocol.CrawlerMessage{
		SourceId:           "1",
		SubSourceId:        "2",
		Title:              "hello",
		Content:            "hello world!",
		ImageUrls:          []string{"1", "4"},
		FilesUrls:          []string{"2", "3"},
		CrawledAt:          &timestamp.Timestamp{},
		OriginUrl:          "aaa",
		CrawlerIp:          "123",
		CrawlerVersion:     "vde",
		IsTest:             false,
		ContentGeneratedAt: &timestamp.Timestamp{},
	}

	reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
		&origin,
	})

	// Inject test dependent reader
	processor := NewPiblisherMessageProcessor(reader)

	msgs, _ := reader.ReceiveMessages(1)
	assert.Equal(t, len(msgs), 1)

	// This is the function we tested here
	//given MessageQueueMessage, decode it into struct
	decodedObj, _ := processor.decodeCrawlerMessage(msgs[0])

	assert.True(t, cmp.Equal(*decodedObj, origin, cmpopts.IgnoreUnexported(protocol.CrawlerMessage{}, timestamp.Timestamp{})))
}
