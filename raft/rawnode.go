package raft

import (
	raftPB "MyRaft/raft/raftpb"
	"errors"
)

var ErrStepPeerNotFound = errors.New("raft: cannot step as peer not found")

type RawNode struct {
	raft       *raft
	prevSoftSt *SoftState
	preHardSt  raftPB.HardState
}

func NewRawNode(config *Config) (*RawNode, error) {
	r := newRaft(config)
	rn := &RawNode{
		raft: r,
	}
	return rn, nil
}

func (rn *RawNode) Tick() {
	rn.raft.tick()
}
func (rn *RawNode) TickQuiesced() {
	rn.raft.electionElapsed++
}
func (rn *RawNode) Campaign() error {
	return rn.raft.Step(raftPB.RaftMessage{
		Type: raftPB.MessageType_MsgHup,
	})
}

// Propose 提议将data 添加到raft日志中
func (rn *RawNode) Propose(data []byte) error {
	return rn.raft.Step(raftPB.RaftMessage{
		Type: raftPB.MessageType_MsgProp,
		From: rn.raft.id,
		Entries: []*raftPB.Entry{
			{Data: data},
		},
	})
}

func (rn *RawNode) ProposeConfChange(cc raftPB.ConfChange) error {
	//m, err := config
	return nil
}
func (rn *RawNode) Step(m raftPB.RaftMessage) error {
	if IsLocalMsg(m.Type) {
		return ErrStepLocalMsg
	}
	if pr := rn.raft.trk.Progress[m.From]; pr != nil || !IsResponseMsg(m.Type) {
		return rn.raft.Step(m)
	}
	return ErrStepPeerNotFound
}

//func (rn *RawNode) Ready() Ready {
//
//}
func (rn *RawNode) TransferLeader(transferee uint64) {
	_ = rn.raft.Step(raftPB.RaftMessage{Type: raftPB.MessageType_MsgTransferLeader, From: transferee})
}

func (rn *RawNode) ReadIndex(rctx []byte) {
	_ = rn.raft.Step(raftPB.RaftMessage{Type: raftPB.MessageType_MsgReadIndex, Entries: []*raftPB.Entry{{Data: rctx}}})
}
