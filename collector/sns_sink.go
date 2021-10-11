package collector

import (
	"github.com/Luismorlan/newsmux/protocol"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
)

const (
	testArn = "arn:aws:sns:us-west-1:213288384225:test_sns"
	prodArn = "arn:aws:sns:us-west-1:213288384225:newsfeed.fifo"
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

	return &SnsSink{
		arn:    prodArn,
		client: svc,
	}, nil
}

func (s *SnsSink) Push(msg *protocol.CrawlerMessage) error {
	if msg == nil {
		Logger.Log.Warn("push empty message into queue")
		return nil
	}
	serializedMsg := msg.String()
	// ignore the returned seq number for FIFO
	_, err := s.client.Publish(&sns.PublishInput{
		Message:  &serializedMsg,
		TopicArn: &s.arn,
	})
	return err
}
