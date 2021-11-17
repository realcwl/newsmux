package main

import (
	"fmt"
	"log"

	b64 "encoding/base64"

	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils/dotenv"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// This binary is to generate a test message for end to end testing
func main() {
	dotenv.LoadDotEnvs()

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := sqs.New(sess)

	queueName := "crawler-publisher-queue"
	qURL, _ := svc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: &queueName,
	})
	fmt.Println(qURL)

	// Example crawled Post
	origin := protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			SubSource: &protocol.CrawledSubSource{
				Id: "ebe51059-6c1c-4dd5-ad66-4d377c306b82",
			},
			Title:     "阿富汗官员：塔利班已占领北部城市马扎里沙里夫，安全部队向乌兹别克斯坦边境方向逃跑。",
			Content:   "【江苏中小学今秋开学后推行“5加2”课后服务】江苏省四部门要求，今年秋学期开学后，确保全省所有义务教育学校和有需要的学生全覆盖。推行课后服务“5+2”模式，学校每周5天（周一至周五）都要开展课后服务，每天至少提供2小时的课后服务，课后服务结束时间原则上不早于当地正常下班时间。有条件的初中，周一至周五可设晚自习（一般安排2小时以内，原则上不晚于20：30结束）。课后服务必须突出育人导向，提供丰富课程，学校不得利用课后服务时间讲授新课。（央视",
			ImageUrls: []string{"http://54.176.72.76:8080/api/weiboimage/0a4bdbb3eff672f6ef5f811f13cf65ab.jpg", "http://54.176.72.76:8080/api/weiboimage/0a4bdbb3eff672f6ef5f811f13cf65ab.jpg"},
			FilesUrls: []string{"http://54.176.72.76:8080/api/weiboimage/0a4bdbb3eff672f6ef5f811f13cf65ab.jpg", "http://54.176.72.76:8080/api/weiboimage/0a4bdbb3eff672f6ef5f811f13cf65ab.jpg"},
			OriginUrl: "aaa.com",
		},
		CrawledAt:      &timestamppb.Timestamp{},
		CrawlerIp:      "123",
		CrawlerVersion: "123",
		IsTest:         false,
	}

	encodedBytes, err := proto.Marshal(&origin)
	if err != nil {
		log.Fatalln("Failed to encode:", err)
	}

	str := b64.StdEncoding.EncodeToString(encodedBytes)

	result, err := svc.SendMessage(&sqs.SendMessageInput{
		DelaySeconds: aws.Int64(10),
		MessageBody:  aws.String(str),
		QueueUrl:     qURL.QueueUrl,
	})

	if err != nil {
		fmt.Println("Failed to send message:", err)
		return
	}

	fmt.Println("Success", *result.MessageId)
}
