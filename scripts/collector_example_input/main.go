package main

import (
	"fmt"

	"github.com/Luismorlan/newsmux/protocol"
	"google.golang.org/protobuf/proto"
)

// use this script to generate a request you can use to send in Lambda->Test
func main() {
	job := protocol.PanopticJob{
		Tasks: []*protocol.PanopticTask{
			{
				TaskId:          "123",
				DataCollectorId: protocol.PanopticTask_COLLECTOR_JINSHI,
				TaskParams: &protocol.TaskParams{
					HeaderParams: []*protocol.KeyValuePair{},
					Cookies:      []*protocol.KeyValuePair{},
					SourceId:     "123",
					SubSources: []*protocol.PanopticSubSource{
						{
							Name:       "快讯",
							Type:       protocol.PanopticSubSource_FLASHNEWS,
							ExternalId: "1",
						},
					},
				},
			},
		}}
	bytes, _ := proto.Marshal(&job)
	fmt.Println("serializedJob ", bytes)
}
