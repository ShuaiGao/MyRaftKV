package node

import (
	"MyRaft/logger"
	"MyRaft/node/rpc"
	"fmt"
	"go.uber.org/zap"
	"sync"
	"time"
)

type DBServiceConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}
type NodeConfig struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	LogPath string `json:"log"`
}

func (nc *NodeConfig) Address() string {
	return fmt.Sprintf("%s:%d", nc.Host, nc.Port)
}

type WorkNode struct {
	id          int
	cfg         NodeConfig
	term        uint
	commitIndex uint // commit 索引
	acceptIndex uint // accept 索引
}

type Item struct {
	Index uint   `json:"index"`
	Term  uint   `json:"term"`
	Log   string `json:"log"`
}

type Node struct {
	*rpc.UnimplementedElectionServiceServer
	*rpc.UnimplementedDBServiceServer
	Id            string
	electing      bool
	lock          sync.RWMutex
	Term          uint      // 任期
	leaderId      string    // leader Id
	heartBeat     time.Time // 心跳记录
	state         NodeState // 节点状态
	OathAcceptNum int       // 同意选举的数量
	config        []NodeConfig
	configIndex   uint
	OtherNodeList []*WorkNode
	ItemList      []*Item
	CommitIndex   uint
	AcceptIndex   uint
	PreTerm       uint // 任期
}

func (n *Node) GetItemByIndex(index uint) *Item {
	for _, item := range n.ItemList {
		if item.Index == index {
			return item
		}
	}
	return nil
}

func (n *Node) GetPreItem(item *Item) *Item {
	var pre *Item
	for _, v := range n.ItemList {
		if v.Index == item.Index {
			return pre
		}
		pre = v
	}
	return &Item{Index: 0, Term: 0, Log: ""}
}
func (n *Node) GetTailItem() *Item {
	if len(n.ItemList) == 0 {
		return &Item{Index: 0, Term: 0, Log: ""}
	}
	return n.ItemList[len(n.ItemList)-1]
}
func (n *Node) SetState(state NodeState) {
	n.lock.Lock()
	defer n.lock.Unlock()
	//logger.Logger().Info("切换状态", zap.Int("pre_state", int(n.state)), zap.Int("after_state", int(state)))
	n.state = state
}
func (n *Node) GetState() NodeState {
	n.lock.Lock()
	defer n.lock.Unlock()
	return n.state
}
func (n *Node) GetConfigIndex() uint {
	return n.configIndex
}
func (n *Node) GetMyConfig() NodeConfig {
	return n.config[n.configIndex]
}

var thisNode *Node

func CreateNode(nodeCfgList []NodeConfig, index int) *Node {
	thisNode = &Node{
		UnimplementedElectionServiceServer: nil,
		UnimplementedDBServiceServer:       nil,
		Id:                                 fmt.Sprintf("%d", index),
		electing:                           false,
		lock:                               sync.RWMutex{},
		Term:                               0,
		leaderId:                           "",
		heartBeat:                          time.Time{},
		state:                              StateFollower,
		OathAcceptNum:                      0,
		config:                             nodeCfgList,
		configIndex:                        uint(index),
		OtherNodeList:                      []*WorkNode{},
		ItemList:                           []*Item{{Index: 0, Term: 0, Log: "init"}},
		CommitIndex:                        0,
		AcceptIndex:                        0,
	}
	for i, cfg := range nodeCfgList {
		if i == index {
			continue
		}
		thisNode.OtherNodeList = append(thisNode.OtherNodeList, &WorkNode{id: index, cfg: cfg, commitIndex: 0, acceptIndex: 0})
	}
	return thisNode
}

func GetNode() *Node {
	return thisNode
}

const TimeoutNanSecond = int64(time.Second * 5)
const IntervalNanSecond = time.Millisecond * 2000

func (n *Node) Start() {
	n.heartBeat = time.Now()
	for {
		time.Sleep(IntervalNanSecond)
		n.loop()
	}
}
func (n *Node) loop() {
	//logger.Logger().Info("node_loop", zap.Uint("term", n.Term), zap.Int("state", int(n.GetState())))
	if n.GetState() == StateCandidate {
		// 选举过程
		n.election()
	} else if n.GetState() == StateLeader {
		//logger.Logger().Info("Leader 广播心跳")
		go n.leaderSendHeart()
	} else {
		// 心跳
		dis := time.Since(n.heartBeat)
		if dis.Nanoseconds() > TimeoutNanSecond {
			// 超时
			//n.OtherNodeList = []WorkNode{}
			n.timeout()
		}
	}
}

func (n *Node) leaderSendHeart() {
	for _, node := range n.OtherNodeList {
		n.SendHeart(node)
	}
	n.CheckCommitId()
}
func (n *Node) timeout() {
	logger.Logger().Info("心跳超时，切换到候选人状态")
	n.SetState(StateCandidate)
	n.OathAcceptNum = 0
}

func (n *Node) election() {
	n.OathAcceptNum = 1
	n.Term++
	for _, node := range n.OtherNodeList {
		n.SendElectionRequest(node.cfg)
	}
	if n.OathAcceptNum*2 > len(n.config) {
		// 选举成功
		logger.Logger().Info("选举成功",
			zap.Uint("term", n.Term),
			zap.String("leader", n.Id),
			zap.Int("oath_accept_num", n.OathAcceptNum))
		n.SetState(StateLeader)
		go n.leaderSendHeart()
	} else {
		logger.Logger().Info("选举失败",
			zap.Uint("term", n.Term),
			zap.String("Leader", n.Id),
			zap.Int("oath_accept_num", n.OathAcceptNum))
		// 选举失败,重新选举
	}
}

func (n *Node) CheckCommitId() {
	acceptCommitCounter := 0
	for _, otherNode := range n.OtherNodeList {
		if otherNode.commitIndex > n.CommitIndex {
			acceptCommitCounter++
		}
	}
	logger.Logger().Info("check_commit_id", zap.Int("commit_counter", acceptCommitCounter), zap.Uint("commit_index", n.CommitIndex))
	if acceptCommitCounter*2 > len(n.OtherNodeList)+1 {
		n.CommitIndex++
		logger.Logger().Info("commit_entry", zap.Uint("commit_index", n.CommitIndex))
	}
}
func (n *Node) CheckAcceptId() {
	acceptAcceptCounter := 0
	for _, otherNode := range n.OtherNodeList {
		logger.Logger().Info("check_accept_id", zap.Int("accept_counter", acceptAcceptCounter), zap.Uint("node_accept_index", n.AcceptIndex))
		if otherNode.acceptIndex > n.AcceptIndex {
			acceptAcceptCounter++
		}
	}
	logger.Logger().Info("check_accept_id", zap.Int("accept_counter", acceptAcceptCounter), zap.Uint("accept_index", n.AcceptIndex))
	if acceptAcceptCounter*2 > len(n.OtherNodeList)+1 {
		n.AcceptIndex++
		logger.Logger().Info("accept_entry", zap.Uint("accept_index", n.AcceptIndex))
	}
}

//func (n *Node) AppendLog(c echo.Context) error {
//	if n.GetState() != StateLeader {
//		return errors.New("node isn't leader")
//	}
//	log := c.QueryParam("log")
//	item := &Item{Index: n.configIndex, Term: n.Term, Log: log}
//	preItem := n.GetTailItem()
//	n.Append(context.Background(), &rpc.AppendEntryRequest{
//		PreIndex: uint32(preItem.Index),
//		PreTerm:  uint32(preItem.Term),
//		Entry:    &rpc.Entry{Term: uint32(n.Term), Index: uint32(n.configIndex), Value: log}})
//	for _, node := range n.OtherNodeList {
//		n.SendAppendEntry(node, item)
//	}
//	return c.JSON(http.StatusOK, &AppendLogRet{code: 0, msg: "ok"})
//}
