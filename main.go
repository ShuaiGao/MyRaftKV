package main

import (
	"MyRaft/logger"
	"MyRaft/node"
	election_grpc "MyRaft/node/rpc"
	"encoding/json"
	"flag"
	"fmt"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"os"
)

type Config struct {
	Hello     string            `json:"hello"`
	NodeList  []node.NodeConfig `json:"node_list"`
	LogConfig zap.Config        `json:"log"`
}

func loadConfig() *Config {
	jsonFile, err := os.Open("config.json")
	if err != nil {
		fmt.Println(err.Error())
	}
	defer jsonFile.Close()
	decoder := json.NewDecoder(jsonFile)
	c := &Config{}
	err = decoder.Decode(c)
	if err != nil {
		panic("load json config error:" + err.Error())
	}
	//for k, v := range c.NodeList{
	//	fmt.Println(k, v)
	//}
	fmt.Printf("config: %#v", c)
	return c
}

var IndexFlag = flag.Int("i", -1, "index os node")

func main() {
	flag.Parse()
	config := loadConfig()
	if *IndexFlag < 0 || *IndexFlag >= len(config.NodeList) {
		panic("index out of range")
	}
	logger.CreateLogger(config.LogConfig)
	log := logger.Logger()
	defer log.Sync()
	nodeConfig := config.NodeList[*IndexFlag]
	log.Sugar().Info("Start Node listen :%d", nodeConfig.Port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", nodeConfig.Port))
	if err != nil {
		panic("端口监听失败：" + err.Error())
		return
	}

	s := grpc.NewServer()
	node := node.CreateNode(config.NodeList, *IndexFlag)
	go node.Start()
	election_grpc.RegisterElectionServiceServer(s, node)
	reflection.Register(s)
	err = s.Serve(lis)
	if err != nil {
		panic("开启服务失败：" + err.Error())
		return
	}
}
