package raft

import (
	"MyRaft/logger"
	raftPB "MyRaft/raft/raftpb"
	"context"
	"errors"
)

type SnapshotStatus int

const (
	SnapshotFinish  SnapshotStatus = 1
	SnapshotFailure SnapshotStatus = 2
)

var (
	emptyState = raftPB.HardState{}
	ErrStopped = errors.New("raft: stopped")
)

type Node interface {
	Tick()
	// Campaign 引起节点状态转换到candidate节点并且开始leader的竞争
	//Campaign(ctx context.Context) error
	// Propose 提议添加log，要注意提议可能会被丢弃，且没有通知
	// 因此用户需要处理提议重试任务
	Propose(ctx context.Context, data []byte) error
	ProposeConfChange(ctx context.Context, cc raftPB.ConfChange) error
	// Step 使用收到的消息推进状态机运行，出错后，ctx.Err()h 会被返回
	Step(ctx context.Context, msg raftPB.RaftMessage) error
	// Ready 返回一个通道
	Ready() <-chan Ready
	// Advance 通知Node，程序已经保存进程到
	Advance()
	// ApplyConfChange 应用配置修改
	//ApplyConfChange(cc raftPB.ConfChange) *raftPB.ConfState
	// TransferLeadership 试图转移领导权到受让人
	TransferLeadership(ctx context.Context, lead, transferee uint64)

	// ReadIndex 请求一个读状态
	// 读状态又一个读索引
	// 一旦应用的索引大于读索引，在读索引之前的线性读请求事务可以被安全处理
	// 读状态将附加相同的rctx
	// 注意：请求可能会被丢弃，且没有任何通知，因此用户有责任处理读索引的重试
	ReadIndex(ctx context.Context, rctx []byte) error

	// Status 返回当前raft状态机状态
	Status() Status
	// ReportUnreachable 报告结果吗最后一次发送失败
	ReportUnreachable(id uint64)
	Stop()
}
type SoftState struct {
	Lead      uint64
	RaftState StateType
}

func (s *SoftState) equal(b *SoftState) bool {
	return s.Lead == b.Lead && s.RaftState == b.RaftState
}
func isHardStateEqual(a, b raftPB.HardState) bool {
	return a.Term == b.Term && a.Vote == b.Vote && a.Commit == b.Commit
}
func IsEmptyHardState(st raftPB.HardState) bool {
	return isHardStateEqual(st, emptyState)
}
func IsEmptySnap(sp raftPB.Snapshot) bool {
	//return sp.Metadata.In
	return true
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
	// 当前节点的容易改变的状态
	// 当未更新时，SoftState为nil
	// 不要求使用或者存储SoftState
	*SoftState

	// 节点用于在消息发送前存储到稳定存储的状态
	// 当未更新时，HardState 为空
	raftPB.HardState

	ReadStates []ReadState
	// 在Entries发送前，用于写入稳定存储的Entries
	Entries []raftPB.Entry

	// Snapshot 表示用于写入稳定存储的快照
	Snapshot raftPB.Snapshot
	// CommittedEntries 指定将要提交的entries, 这些entries已经被写入稳定存储
	CommittedEntries []raftPB.Entry
	Messages         []*raftPB.RaftMessage

	// MustSync 表明HardState 和 Entries 需要被同步或异步写入硬盘
	MustSync bool
}

func newReady(r *raft, prevSoftSt *SoftState, preHardSt raftPB.HardState) Ready {
	rd := Ready{
		Entries: r.raftLog.unstableEntries(),
		//CommittedEntries: r.raftLog.ne
	}
	return rd
}

type msgWithResult struct {
	m      raftPB.RaftMessage
	result chan error
}
type node struct {
	propc    chan msgWithResult
	recvc    chan raftPB.RaftMessage
	confc    chan raftPB.ConfChange
	readyc   chan Ready
	advancec chan struct{}
	tickc    chan struct{}
	done     chan struct{}
	stop     chan struct{}
	status   chan chan Status
	rn       *RawNode
}

func newNode(rn *RawNode) *node {
	return &node{
		status: make(chan chan Status),
	}
}
func (n *node) Ready() <-chan Ready {
	return n.readyc
}
func (n *node) Advance() {
	select {
	case n.advancec <- struct{}{}:
	case <-n.done:
	}
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
	var propc chan msgWithResult
	var readyc chan Ready
	var advancec chan struct{}
	var rd Ready

	r := n.rn.raft

	lead := None

	for {
		if advancec != nil {
			readyc = nil
		} else if n.rn.HasReady() {
			rd = n.rn.readyWithoutAccept()
			readyc = n.readyc
		}
		if lead != r.lead {
			if r.hasLeader() {
				if lead == None {
					r.logger.Infof("raft.node: %x elected leader %x at term %d", r.id, r.lead, r.Term)
				} else {
					r.logger.Infof("raft.node:%x changed leader from %x to %x at term %d", r.id, lead, r.lead, r.Term)
				}
				propc = n.propc
			} else {
				r.logger.Infof("raft.node: %x lost leader %x at term %d", r.id, lead, r.Term)
				propc = nil
			}
			lead = r.lead
		}
		select {
		case pm := <-propc:
			m := pm.m
			m.From = r.id
			err := r.Step(m)
			if pm.result != nil {
				pm.result <- err
				close(pm.result)
			}
		case m := <-n.recvc:
			if pr := r.trk.Progress[m.From]; pr != nil || !IsResponseMsg(m.Type) {
				r.Step(m)
			}
			//case cc:= <-n.confc:
			//	_, okBefore := r.trk.Progress[r.id]
			//	cs := r.app
		case <-n.confc:
			// TODO 配置变动
		case <-n.tickc:
			n.rn.Tick()
		case readyc <- rd:
			n.rn.acceptReady(rd)
			advancec = n.advancec
		case <-advancec:
			n.rn.Adbvance(rd)
			rd = Ready{}
			advancec = nil
		case c := <-n.status:
			c <- getStatus(r)
		case <-n.stop:
			close(n.done)
			return
		}
	}
}
func (n *node) ProposeConfChange(ctx context.Context, cc raftPB.ConfChange) error {
	//msg, err := co
	return nil
}
func (n *node) Propose(ctx context.Context, data []byte) error {
	return n.stepWait(ctx, raftPB.RaftMessage{Type: raftPB.MessageType_MsgProp, Entries: []*raftPB.Entry{{Data: data}}})
}
func (n *node) Step(ctx context.Context, m raftPB.RaftMessage) error {
	if IsLocalMsg(m.Type) {
		return nil
	}
	return n.step(ctx, m)
}
func (n node) step(ctx context.Context, m raftPB.RaftMessage) error {
	return n.stepWithWaitOption(ctx, m, false)
}
func (n *node) stepWait(ctx context.Context, m raftPB.RaftMessage) error {
	return n.stepWithWaitOption(ctx, m, true)
}
func (n *node) stepWithWaitOption(ctx context.Context, m raftPB.RaftMessage, wait bool) error {
	if m.Type != raftPB.MessageType_MsgProp {
		select {
		case n.recvc <- m:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-n.done:
			return ErrStopped
		}
	}
	ch := n.propc
	pm := msgWithResult{m: m}
	if wait {
		pm.result = make(chan error, 1)
	}
	select {
	case ch <- pm:
		if !wait {
			return nil
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-n.done:
		return ErrStopped
	}
	select {
	case err := <-pm.result:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-n.done:
		return ErrStopped
	}
	return nil
}
func (n *node) Status() Status {
	c := make(chan Status)
	select {
	case n.status <- c:
		return <-c
	case <-n.done:
		return Status{}
	}
}
func (n *node) ReportUnreachable(id uint64) {
	select {
	case n.recvc <- raftPB.RaftMessage{Type: raftPB.MessageType_MsgUnreachable, From: id}:
	case <-n.done:
	}
}
func (n *node) ReportSnapshot(id uint64, status SnapshotStatus) {
	rej := status == SnapshotFailure
	select {
	case n.recvc <- raftPB.RaftMessage{Type: raftPB.MessageType_MsgSnapStatus, From: id, Reject: rej}:
	case <-n.done:
	}
}
func (n *node) TransferLeadership(ctx context.Context, lead, transferee uint64) {
	select {
	case n.recvc <- raftPB.RaftMessage{Type: raftPB.MessageType_MsgTransferLeader, From: transferee, To: lead}:
	case <-n.done:
	case <-ctx.Done():
	}
}
func (n *node) ReadIndex(ctx context.Context, rctx []byte) error {
	return n.step(ctx, raftPB.RaftMessage{Type: raftPB.MessageType_MsgReadIndex, Entries: []*raftPB.Entry{{Data: rctx}}})
}
