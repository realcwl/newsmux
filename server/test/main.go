package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/slack-go/slack"
)

func main() {
	divSection := slack.NewDividerBlock()
	blocks := []slack.Block{divSection}

	unsubscribeBtnText := slack.NewTextBlockObject("plain_text", "Unsubscribe", false, false)
	unsubscribeBtnEle := slack.NewButtonBlockElement("", "click_me_123", unsubscribeBtnText)

	// unsubscribe section
	blocks = append(blocks, divSection)

	// TODO(boning): this is a demo for unsubscribe, will delete it when we have real channel feed table
	optionSample := slack.NewTextBlockObject("mrkdwn", "*恒大足球* \t _Jamie_ \t `5`", false, false)
	optionSection := slack.NewSectionBlock(optionSample, nil, slack.NewAccessory(unsubscribeBtnEle))

	blocks = append(blocks, optionSection)

	msg := slack.NewBlockMessage(blocks...)
	b, err := json.MarshalIndent(msg, "", "    ")
	if err != nil {
		fmt.Println(err)
		return
	}

	req, err := http.NewRequest("POST", "https://hooks.slack.com/services/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", bytes.NewBuffer(b))
	fmt.Println("1", err)

	client := &http.Client{}

	resp, err := client.Do(req)
	fmt.Println("2", err)

	fmt.Println(resp.StatusCode)

}
