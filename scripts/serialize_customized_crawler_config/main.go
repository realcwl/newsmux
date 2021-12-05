package main

import (
	"fmt"

	"github.com/Luismorlan/newsmux/protocol"
	"google.golang.org/protobuf/encoding/prototext"
)

// use this script to generate a request you can use to send in Lambda->Test
func main() {
	clsUrl := "https://www.cls.cn/telegraph"
	clsBaseSelector := ".telegraph-list"
	clsTitleRelativeSelector := ".telegraph-content-box span:not(.telegraph-time-box) > strong"
	clsContentRelativeSelector := ".telegraph-content-box span:not(.telegraph-time-box)"
	clsImageRelativeSelector := ".telegraph-images-box > img"

	config := protocol.PanopticConfig{
		Name:            "test",
		DataCollectorId: protocol.PanopticTask_COLLECTOR_CUSTOMIZED,
		TaskParams: &protocol.TaskParams{
			Params: &protocol.TaskParams_CustomizedCrawlerTaskParams{
				CustomizedCrawlerTaskParams: &protocol.CustomizedCrawlerParams{
					CrawlUrl:                &clsUrl,
					BaseSelector:            &clsBaseSelector,
					TitleRelativeSelector:   &clsTitleRelativeSelector,
					ContentRelativeSelector: &clsContentRelativeSelector,
					ImageRelativeSelector:   &clsImageRelativeSelector,
				},
			},
		},
		TaskSchedule: &protocol.TaskSchedule{
			StartImmediatly: true,
			Schedule: &protocol.TaskSchedule_Routinely{
				Routinely: &protocol.Routinely{
					EveryMilliseconds: 1000000,
				},
			},
		},
		DryRun: true,
	}

	bytes, err := prototext.Marshal(&config)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("serialized:")
	fmt.Println(string(bytes))

	var panopticConfig protocol.PanopticConfig
	if err := prototext.Unmarshal(bytes, &panopticConfig); err != nil {
		fmt.Printf("can't unmarshal panoptic config error %v", err)
		return
	}

	fmt.Println("de-serialized:")
	fmt.Println(panopticConfig)
}
