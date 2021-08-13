package protocol

import (
	"log"
	"testing"

	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestProtoBuf(t *testing.T) {
	origin := CrawlerMessage{
		SourceId:       "1",
		SubSourceId:    "2",
		Title:          "hello",
		Content:        "hello world!",
		ImageUrls:      []string{"1", "4"},
		FilesUrls:      []string{"2", "3"},
		CrawledAt:      &timestamp.Timestamp{},
		OriginUrl:      "aaa",
		CrawlerIp:      "123",
		CrawlerVersion: "vde",
		IsTest:         false,
	}
	out, err := proto.Marshal(&origin)
	if err != nil {
		log.Fatalln("Failed to encode:", err)
	}

	decoded := &CrawlerMessage{}
	if err := proto.Unmarshal(out, decoded); err != nil {
		log.Fatalln("Failed to parse:", err)
	}

	assert.Equal(t, decoded.SourceId, origin.SourceId)
	assert.Equal(t, decoded.SubSourceId, origin.SubSourceId)
	assert.Equal(t, decoded.Content, origin.Content)
	assert.Equal(t, decoded.ImageUrls, origin.ImageUrls)
	assert.Equal(t, decoded.CrawledAt, origin.CrawledAt)
	assert.Equal(t, decoded.IsTest, origin.IsTest)
}
