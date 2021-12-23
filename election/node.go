package election

import (
	election_grpc "MyRaft/election/rpc"
	"MyRaft/logger"
	"context"
	"fmt"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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

type Node struct {
	id            string
	electing      bool
	lock          sync.RWMutex
	Term          uint      // 任期
	heartBeat     time.Time // 心跳记录
	state         NodeState // 节点状态
	role          NodeRole  // 节点角色
	OathAcceptNum int       // 同意选举的数量
	config        []NodeConfig
	configIndex   int
	otherNodeList []Node
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

const TimeoutNanSecond = int64(time.Millisecond) * 100

func (n *Node) Heart(ctx context.Context, in *election_grpc.HeartRequest) (*election_grpc.HeartReply, error) {
	return &election_grpc.HeartReply{}, nil
}
func (n *Node) Oath(ctx context.Context, in *election_grpc.OathRequest) (*election_grpc.OathReply, error) {
	accept := false
	n.heartBeat = time.Now()
	inTerm := uint(in.Term)
	if n.Term < inTerm {
		n.Term = inTerm
		accept = true
	} else if n.Term == inTerm {
		accept = false
	} else {
		accept = false
	}
	logger.Logger().Info("rpc Oath",
		zap.String("from", in.NodeId),
		zap.Uint32("term", in.Term),
		zap.Bool("accept", accept))
	return &election_grpc.OathReply{Accept: accept}, nil
}
func (n *Node) Start() {
	for {
		time.Sleep(time.Second)
		n.loop()
	}
}
func (n *Node) loop() {
	//logger.Logger().Info("node loop")
	if n.GetState() == NodeStateElecting {
		// 选举过程
		n.election()
	} else {
		// 心跳
		dis := time.Since(n.heartBeat)
		if dis.Microseconds() > TimeoutNanSecond {
			// 超时
			n.timeout()
		} else {
			n.heartBeat = time.Now()
		}
	}
}

func (n *Node) timeout() {
	logger.Logger().Info("心跳超时，切换到候选人状态")
	n.SetState(NodeStateElecting)
	n.role = NodeRoleCandidate
	n.OathAcceptNum = 0
}

func (n *Node) election() {
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
		n.state = NodeStateWorking
		n.role = NodeRoleLeader
	} else {
		logger.Logger().Info("选举失败",
			zap.Uint("term", n.Term),
			zap.String("Leader", n.id),
			zap.Int("oath_accept_num", n.OathAcceptNum))
		// 选举失败
		// 重新选举
		n.OathAcceptNum = 1
		n.Term++
	}
}

func (n *Node) SendOathRequest(cfg NodeConfig) {
	conn, err := grpc.Dial(cfg.Address(), grpc.WithInsecure())
	if err != nil {
		logger.Logger().Sugar().Fatal("did not connect:%s", cfg.Address())
	}
	defer conn.Close()
	c := election_grpc.NewElectionServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.Oath(ctx, &election_grpc.OathRequest{Term: uint32(n.Term), NodeId: n.id})
	if err != nil {
		logger.Logger().Warn("SendOathRequest_oath_error", zap.String("address", cfg.Address()), zap.Error(err))
		return
	}
	if r.Accept {
		n.OathAcceptNum++
	}
	logger.Logger().Info("SendOathRequest",
		zap.String("vote_id", n.id),
		zap.String("ticket_from", r.GetNodeId()),
		zap.Bool("accept", r.Accept),
		zap.Int("oath_accept_num", n.OathAcceptNum))
}
func (n *Node) SendHeart(node Node) {
	cfg := node.GetMyConfig()
	conn, err := grpc.Dial(cfg.Address(), grpc.WithInsecure())
	if err != nil {
		logger.Logger().Sugar().Fatal("did not connect:%s", cfg.Address())
	}
	defer conn.Close()
	c := election_grpc.NewElectionServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = c.Heart(ctx, &election_grpc.HeartRequest{})
	if err != nil {
		logger.Sugar().Fatal("did not connect:%s", cfg.Address())
	}
}
