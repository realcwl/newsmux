package main

import (
	"flag"
	"fmt"

	"github.com/Luismorlan/newsmux/protocol"
	. "github.com/Luismorlan/newsmux/utils/log"
	"google.golang.org/protobuf/encoding/prototext"
)

// use this script to generate a request you can use to send in Lambda->Test
func main() {
	flag.Parse()
	InitLogger()

	job := protocol.PanopticJob{
		Tasks: []*protocol.PanopticTask{
			{
				TaskId:          "123",
				DataCollectorId: protocol.PanopticTask_COLLECTOR_JINSHI,
				TaskParams: &protocol.TaskParams{
					HeaderParams: []*protocol.KeyValuePair{},
					Cookies:      []*protocol.KeyValuePair{},
					SourceId:     "a882eb0d-0bde-401a-b708-a7ce352b7392",
					SubSources: []*protocol.PanopticSubSource{
						{
							Name:       "快讯",
							Type:       protocol.PanopticSubSource_FLASHNEWS,
							ExternalId: "1",
						},
						{
							Name:       "要闻",
							Type:       protocol.PanopticSubSource_KEYNEWS,
							ExternalId: "2",
						},
					},
					Params: &protocol.TaskParams_JinshiTaskParams{
						JinshiTaskParams: &protocol.JinshiTaskParams{
							SkipKeyWords: []string{"【黄金操作策略】"},
						},
					},
				},
			},
		}}
	// bytes, _ := proto.Marshal(&job)
	fmt.Println("serializedJob ", prototext.Format(&job))
}
