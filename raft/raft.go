package raft

import (
	"MyRaft/raft/quorum"
	raftPB "MyRaft/raft/raftpb"
	"MyRaft/raft/tracker"
	"errors"
	"fmt"
	"golang.org/x/exp/rand"
	"sort"
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

type CampaignType string

const (
	campaignPreElection CampaignType = "CampaignPreElection"
	campaignElection    CampaignType = "CampaignElection"
	campaignTransfer    CampaignType = "CampaignTransfer"
)

type ReadOnlyOption int

const (
	// ReadOnlySafe 确保了只读请求的线性一致性，他通过与法定节点通讯保证
	// 它是默认和建议的设置
	ReadOnlySafe ReadOnlyOption = iota
	// ReadOnlyLeaseBased 通过leader领导权保证只读请求的线性一致性
	// 他可能会受时钟漂移的影响
	// 如果始终飘逸没有限制，leader会保持领导权更长时间（时钟可能会不受任何限制地后移或暂停）
	// 在这种情况下，读索引是不安全的
	ReadOnlyLeaseBased
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

	maxMsgSize         uint64
	maxUncommittedSize uint64
	readStates         []ReadState
	readOnly           *readOnly

	state                     StateType
	electionTimeout           int
	heartbeatTimeout          int
	randomizedElectionTimeout int
	disableProposalForwarding bool
	electionElapsed           int
	heartbeatElapsed          int
	checkQuorum               bool
	preVote                   bool
	lead                      uint64
	leadTransferee            uint64 // leader主动转换时的目标id
	tick                      func()
	step                      stepFunc
	logger                    Logger
	raftLog                   *raftLog
	msgs                      []raftPB.RaftMessage
	trk                       tracker.ProgressTracker
	// pendingReadIndexMessage 用于存储MsgReadIndex消息
	//
	pendingReadIndexMessages []raftPB.RaftMessage
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
func numOfPendingConf(ents []raftPB.Entry) int {
	n := 0
	for i := range ents {
		if ents[i].Type == raftPB.EntryType_EntryConfChange {
			n++
		}
	}
	return n
}
func (r *raft) sendTimeoutNow(to uint64) {
	r.send(raftPB.RaftMessage{To: to, Type: raftPB.MessageType_MsgTimeoutNow})
}
func (r *raft) send(m raftPB.RaftMessage) {
	if m.From == None {
		m.From = r.id
	}
	if m.Type == raftPB.MessageType_MsgVote || m.Type == raftPB.MessageType_MsgVoteResp {
		if m.Term == 0 {
			panic(fmt.Sprintf("term should be set when sending %s", m.Type))
		}
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
func (r *raft) appendEntry(es ...*raftPB.Entry) (accepted bool) {
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

// promotable 表明状态机是否可以提升为leader
// 当raft id在progress list里面的时候为true
func (r *raft) promotable() bool {
	trk := r.trk.Progress[r.id]
	return trk != nil && !trk.IsLearner && !r.raftLog.hasPendingSnapshot()
}
func (r *raft) pastElectionTimeout() bool {
	return r.electionElapsed >= r.randomizedElectionTimeout
}
func (r *raft) responseToReadIndexReq(req raftPB.RaftMessage, readIndex uint64) raftPB.RaftMessage {
	if req.From == None || req.From == r.id {
		r.readStates = append(r.readStates, ReadState{
			Index:      readIndex,
			RequestCtx: req.Entries[0].Data,
		})
		return raftPB.RaftMessage{}
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
func (r *raft) becomePreCandidate() {

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
		if r.preVote {
			r.hup(campaignPreElection)
		} else {
			r.hup(campaignElection)
		}

	case raftPB.MessageType_MsgVote, raftPB.MessageType_MsgPreVote:

	}
	return nil
}
func stepLeader(r *raft, m raftPB.RaftMessage) error {
	switch m.Type {
	case raftPB.MessageType_MsgBeat:
		r.bcastHeartbeat()
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
		for i := range m.Entries {
			e := m.Entries[i]
			var cc *raftPB.ConfChange
			if e.Type == raftPB.EntryType_EntryConfChange {

			}
			if cc != nil {

			}
		}
		if !r.appendEntry(m.Entries...) {
			return ErrProposalDropped
		}
	case raftPB.MessageType_MsgReadIndex:
		if r.trk.IsSingleton() {
			if resp := r.responseToReadIndexReq(m, r.raftLog.committed); resp.To != None {
				r.send(resp)
			}
			return nil
		}
		if !r.committedEntryInCurrentTerm() {
			r.pendingReadIndexMessages = append(r.pendingReadIndexMessages, m)
			return nil
		}
		sendMsgReadIndexResponse(r, m)
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
			r.logger.Debugf("%x received MsgAppResp(rejected, hint: (index %d, term %d)) from %x for index %d",
				r.id, m.RejectHint, m.LogTerm, m.From, m.Index)
			nextProbeIdx := m.RejectHint
			if m.LogTerm > 0 {
				nextProbeIdx = r.raftLog.findConflictByTerm(m.RejectHint, m.LogTerm)
			}
			if pr.MaybeDecrTo(m.Index, nextProbeIdx) {
				r.logger.Debugf("%x decreased progress of %x to [%s]", r.id, m.From, pr)
				if pr.State == tracker.StateReplicate {
					pr.BecomeProbe()
				}
				r.sendAppend(m.From)
			}
		} else {
			oldPaused := pr.IsPaused()
			if pr.MaybeUpdate(m.Index) {
				switch {
				case pr.State == tracker.StateProbe:
					pr.BecomeReplicate()
				case pr.State == tracker.StateSnapshot && pr.Match >= pr.PendingSnapshot:
					r.logger.Debugf("%x recovered from needing snapshot, resumed sending replication message to %x [%s]", r.id, m.From, pr)
					pr.BecomeProbe()
					pr.BecomeReplicate()
				case pr.State == tracker.StateReplicate:
					pr.Inflights.FreeLE(m.Index)
				}

				if r.maybeCommit() {
					// 当前任期的term已经提交，现在回复读请求是安全的
					releasePendingReadIndexMessages(r)
					r.bcastAppend()
				} else if oldPaused {
					// 如果当前线程之前已暂停，
					// TODO
					r.sendAppend(m.From)
				}
				for r.maybeSendAppend(m.From, false) {
				}
				if m.From == r.leadTransferee && pr.Match == r.raftLog.lastIndex() {
					r.logger.Infof("%x sent MsgTimeoutNow to %x after received MsgAppResp", r.id, m.From)
					r.sendTimeoutNow(m.From)
				}
			}
		}
	case raftPB.MessageType_MsgHeartbeatResp:
		pr.RecentActive = true
		pr.ProbeSent = false

		if pr.State == tracker.StateReplicate && pr.Inflights.Full() {
			pr.Inflights.FreeFirstOne()
		}
	case raftPB.MessageType_MsgSnapStatus:
		if pr.State != tracker.StateSnapshot {
			return nil
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
		if leadTransferee == r.id {
			r.logger.Debugf("%x is already leader. Ignored transferring leadership to self", r.id)
			return nil
		}
		r.logger.Infof("%x [term %d] starts to transfer leadership to %x", r.id, r.Term, leadTransferee)
		r.electionElapsed = 0
		r.leadTransferee = leadTransferee
		if pr.Match == r.raftLog.lastIndex() {
			r.sendTimeoutNow(leadTransferee)
			r.logger.Infof("%x sends MsgTimeoutNow to %x immediately as %x already has up-to-date log", r.id, leadTransferee, leadTransferee)
		} else {
			// TODO
			r.sendAppend(leadTransferee)
		}
	}
	return nil
}
func (r *raft) maybeCommit() bool {
	mci := r.trk.Committed()
	return r.raftLog.maybeCommit(mci, r.Term)
}
func (r *raft) sendAppend(to uint64) {
	r.maybeSendAppend(to, true)
}
func (r *raft) maybeSendAppend(to uint64, sendIfEmpty bool) bool {
	pr := r.trk.Progress[to]
	if pr.IsPaused() {
		return false
	}
	m := raftPB.RaftMessage{}
	m.To = to

	term, errt := r.raftLog.term(pr.Next - 1)
	ents, erre := r.raftLog.entries(pr.Next, r.maxMsgSize)
	if len(ents) == 0 && !sendIfEmpty {
		return false
	}
	if errt != nil || erre != nil {
		if !pr.RecentActive {
			r.logger.Debugf("ignore sending snapshot to %x since it is not recently actice", to)
			return false
		}

		m.Type = raftPB.MessageType_MsgSnap
		snapshot, err := r.raftLog.snapshot()
		if err != nil {
			if err == ErrSnapshotTemporarilyUnavailable {
				r.logger.Debugf("%x failed to send snapshot to %x because snapshot is temporarily unavailable", r.id, to)
				return false
			}
			panic(err)
		}
		if IsEmptySnap(snapshot) {
			panic("need non-empty snapshot")
		}
		m.Snapshot = snapshot
		sindex, sterm := snapshot.Metadata.Index, snapshot.Metadata.Term
		r.logger.Debugf("%x [firstindex: %d, commit:%d] sent snapshot[index: %d, term: %d] to %x [%s]",
			r.id, r.raftLog.firstIndex(), r.raftLog.committed, sindex, sterm, to, pr)
		pr.BecomeSnapshot(sindex)
		r.logger.Debugf("%x paused sending replication messages to %x [%s]", r.id, to, pr)
	} else {
		m.Type = raftPB.MessageType_MsgApp
		m.Index = pr.Next - 1
		m.LogTerm = term
		m.Entries = ents
		m.Commit = r.raftLog.committed
		if n := len(m.Entries); n != 0 {
			switch pr.State {
			case tracker.StateReplicate:
				last := m.Entries[n-1].Index
				pr.OptimisticUpdate(last)
				pr.Inflights.Add(last)
			case tracker.StateProbe:
				pr.ProbeSent = true
			default:
				r.logger.Panicf("%x is sending append in unhandled state %s", r.id, pr.State)
			}
		}
	}
	r.send(m)
	return true
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
	case raftPB.MessageType_MsgProp:
		if r.lead == None {
			r.logger.Infof("%x no leader at term %d; dropping proposal", r.id, r.Term)
			return ErrProposalDropped
		} else if r.disableProposalForwarding {
			r.logger.Infof("%x not forwarding to leader %x at term %d; dtopping proposal", r.id, r.lead, r.Term)
			return ErrProposalDropped
		}
		m.To = r.lead
		r.send(m)
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
		// 领导权转移不需要使用pre-vote，即使r.preVote为true
		// 自己知道不是从网络分区中恢复的，因此没有必要多一个额为请求循环
		r.hup(campaignTransfer)
	case raftPB.MessageType_MsgReadIndex:
		if r.lead == None {
			r.logger.Infof("%x no leader at term %d; dropping index reading msg", r.id, r.Term)
			return nil
		}
		// 将消息转发给leader处理
		m.To = r.lead
		r.send(m)
	case raftPB.MessageType_MsgReadIndexResp:
		if len(m.Entries) != 1 {
			r.logger.Errorf("%x invalid format of MsgReadIndexResp from %x, entries count: %d", r.id, m.From, len(m.Entries))
			return nil
		}
		r.readStates = append(r.readStates, ReadState{Index: m.Index, RequestCtx: m.Entries[0].Data})
	}
	return nil

}
func (r *raft) hup(t CampaignType) {
	if r.state == StateLeader {
		r.logger.Debugf("%x ignoring MsgHup because already leader", r.id)
		return
	}
	if !r.promotable() {
		r.logger.Warnf("%x is unpromotable and can not campaign", r.id)
		return
	}
	ents, err := r.raftLog.slice(r.raftLog.applied+1, r.raftLog.committed+1, noLimit)
	if err != nil {
		r.logger.Panicf("unexpect error getting unapplied entries (%v)", err)
	}
	if n := numOfPendingConf(ents); n != 0 && r.raftLog.committed > r.raftLog.applied {
		r.logger.Warnf("%x cannot campaign at term %d since there are still %d pending configuration changes to apply", r.id, r.Term, n)
		return
	}
	r.logger.Infof("%x is starting a new election at term %d", r.id, r.Term)
	r.campaign(t)
}
func (r *raft) campaign(t CampaignType) {
	if !r.promotable() {
		r.logger.Warnf("%x is unpromotable; campaign should have been called", r.id)
	}
	var term uint64
	var voteMsg raftPB.MessageType
	if t == campaignElection {
		r.becomePreCandidate()
		voteMsg = raftPB.MessageType_MsgPreVote
		// PreVote RPCs are sent for the next term before we're incremented r.Term.
		term = r.Term + 1
	} else {
		r.becomeCandidate()
		voteMsg = raftPB.MessageType_MsgVote
		term = r.Term
	}
	if _, _, res := r.poll(r.id, voteRespMsgType(voteMsg), true); res == quorum.VoteWon {
		if t == campaignPreElection {
			r.campaign(campaignElection)
		} else {
			r.becomeLeader()
		}
		return
	}
	var ids []uint64
	{
		idMap := r.trk.Voters.IDs()
		ids = make([]uint64, 0, len(idMap))
		for id := range idMap {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	}
	for _, id := range ids {
		if id == r.id {
			continue
		}
		r.logger.Infof("%x [logterm: %d, index: %d] sent %s request to %x at term %d",
			r.id, r.raftLog.lastTerm(), r.raftLog.lastIndex(), voteMsg, id, r.Term)
		var ctx []byte
		if t == campaignTransfer {
			ctx = []byte(t)
		}
		r.send(raftPB.RaftMessage{Term: term, To: id, Type: voteMsg, Index: r.raftLog.lastIndex(), LogTerm: r.raftLog.lastTerm(), Context: ctx})
	}
}
func (r *raft) poll(id uint64, t raftPB.MessageType, v bool) (granted int, rejected int, result quorum.VoteResult) {
	if v {
		r.logger.Infof("%x received %s from %x at term %d", r.id, t, id, r.Term)
	} else {
		r.logger.Infof("%x received %s rejection from %x at term %d", r.id, t, id, r.Term)
	}
	r.trk.RecordVote(id, v)
	return r.trk.TallyVotes()
}
func (r *raft) handleAppendEntries(m raftPB.RaftMessage) {
	if m.Index < r.raftLog.committed {
		r.send(raftPB.RaftMessage{To: m.From, Type: raftPB.MessageType_MsgAppResp, Index: r.raftLog.committed})
		return
	}
	if mLastIndex, ok := r.raftLog.maybeAppend(m.Index, m.LogTerm, m.Commit, m.Entries...); ok {
		r.send(raftPB.RaftMessage{To: m.From, Type: raftPB.MessageType_MsgAppResp, Index: mLastIndex})
	} else {
		r.logger.Debugf("%x [logterm: %d, index %d] rejected MsgApp [logterm: %d, index: %d] from %x",
			r.id, r.raftLog.zeroTermOnErrCompacted(r.raftLog.term(m.Index)), m.Index, m.LogTerm, m.Index, m.From)
		hintIndex := min(m.Index, r.raftLog.lastIndex())
		hintIndex = r.raftLog.findConflictByTerm(hintIndex, m.LogTerm)
		hintTerm, err := r.raftLog.term(hintIndex)
		if err != nil {
			panic(fmt.Sprintf("term(%d) must be valid, but got %v", hintIndex, err))
		}
		r.send(raftPB.RaftMessage{
			To:         m.From,
			Type:       raftPB.MessageType_MsgAppResp,
			Index:      m.Index,
			Reject:     true,
			RejectHint: hintIndex,
			LogTerm:    hintTerm,
		})
	}
}
func (r *raft) handleHeartbeat(m raftPB.RaftMessage) {
	r.raftLog.commitTo(m.Commit)
	r.send(raftPB.RaftMessage{To: m.From, Type: raftPB.MessageType_MsgHeartbeatResp, Context: m.Context})
}
func (r *raft) handleSnapshot(m raftPB.RaftMessage) {
	sindex, sterm := m.Snapshot.Metadata.Index, m.Snapshot.Metadata.Term
	if r.restore(m.Snapshot) {
		r.logger.Infof("%x [commit: %d] restored snapshot [index: %d, term: %d]",
			r.id, r.raftLog.committed, sindex, sterm)
		r.send(raftPB.RaftMessage{To: m.From, Type: raftPB.MessageType_MsgAppResp, Index: r.raftLog.lastIndex()})
	} else {
		r.logger.Infof("%x [commit: %d] ignore snapshot [index: %d, term: %d]",
			r.id, r.raftLog.committed, sindex, sterm)
		r.send(raftPB.RaftMessage{To: m.From, Type: raftPB.MessageType_MsgAppResp, Index: r.raftLog.committed})
	}
}
func (r *raft) restore(s *raftPB.Snapshot) bool {
	if s.Metadata.Index <= r.raftLog.committed {
		return false
	}
	if r.state != StateFollower {
		r.logger.Warnf("%s attemped to restore snapshot as leader; should never happen", r.id)
		r.becomeFollower(r.Term+1, None)
		return false
	}
	found := false
	cs := s.Metadata.ConfState
	for _, set := range [][]uint64{cs.Voters, cs.Learners, cs.VotersOutgoing} {
		for _, id := range set {
			if id == r.id {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		r.logger.Warnf("%x attempted to restore snapshot but it is not in the ConfState %v; should never happen", r.id, cs)
		return false
	}
	if r.raftLog.matchTerm(s.Metadata.Index, s.Metadata.Term) {
		r.logger.Infof("%x [commit: %d, lastindex:%d, lastterm: %d] fast-forwarded commit to snapshot [index: %d, term: %d]",
			r.id, r.raftLog.committed, r.raftLog.lastIndex(), r.raftLog.lastTerm(), s.Metadata.Index, s.Metadata.Term)
		r.raftLog.commitTo(s.Metadata.Index)
	}
	r.raftLog.restore(s)

	// Reset the configuration and add the (potentially updated) peers in anew
	r.trk = tracker.MakeProgressTracker(r.trk.MaxInflight)
	//cfg, trk, err := confchange
	// TODO

	return true
}

// sendHeartbeat 给其他工作线程发送心跳
func (r *raft) sendHeartbeat(to uint64, ctx []byte) {
	commit := min(r.trk.Progress[to].Match, r.raftLog.committed)
	m := raftPB.RaftMessage{
		To:      to,
		Type:    raftPB.MessageType_MsgHeartbeat,
		Commit:  commit,
		Context: ctx,
	}
	r.send(m)
}
func (r *raft) bcastHeartbeat() {
	lastCtx := r.readOnly.lastPendingRequestCtx()
	if len(lastCtx) == 0 {
		r.bcastHeartbeatWithCtx(nil)
	} else {
		r.bcastHeartbeatWithCtx([]byte(lastCtx))
	}
}

// bcastAppend 发送RPC
func (r *raft) bcastAppend() {
	r.trk.Visit(func(id uint64, _ *tracker.Progress) {
		if id == r.id {
			return
		}
		r.sendAppend(id)
	})
}
func (r *raft) bcastHeartbeatWithCtx(ctx []byte) {
	r.trk.Visit(func(id uint64, _ *tracker.Progress) {
		if id == r.id {
			return
		}
		r.sendHeartbeat(id, ctx)
	})
}

func (r *raft) committedEntryInCurrentTerm() bool {
	return r.raftLog.zeroTermOnErrCompacted(r.raftLog.term(r.raftLog.committed)) == r.Term
}
func sendMsgReadIndexResponse(r *raft, m raftPB.RaftMessage) {
	switch r.readOnly.option {
	case ReadOnlySafe:
		r.readOnly.addRequest(r.raftLog.committed, m)
		r.readOnly.recvAck(r.id, m.Entries[0].Data)
		r.bcastHeartbeatWithCtx(m.Entries[0].Data)
	case ReadOnlyLeaseBased:
		if resp := r.responseToReadIndexReq(m, r.raftLog.committed); resp.To != None {
			r.send(resp)
		}
	}
}
func releasePendingReadIndexMessages(r *raft) {
	if !r.committedEntryInCurrentTerm() {
		r.logger.Error("pending MsgReadIndex should be realease only after first commit in current term")
		return
	}

	msgs := r.pendingReadIndexMessages
	r.pendingReadIndexMessages = nil

	for _, m := range msgs {
		sendMsgReadIndexResponse(r, m)
	}
}
