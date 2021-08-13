package main

import (
	"fmt"
	"time"

	. "github.com/Luismorlan/newsmux/publisher/protocol"
	. "github.com/Luismorlan/newsmux/utils"
	"google.golang.org/protobuf/proto"
)

func main() {
	reader, err := NewSQSMessageQueueReader("crawler-publisher-queue", 20)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for {
		// Receive 1 message
		msgs, _ := reader.ReceiveMessages(1)
		if len(msgs) == 0 {
			continue
		}
		msg := msgs[0]

		// Parse data into meaningful structure
		str, _ := msg.Read()
		fmt.Println(str)

		// Process data
		decodedMsg := &CrawlerMessage{}
		if err := proto.Unmarshal([]byte(str), decodedMsg); err != nil {
			fmt.Println("wrong!")
		}
		fmt.Println(decodedMsg)

		// Delete message if the process is successful
		reader.DeleteMessage(msg)

		protectivePause()
	}
}

func protectivePause() {
	time.Sleep(2 * time.Second)
}
