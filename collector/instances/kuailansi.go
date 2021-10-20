package collector_instances

import (
	"bytes"
	"encoding/json"
	"io"

	Collector "github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/protocol"
)

const KUAILANSI_URI = "http://m.fbecn.com/24h/news_fbe0406.json?newsid=0"

type KuailansiApiCrawler struct {
	Sink Collector.CollectedDataSink
}

type KuailansiPost struct {
	NewsId   string `json:"newsID"`
	Time     string `json:"time"`
	Content  string `json:"content"`
	Level    string `json:"Level"`
	Type     string `json:"Type"`
	Keywords string `json:"Keywords"`
}

type KuailansiApiResponse struct {
	List []struct {
		NewsId   string `json:"newsID"`
		Time     string `json:"time"`
		Content  string `json:"content"`
		Level    string `json:"Level"`
		Type     string `json:"Type"`
		Keywords string `json:"Keywords"`
	} `json:"list"`
	NextPage string `json:"nextpage"`
}

// print the contents of the obj

func (k KuailansiApiCrawler) CollectAndPublish(task *protocol.PanopticTask) {
	httpClient := Collector.HttpClient{}
	res, err := httpClient.Get(KUAILANSI_URI)
	if err != nil {
		task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		return
	}
	body, _ := io.ReadAll(res.Body)
	// Remove BOM before parsing, see https://en.wikipedia.org/wiki/Byte_order_mark for details.
	body = bytes.TrimPrefix(body, []byte("\xef\xbb\xbf"))
	resp := &KuailansiApiResponse{}
	err = json.Unmarshal(body, resp)
	if err != nil {
		panic(err)
	}
	Collector.PrettyPrint(resp)
}
