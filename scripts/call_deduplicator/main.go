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

	res2, err := client.GetSimHash(context.TODO(), &protocol.GetSimHashRequest{
		Text:   "感谢关注，这篇置顶帮助您对我的微博内容有个大体了解：  个人关键字：货币和信用体系研究者、西甲球队Eibar股东😂、CFA  一个努力中的Behavioral Macro交易员  微博内容关键字：流动性、中央银行、货币市场、金融市场监管......  我一直认为绝大多数市场参与者对于上述几块内容的理解有所欠缺，正好 ...全文",
		Length: 128,
	})
	if err != nil {
		log.Fatalln("fail to call deduplicator: ", err)
	}

	count := 0
	for i := 0; i < len(res.Binary); i++ {
		if res.Binary[i] != res2.Binary[i] {
			count++
		}
	}
	fmt.Println("distance: ", count)
}
