package raft

import (
	raftPB "MyRaft/raft/raftpb"
	"MyRaft/raft/tracker"
	"errors"
	"fmt"
	"golang.org/x/exp/rand"
)

const None uint64 = 0

type StateType uint64

var stmap = [...]string{
	"StateFollower",
	"StateCandidate",
	"StateLeader",
	"StatePreCandidate",
}

const (
	StateFollower StateType = iota
	StateCandidate
	StateLeader
	StatePreCandidate
	numStates
)

func (st StateType) String() string {
	return stmap[st]
}

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
	id   uint64
	Term uint64
	Vote uint64

	readStates []ReadState

	state                     StateType
	electionTimeout           int
	heartbeatTimeout          int
	randomizedElectionTimeout int
	electionElapsed           int
	heartbeatElapsed          int
	checkQuorum               bool
	lead                      uint64
	leadTransferee            uint64 // leader主动转换时的目标id
	tick                      func()
	step                      stepFunc
	logger                    Logger
	raftLog                   *raftLog
	msgs                      []raftPB.RaftMessage
	trk                       tracker.ProgressTracker
}

func (r *raft) hasLeader() bool {
	return r.lead != None
}
func (r *raft) softState() *SoftState {
	return &SoftState{
		Lead:      r.lead,
		RaftState: r.state,
	}
}
func (r *raft) hardState() raftPB.HardState {
	return raftPB.HardState{
		Term:   r.Term,
		Vote:   r.Vote,
		Commit: r.raftLog.committed,
	}
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
func (r *raft) advance(rd Ready) {

}
func (r *raft) promotable() bool {
	return true
}
func (r *raft) pastElectionTimeout() bool {
	return r.electionElapsed >= r.randomizedElectionTimeout
}
func (r *raft) responseToReadIndexReq(req raftPB.RaftMessage, readIndex uint64) raftPB.RaftMessage {
	if req.From == None || req.From == r.id {
	}
	return raftPB.RaftMessage{
		Type:    raftPB.MessageType_MsgReadIndexResp,
		To:      req.From,
		Index:   readIndex,
		Entries: req.Entries,
	}
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
		if r.state == StateLeader && r.leadTransferee != None {
			r.abortLeaderTransfer()
		}
	}
	if r.state != StateLeader {
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
	r.state = StateFollower
	r.logger.Infof("%x become follower at term %d", r.id, r.Term)
}
func (r *raft) becomeCandidate() {
	if r.state == StateLeader {
		panic("invalid transition [leader -> candidate]")
	}
	r.step = stepCandidate
	r.reset(r.Term + 1)
	r.tick = r.tickElection
	r.Vote = r.id
	r.state = StateCandidate
	r.logger.Infof("%x became candidate at term %d", r.id, r.Term)
}
func (r *raft) becomeLeader() {
	if r.state == StateFollower {
		panic("invalid transition [follower -> leader]")
	}
	r.step = stepLeader
	r.reset(r.Term)
	r.tick = r.tickHeartBeat
	r.lead = r.id
	r.state = StateLeader

	// TODO 暂时忽略

	r.logger.Infof("%x become leader at term %d", r.id, r.Term)
}
func (r *raft) Step(m raftPB.RaftMessage) error {
	switch {
	case m.Term == 0:
		// 本地消息
	case m.Term > r.Term:
		if m.Type == raftPB.MessageType_MsgPreVote || m.Type == raftPB.MessageType_MsgVote {

		}
		switch {
		case m.Type == raftPB.MessageType_MsgPreVote:
		// 回复PreVote消息时，不改变term
		case m.Type == raftPB.MessageType_MsgPreVoteResp && !m.Reject:
		default:
			r.logger.Infof("%x [term: %d] received a %s message with higher term from %x [term :%d]", r.id, r.Term, m.Type, m.From, m.Term)
			if m.Type == raftPB.MessageType_MsgApp || m.Type == raftPB.MessageType_MsgHeartbeat || m.Type == raftPB.MessageType_MsgSnap {
				r.becomeFollower(m.Term, m.From)
			} else {
				r.becomeFollower(m.Term, None)
			}
		}
	case m.Term < r.Term:
	}
	switch m.Type {
	case raftPB.MessageType_MsgHup:
		//if r.preVote
	case raftPB.MessageType_MsgVote, raftPB.MessageType_MsgPreVote:

	}
	return nil
}
func stepLeader(r *raft, m raftPB.RaftMessage) error {
	switch m.Type {
	case raftPB.MessageType_MsgBeat:
		r.becomeLeader()
		return nil
	case raftPB.MessageType_MsgCheckQuorum:
		return nil
	case raftPB.MessageType_MsgProp:
		if len(m.Entries) == 0 {
			r.logger.Panicf("%s stepped empty MsgProp", r.id)
		}
		if r.trk.Progress[r.id] == nil {
			return ErrProposalDropped
		}
		if r.leadTransferee != None {
			r.logger.Debugf("%x [term %d] transfer leadership to %x is in progress; dropping proposal", r.id, r.Term, r.leadTransferee)
			return ErrProposalDropped
		}
	//for i:= range m.Entries{
	//	e := &m.Entries[i]
	//	var cc raftPB.ConfChange
	//}
	case raftPB.MessageType_MsgReadIndex:
		if r.trk.IsSingleton() {
			if resp := r.responseToReadIndexReq(m, r.raftLog.committed); resp.To != None {
				r.send(resp)
			}
			return nil
		}
		return nil
	}

	pr := r.trk.Progress[m.From]
	if pr == nil {
		r.logger.Debugf("%s no progress available for %x", r.id, m.From)
		return nil
	}
	switch m.Type {
	case raftPB.MessageType_MsgAppResp:
		pr.RecentActive = true
		if m.Reject {

		} else {

		}
	case raftPB.MessageType_MsgHeartbeatResp:
		pr.RecentActive = true
		pr.ProbeSent = false

		if pr.State == tracker.StateReplicate && pr.Inflights.Full() {
			pr.Inflights.FreeFirstOne()
		}
	case raftPB.MessageType_MsgUnreachable:
		if pr.State == tracker.StateReplicate {
			pr.BecomeProbe()
		}
		r.logger.Debugf("%x failed to send message to %x because it is unreachable [%s]", r.id, m.From, pr)
	case raftPB.MessageType_MsgTransferLeader:
		if pr.IsLearner {
			r.logger.Debugf("%x is learner. Ignored transferring leadership", r.id)
			return nil
		}
		leadTransferee := m.From
		lastLeadTransferee := r.leadTransferee
		if lastLeadTransferee != None {
			if lastLeadTransferee == leadTransferee {
				r.logger.Infof("%x [term %d] transfer leadership to %x is in progress, ignores request to same node %x", r.id, r.Term, leadTransferee, leadTransferee)
				return nil
			}
			r.abortLeaderTransfer()
			r.logger.Infof("%x [term %d] abort previous Transferring leadership to %x", r.id, r.Term, lastLeadTransferee)
		}
	}
	return nil
}

var ErrStepLocalMsg = errors.New("raft: cannot step raft local message")
var ErrProposalDropped = errors.New("raft proposal dropped")

func stepCandidate(r *raft, m raftPB.RaftMessage) error {
	var myVoteRespType raftPB.MessageType
	if r.state == StatePreCandidate {
		myVoteRespType = raftPB.MessageType_MsgPreVoteResp
	} else {
		myVoteRespType = raftPB.MessageType_MsgVoteResp
	}
	switch m.Type {
	case raftPB.MessageType_MsgProp:
		r.logger.Infof("%x no leader at term %d; dropping proposal", r.id, r.Term)
		return ErrProposalDropped
	case raftPB.MessageType_MsgApp:
		r.becomeFollower(m.Term, m.From)
		r.handleAppendEntries(m)
	case raftPB.MessageType_MsgHeartbeat:
		r.becomeFollower(m.Term, m.From)
		r.handleHeartbeat(m)
	case raftPB.MessageType_MsgSnap:
		r.becomeFollower(m.Term, m.From)
		r.handleSnapshot(m)
	case myVoteRespType:
	//gr, rj, res := r.poo
	case raftPB.MessageType_MsgTimeoutNow:
		r.logger.Debugf("%x [term %d state %v] ignored MsgTimeoutNow from %x", r.id, r.Term, r.state, m.From)
	}
	return nil
}
func stepFollower(r *raft, m raftPB.RaftMessage) error {
	switch m.Type {
	case raftPB.MessageType_MsgApp:
		r.electionElapsed = 0
		r.lead = m.From
	case raftPB.MessageType_MsgHeartbeat:
		r.electionElapsed = 0
		r.lead = m.From
		r.handleHeartbeat(m)
	case raftPB.MessageType_MsgSnap:
		r.electionElapsed = 0
		r.lead = m.From
		r.handleSnapshot(m)
	case raftPB.MessageType_MsgTransferLeader:
		if r.lead == None {
			r.logger.Infof("%x no leader at term %d; dropping leader transfer msg", r.id, r.Term)
			return nil
		}
		m.To = r.lead
		r.send(m)
	case raftPB.MessageType_MsgTimeoutNow:
		r.logger.Infof("%x [term %d] received MsgTimeoutNow from %x and starts an election to get leadership.", r.id, r.Term, m.From)
	}
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
