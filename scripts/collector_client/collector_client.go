package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Luismorlan/newsmux/protocol"
	"google.golang.org/grpc"
)

func main() {

	conn, err := grpc.Dial("localhost:5001", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()

	client := protocol.NewDataCollectClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	response, err := client.Collect(
		ctx,
		&protocol.PanopticJob{
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
				{
					TaskId:          "456",
					DataCollectorId: protocol.PanopticTask_COLLECTOR_JINSHI,
					TaskParams: &protocol.TaskParams{
						HeaderParams: []*protocol.KeyValuePair{},
						Cookies:      []*protocol.KeyValuePair{},
						SourceId:     "456",
						SubSources: []*protocol.PanopticSubSource{
							{
								Name:       "要闻",
								Type:       protocol.PanopticSubSource_KEYNEWS,
								ExternalId: "2",
							},
						},
					},
				},
			}})
	if err != nil {
		log.Fatalf("Failed when Collect(): %v", err)
	}
	fmt.Println(response.String())
}
