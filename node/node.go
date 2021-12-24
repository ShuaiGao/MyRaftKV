package node

import (
	"MyRaft/logger"
	"MyRaft/node/rpc"
	"fmt"
	"go.uber.org/zap"
	"sync"
	"time"
)

type NodeConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func (nc *NodeConfig) Address() string {
	return fmt.Sprintf("%s:%d", nc.Host, nc.Port)
}

type WorkNode struct {
	id          string
	cfg         NodeConfig
	term        uint
	commitIndex uint // commit 索引
	acceptIndex uint // accept 索引
}

type Item struct {
	Index uint   `json:"index"`
	Term  uint   `json:"term"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Node struct {
	*rpc.UnimplementedElectionServiceServer
	id            string
	electing      bool
	lock          sync.RWMutex
	Term          uint      // 任期
	leaderId      string    // leader id
	heartBeat     time.Time // 心跳记录
	state         NodeState // 节点状态
	role          NodeRole  // 节点角色
	OathAcceptNum int       // 同意选举的数量
	config        []NodeConfig
	configIndex   int
	otherNodeList []WorkNode
	ItemList      []*Item
	CommitIndex   uint
}

func (n *Node) GetPreItem() *Item {
	return n.ItemList[len(n.ItemList)-1]
}
func (n *Node) SetState(state NodeState) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.state = state
}
func (n *Node) GetState() NodeState {
	n.lock.Lock()
	defer n.lock.Unlock()
	return n.state
}
func (n *Node) GetMyConfig() NodeConfig {
	return n.config[n.configIndex]
}

var thisNode *Node

func CreateNode(nodeCfgList []NodeConfig, index int) *Node {
	return &Node{id: fmt.Sprintf("%d", index), config: nodeCfgList, configIndex: index}
}

func GetNode() *Node {
	return thisNode
}

const TimeoutNanSecond = int64(time.Second * 2)
const IntervalNanSecond = time.Second

func (n *Node) Start() {
	n.heartBeat = time.Now()
	for {
		time.Sleep(IntervalNanSecond)
		n.loop()
	}
}
func (n *Node) loop() {
	//logger.Logger().Info("node_loop", zap.Uint("term", n.Term), zap.Int("state", int(n.GetState())))
	if n.GetState() == NodeStateElecting {
		// 选举过程
		n.election()
	} else if n.GetState() == NodeStateLeader {
		logger.Logger().Info("Leader 广播心跳")
		go n.leaderSendHeart()
	} else {
		// 心跳
		dis := time.Since(n.heartBeat)
		if dis.Nanoseconds() > TimeoutNanSecond {
			// 超时
			n.timeout()
		}
	}
}

func (n *Node) leaderSendHeart() {
	for _, node := range n.otherNodeList {
		n.SendHeart(node)
	}
}
func (n *Node) timeout() {
	logger.Logger().Info("心跳超时，切换到候选人状态")
	n.SetState(NodeStateElecting)
	n.role = NodeRoleCandidate
	n.OathAcceptNum = 0
}

func (n *Node) election() {
	n.OathAcceptNum = 1
	n.Term++
	for i, node := range n.config {
		if i == n.configIndex {
			continue
		}
		n.SendOathRequest(node)
	}
	if n.OathAcceptNum*2 > len(n.config) {
		// 选举成功
		logger.Logger().Info("选举成功",
			zap.Uint("term", n.Term),
			zap.String("leader", n.id),
			zap.Int("oath_accept_num", n.OathAcceptNum))
		n.SetState(NodeStateLeader)
		n.role = NodeRoleLeader
	} else {
		logger.Logger().Info("选举失败",
			zap.Uint("term", n.Term),
			zap.String("Leader", n.id),
			zap.Int("oath_accept_num", n.OathAcceptNum))
		// 选举失败,重新选举
	}
}
