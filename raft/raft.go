package raft

import (
	raftPB "MyRaft/raft/raftpb"
	"MyRaft/raft/tracker"
	"MyRaft/state"
	"errors"
	"fmt"
	"golang.org/x/exp/rand"
)

const None uint64 = 0

type Config struct {
	ID uint64
	// 选举超时时长，follower在该时长内未收到leader的任何消息时，将转换为candidate
	// 建议 ElectionTick = 10 * HeartbeatTick, 以避免不必要的leader切换
	ElectionTick  int
	HeartbeatTick int
}

func (c *Config) validate() error {
	if c.ID == None {
		return errors.New("cannot use none as id")
	}
	if c.HeartbeatTick <= 0 {
		return errors.New("heartbeat tick must be greater than 0")
	}
	if c.ElectionTick <= c.HeartbeatTick {
		return errors.New("election tick must be greater than heartbeat tick")
	}
	return nil
}

type stepFunc func(r *raft, m raftPB.RaftMessage) error
type raft struct {
	id                        uint64
	state                     state.StateEnum
	electionTimeout           int
	heartbeatTimeout          int
	randomizedElectionTimeout int
	electionElapsed           int
	heartbeatElapsed          int
	checkQuorum               bool
	Term                      uint64
	Vote                      uint64
	lead                      uint64
	leadTransferee            uint64 // leader主动转换时的目标id
	tick                      func()
	step                      stepFunc
	logger                    Logger
	raftLog                   *raftLog
	msgs                      []raftPB.RaftMessage
	trk                       tracker.ProgressTracker
}

func (r *raft) resetRandomizedElectionTimeout() {
	r.randomizedElectionTimeout = r.electionTimeout + rand.Intn(r.electionTimeout)
}
func (r *raft) sendTimeoutNow(to uint64) {
	r.send(raftPB.RaftMessage{To: to, Type: raftPB.MessageType_MsgTimeoutNow})
}
func (r *raft) send(m raftPB.RaftMessage) {
	if m.From == None {
		m.From = r.id
	}
	if m.Type == raftPB.MessageType_MsgVote || m.Type == raftPB.MessageType_MsgVoteResp {

	} else {
		if m.Term != 0 {
			panic(fmt.Sprintf("term should not be set when sending %s (was %d)", m.Type, m.Term))
		}
	}
	r.msgs = append(r.msgs, m)
}
func (r *raft) reset(term uint64) {
	if r.Term != term {
		r.Term = term
		r.Vote = None
	}
	r.lead = None
	r.electionElapsed = 0
	r.heartbeatElapsed = 0
	r.resetRandomizedElectionTimeout()
	r.abortLeaderTransfer()
}
func (r *raft) appendEntry(es ...raftPB.Entry) (accepted bool) {
	li := r.raftLog.lastIndex()
	for i := range es {
		es[i].Term = r.Term
		es[i].Index = li + 1 + uint64(i)
	}
	li = r.raftLog.append(es...)
	return true
}
func (r *raft) abortLeaderTransfer() {
	r.leadTransferee = None
}
func (r *raft) promotable() bool {
	return true
}
func (r *raft) pastElectionTimeout() bool {
	return r.electionElapsed >= r.randomizedElectionTimeout
}

func newRaft(c *Config) *raft {
	if err := c.validate(); err != nil {
		panic(err.Error())
	}
	r := &raft{
		id:               c.ID,
		electionTimeout:  c.ElectionTick,
		heartbeatTimeout: c.HeartbeatTick,
	}

	r.becomeFollower(r.Term, None)
	return r
}
func (r *raft) tickElection() {
	r.electionElapsed++

	if r.promotable() && r.pastElectionTimeout() {
		r.electionElapsed = 0
		if err := r.Step(raftPB.RaftMessage{From: r.id, Type: raftPB.MessageType_MsgHup}); err != nil {
			r.logger.Debugf("error occurred during election: %v", err)
		}
	}
}
func (r *raft) tickHeartBeat() {
	r.heartbeatElapsed++
	r.electionElapsed++
	if r.electionElapsed >= r.electionTimeout {
		r.electionElapsed = 0
		if r.checkQuorum {
			if err := r.Step(raftPB.RaftMessage{From: r.id, Type: raftPB.MessageType_MsgCheckQuorum}); err != nil {
				r.logger.Debugf("error occurred during checking sending heartbeat: %v", err)
			}
		}
		// 如果当前leader不能在选举超时时转移leader，再次成为leader
		if r.state == state.StateLeader && r.leadTransferee != None {
			r.abortLeaderTransfer()
		}
	}
	if r.state != state.StateLeader {
		return
	}
	if r.heartbeatElapsed >= r.heartbeatTimeout {
		r.heartbeatElapsed = 0
		if err := r.Step(raftPB.RaftMessage{From: r.id, Type: raftPB.MessageType_MsgBeat}); err != nil {
			r.logger.Debugf("error occurred during checking sending heartbeat: %v", err)
		}
	}
}
func (r *raft) becomeFollower(term uint64, lead uint64) {
	r.step = stepFollower
	r.reset(term)
	r.tick = r.tickElection
	r.lead = lead
	r.state = state.StateFollower
	r.logger.Infof("%x become follower at term %d", r.id, r.Term)
}
func (r *raft) becomeCandidate() {
	if r.state == state.StateLeader {
		panic("invalid transition [leader -> candidate]")
	}
	r.step = stepCandidate
	r.reset(r.Term + 1)
	r.tick = r.tickElection
	r.Vote = r.id
	r.state = state.StateCandidate
	r.logger.Infof("%x became candidate at term %d", r.id, r.Term)
}
func (r *raft) becomeLeader() {
	if r.state == state.StateFollower {
		panic("invalid transition [follower -> leader]")
	}
	r.step = stepLeader
	r.reset(r.Term)
	r.tick = r.tickHeartBeat
	r.lead = r.id
	r.state = state.StateLeader

	// TODO 暂时忽略

	r.logger.Infof("%x become leader at term %d", r.id, r.Term)
}
func (r *raft) Step(m raftPB.RaftMessage) error {
	return nil
}
func stepLeader(r *raft, m raftPB.RaftMessage) error {
	return nil
}
func stepCandidate(r *raft, m raftPB.RaftMessage) error {
	return nil
}
func stepFollower(r *raft, m raftPB.RaftMessage) error {
	return nil

}
func (r *raft) handleAppendEntries(m raftPB.RaftMessage) {

}
func (r *raft) handleHeartbeat(m raftPB.RaftMessage) {

}
func (r *raft) handleSnapshot(m raftPB.RaftMessage) {

}
func (r *raft) sendHeartbeat(to uint64, ctx []byte) {
}
