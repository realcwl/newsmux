package main

import (
	"context"
	"flag"
	"fmt"
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
		Text:   "恒指收涨0.32%，科技、可选消费板块涨幅居前 恒指收涨0.32%，恒生科技指数涨1.56%。科技、可选消费板块涨幅居前，比亚迪电子涨近10%，小鹏汽车涨超10%。电子烟概念爆发，思摩尔国际涨超14%。地产股分化，中国恒大涨近10%。",
		Length: 128,
	})
	if err != nil {
		log.Fatalln("fail to call deduplicator: ", err)
	}
	fmt.Println("hashing 1:", res.Binary)

	res2, err := client.GetSimHash(context.TODO(), &protocol.GetSimHashRequest{
		Text:   "恒指收涨0.32%，科技、可选消费板块领涨，比亚迪电子涨近10%，小鹏汽车涨超10%。电子烟概念爆发，思摩尔国际涨超14%。地产股分化，中国恒大涨近10%。",
		Length: 128,
	})
	if err != nil {
		log.Fatalln("fail to call deduplicator: ", err)
	}
	fmt.Println("hashing 2:", res.Binary)

	count := 0
	for i := 0; i < len(res.Binary); i++ {
		if res.Binary[i] != res2.Binary[i] {
			count++
		}
	}
	fmt.Println("distance: ", count)
}
