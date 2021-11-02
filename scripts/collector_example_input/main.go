package main

import (
	"flag"
	"fmt"

	"github.com/Luismorlan/newsmux/protocol"
	. "github.com/Luismorlan/newsmux/utils/log"
	"google.golang.org/protobuf/proto"
)

// use this script to generate a request you can use to send in Lambda->Test
func main() {
	flag.Parse()
	InitLogger()

	job := protocol.PanopticJob{
		Tasks: []*protocol.PanopticTask{{
			TaskId:          "123",
			DataCollectorId: protocol.PanopticTask_COLLECTOR_WEIBO,
			TaskParams: &protocol.TaskParams{
				HeaderParams: []*protocol.KeyValuePair{},
				Cookies:      []*protocol.KeyValuePair{},
				SourceId:     "0129417c-4987-45c9-86ac-d6a5c89fb4f7",
				SubSources: []*protocol.PanopticSubSource{
					{
						Name:       "37度卡农",
						Type:       protocol.PanopticSubSource_USERS,
						ExternalId: "6103268173",
					},
				},
				Params: &protocol.TaskParams_WeiboTaskParams{
					WeiboTaskParams: &protocol.WeiboTaskParams{
						MaxPages: 2,
					},
				},
			},
			TaskMetadata: &protocol.TaskMetadata{
				ConfigName: "test_weibo_config",
			},
		},
		},
	}
	bytes, _ := proto.Marshal(&job)
	// fmt.Println(string(bytes))
	// To send it in Lambda UI, you need to split with comma instead of space
	fmt.Println("{ SerializedJob: ", bytes, "}")
}
