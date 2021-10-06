package collector

import (
	"os"

	"github.com/Luismorlan/newsmux/protocol"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
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
		Credentials: credentials.NewStaticCredentials(
			os.Getenv("AWS_ACCESS_KEY_ID_FOR_AWS"),
			os.Getenv("SECRET_ACCESS_KEY"),
			"",
		),
	})
	if err != nil {
		return nil, err
	}
	svc := sns.New(sess)

	return &SnsSink{
		arn:    testArn,
		client: svc,
	}, nil
}

func (s *SnsSink) Push(msg *protocol.CrawlerMessage) error {
	if msg == nil {
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
