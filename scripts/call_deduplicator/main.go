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
		Text:   "【#美国通胀率创30年来新高#，民众直言喝不起咖啡了】11月10日美国劳工部表示，10月该国消费者价格指数（CPI）同比涨幅达6.2%，增幅创1990年12月以来新高。有美国民众表示，为了给车加油，已经不喝咖啡了。澎湃视频的微博视频 ",
		Length: 128,
	})
	if err != nil {
		log.Fatalln("fail to call deduplicator: ", err)
	}
	fmt.Println("hashing 1:", res.Binary)

	res2, err := client.GetSimHash(context.TODO(), &protocol.GetSimHashRequest{
		Text:   "国家烟草专卖局原党组成员、中央纪委原派驻国家烟草专卖局纪检组组长潘家华严重违纪违法被开除党籍。（央视）",
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
