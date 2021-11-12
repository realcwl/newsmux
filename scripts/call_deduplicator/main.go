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
		Text:   "字节跳动关联公司在厦门成立新公司 含房地产经纪业务 天眼查App显示，11月12日，厦门好房有幸信息技术有限公司成立，注册资本2000万，法定代表人为王奉坤，经营范围包括软件开发；广告设计、代理；房地产经纪等。股权穿透图显示，该公司由字节跳动关联公司北京好房有幸信息技术有限公司全资持股。\\n",
		Length: 128,
	})
	if err != nil {
		log.Fatalln("fail to call deduplicator: ", err)
	}
	log.Println("success with response: ", res)
}
