package main

import (
	"fmt"

	collector_hander "github.com/Luismorlan/newsmux/collector/handler"
	"github.com/Luismorlan/newsmux/protocol"
	"google.golang.org/protobuf/encoding/prototext"
)

// for local test zsxq collector
func main() {
	job := protocol.PanopticJob{
		Tasks: []*protocol.PanopticTask{{
			TaskId:          "123",
			DataCollectorId: protocol.PanopticTask_COLLECTOR_ZSXQ,
			TaskParams: &protocol.TaskParams{
				HeaderParams: []*protocol.KeyValuePair{
					{Key: "authority", Value: "api.zsxq.com"},
					{Key: "sec-ch-ua", Value: "\"Chromium\";v=\"94\", \"Google Chrome\";v=\"94\", \";Not A Brand\";v=\"99\""},
					{Key: "x-version", Value: "2.9.0"},
					{Key: "x-signature", Value: "2b845b1363061db96b173654f549e5541da04372"},
					{Key: "sec-ch-ua-mobile", Value: "?0"},
					{Key: "user-agent", Value: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.81 Safari/537.36"},
					{Key: "accept", Value: "application/json, text/plain, */*"},
					{Key: "x-timestamp", Value: "1634714964"},
					{Key: "x-request-id", Value: "f7ad02840-98a0-69f1-5999-204abf0bf73"},
					{Key: "sec-ch-ua-platform", Value: "\"macOS\""},
					{Key: "origin", Value: "https,Value://wx.zsxq.com"},
					{Key: "sec-fetch-site", Value: "same-site"},
					{Key: "sec-fetch-mode", Value: "cors"},
					{Key: "sec-fetch-dest", Value: "empty"},
					{Key: "referer", Value: "https,Value://wx.zsxq.com/"},
					{Key: "accept-language", Value: "en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7"},
				},
				Cookies: []*protocol.KeyValuePair{
					{Key: "abtest_env", Value: "product"},
					{Key: "zsxq_access_token", Value: "7BC4A8E7-CAD4-3ECD-3E69-9D68C6E933C4_B52ED7FD31557037"},
				},
				SourceId: "a0fb27a2-03a3-4428-9a22-5bbd9ff738b7",
				SubSources: []*protocol.PanopticSubSource{
					{
						Name:       "48418215854848",
						Type:       protocol.PanopticSubSource_USERS,
						ExternalId: "48418215854848",
					},
					{
						Name:       "828588881882",
						Type:       protocol.PanopticSubSource_USERS,
						ExternalId: "828588881882",
					},
				},
				Params: &protocol.TaskParams_ZsxqTaskParams{
					ZsxqTaskParams: &protocol.ZsxqTaskParams{
						CountPerRequest: 20,
					},
				},
			},
			TaskMetadata: &protocol.TaskMetadata{},
		},
		},
	}
	t := prototext.Format(&job)
	fmt.Println("=========== prototext: ==========")
	fmt.Println(string(t))
	fmt.Println("=========== prototext end ==========")
	var handler collector_hander.DataCollectJobHandler

	fmt.Println("=========== starting collect ==========")
	err := handler.Collect(&job)
	if err != nil {
		panic(err)
	}
}
