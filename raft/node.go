package raft

import (
	"MyRaft/logger"
	raftPB "MyRaft/raft/raftpb"
	"errors"
)

var (
	emptyState = raftPB.HardState{}
	ErrStopped = errors.New("raft: stopped")
)

type Node interface {
	Tick()
	// Campaign 引起节点状态转换到candidate节点并且开始leader的竞争
	//Campaign(ctx context.Context) error
	//Propose(ctx context.Context, data []byte) error
	//ProposeConfChange(ctx context.Context, cc raftPB.ConfChange) error
	//Step(ctx context.Context, msg raftPB.RaftMessage)
	//Ready() <-chan Ready
	//Advance()
	//ApplyConfChange(cc raftPB.ConfChange)*raftPB.ConfState
	//TransferLeadership(ctx context.Context, lead, transferee uint64)

	// ReportUnreachable 报告结果吗最后一次发送失败
	//ReportUnreachable(id uint64)
	Stop()
}
type SoftState struct {
	Lead      uint64
	RaftState StateType
}
type Peer struct {
	ID      uint64
	Context []byte
}

func StartNode(c *Config, peers []Peer) Node {
	if len(peers) == 0 {
		panic("no peers given; use RestartNode instead")
	}
	rn, err := NewRawNode(c)
	if err != nil {
		panic(err)
	}
	err = rn.Bootstrap(peers)
	if err != nil {

	}
	n := newNode(rn)
	go n.run()
	return n
}

// Ready 压缩entries 和消息，等待存储的读取、提交或发送给其他节点
// Ready 的所有字段都是只读的
type Ready struct {
	Entries  []*raftPB.Entry
	Messages []*raftPB.RaftMessage
}
type node struct {
	recvc  chan raftPB.RaftMessage
	confc  chan raftPB.ConfChange
	readyc chan Ready
	tickc  chan struct{}
	done   chan struct{}
	stop   chan struct{}
	rn     *RawNode
}

func newNode(rn *RawNode) *node {
	return &node{}
}
func (n *node) Tick() {
	select {
	case n.tickc <- struct{}{}:
	case <-n.done:
	default:
		logger.Sugar().Warnf("%v tick error", n.rn.raft.id)
	}
}
func (n *node) Stop() {
	select {
	case n.stop <- struct{}{}:
	case <-n.done:
		return
	}
	// 阻塞住，直到done 被run方法捕获
	<-n.done
}
func (n *node) run() {
	r := n.rn.raft
	for {
		select {
		case m := <-n.recvc:
			if pr := r.trk.Progress[m.From]; pr != nil || !IsResponseMsg(m.Type) {
				r.Step(m)
			}
			//case cc:= <-n.confc:
			//	_, okBefore := r.trk.Progress[r.id]
			//	cs := r.app
		case <-n.tickc:
			n.rn.raft.tick()
		case <-n.stop:
			close(n.done)
			return
		}
	}
}
