package main

import (
	"fmt"
	"log"
	"net"

	"github.com/Luismorlan/newsmux/protocol"
	. "github.com/Luismorlan/newsmux/utils"
	"github.com/Luismorlan/newsmux/utils/dotenv"
	. "github.com/Luismorlan/newsmux/utils/log"

	"github.com/Luismorlan/newsmux/collector"
	"google.golang.org/grpc"
)

func init() {
	Log.Info("data collector initialized")
}

func cleanup() {
	CloseProfiler()
	CloseTracer()
	Log.Info("data collector shutdown")
}

func main() {
	defer cleanup()
	if err := dotenv.LoadDotEnvs(); err != nil {
		panic(err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 5001))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var handler collector.DataCollectServerHandler
	grpcServer := grpc.NewServer()
	protocol.RegisterDataCollectServer(grpcServer, handler)

	Log.Info("Started Collector Server..")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}
