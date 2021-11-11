package main

import (
	"context"
	"flag"
	"log"

	"github.com/Luismorlan/newsmux/protocol"
	"google.golang.org/grpc"
)

var (
	serverAddr = flag.String("addr", "localhost:50051", "The server address in the format of host:port")
)

func main() {
	flag.Parse()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := protocol.NewDeduplicatorClient(conn)

	res, err := client.GetSimHash(context.TODO(), &protocol.GetSimHashRequest{
		Text:   "谷歌C股价首次站上3000美元关口，市值逼近2万亿美元，现涨超1%。",
		Length: 128,
	})
	if err != nil {
		log.Fatalln("fail to call deduplicator: ", err)
	}
	log.Println("success with response: ", res)
}
