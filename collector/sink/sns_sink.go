package sink

import (
	"encoding/base64"

	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"google.golang.org/protobuf/proto"
)

const (
	testSnsArn = "arn:aws:sns:us-west-1:213288384225:test_sns"
	prodSnsArn = "arn:aws:sns:us-west-1:213288384225:newsfeed.fifo"
)

type SnsSink struct {
	arn    string
	client *sns.SNS
}

func NewSnsSink() (*SnsSink, error) {
	// AWS client session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-1"),
	})
	if err != nil {
		return nil, err
	}
	svc := sns.New(sess)

	arn := testSnsArn
	if utils.IsProdEnv() {
		arn = prodSnsArn
	}

	return &SnsSink{
		arn:    arn,
		client: svc,
	}, nil
}

func (s *SnsSink) Push(msg *protocol.CrawlerMessage) error {
	if msg == nil {
		Logger.Log.Warn("push empty message into queue")
		return nil
	}

	serializedMsg, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	// we use base64 encoded string in sns
	b64 := base64.StdEncoding.EncodeToString([]byte(serializedMsg))

	messageGroup := "global_queue"
	// ignore the returned seq number for FIFO
	_, err = s.client.Publish(&sns.PublishInput{
		Message:                &b64,
		TopicArn:               &s.arn,
		MessageGroupId:         &messageGroup,
		MessageDeduplicationId: &msg.Post.DeduplicateId,
	})
	return err
}
