package main

import (
	"MyRaft/db"
	"MyRaft/logger"
	"MyRaft/node"
	election_grpc "MyRaft/node/rpc"
	"MyRaft/raft"
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
	Hello            string                 `json:"hello"`
	NodeList         []node.NodeConfig      `json:"node_list"`
	RedisServiceList []node.DBServiceConfig `json:"redis_service_list"`
	LogConfig        zap.Config             `json:"log"`
	RedisConfig      db.RedisConfig         `json:"redis"`
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

func CreateNodeService(nodeList []node.NodeConfig, index int) {
	nodeConfig := nodeList[index]
	logger.Sugar().Info("Start Node listen :%d", nodeConfig.Port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", nodeConfig.Port))
	if err != nil {
		panic("端口监听失败：" + err.Error())
		return
	}

	s := grpc.NewServer()
	node := node.CreateNode(nodeList, *IndexFlag)
	go node.Start()
	election_grpc.RegisterElectionServiceServer(s, node)
	reflection.Register(s)
	err = s.Serve(lis)
	if err != nil {
		panic("开启服务失败：" + err.Error())
		return
	}
}

func CreateService(nodeIns *node.Node, dbServiceList []node.DBServiceConfig, index int) {
	nodeConfig := dbServiceList[index]
	raft.CreateDB(fmt.Sprintf(":%d", nodeConfig.Port), nodeIns)
	//logger.Sugar().Info("Start DBService listen :%d", nodeConfig.Port)
	//lis, err := net.Listen("tcp", fmt.Sprintf(":%d", nodeConfig.Port))
	//if err != nil {
	//	panic("端口监听失败：" + err.Error())
	//	return
	//}
	//
	//s := grpc.NewServer()
	//election_grpc.RegisterDBServiceServer(s, nodeIns)
	//reflection.Register(s)
	//err = s.Serve(lis)
	//if err != nil {
	//	panic("开启服务失败：" + err.Error())
	//	return
	//}
}

func main() {
	flag.Parse()
	config := loadConfig()
	if *IndexFlag < 0 || *IndexFlag >= len(config.NodeList) {
		panic("index out of range")
	}
	nodeConfig := config.NodeList[*IndexFlag]
	config.LogConfig.OutputPaths = append(config.LogConfig.OutputPaths, nodeConfig.LogPath)
	logger.CreateLogger(config.LogConfig)
	log := logger.Logger()
	defer log.Sync()
	db.ConnectRedis(config.RedisConfig)
	log.Sugar().Info("Redis connect :%s", config.RedisConfig)
	go CreateService(node.GetNode(), config.RedisServiceList, *IndexFlag)
	CreateNodeService(config.NodeList, *IndexFlag)
}
