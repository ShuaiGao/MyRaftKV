package raft

import raftPB "MyRaft/raft/raftpb"

type ReadState struct {
	Index      uint64
	RequestCtx []byte
}
type readIndexStatus struct {
	req   raftPB.RaftMessage
	index uint64
	// 这里不会记录false
	// 但是这里使用起来比map[uint64]struct{]更方便，因为又API quorum.VoteResult
	// 如果这里对性能足够敏感（也许）,quorum.VoteResult 可以修改为一个更接近CommittedIndex的api
	acks map[uint64]bool
}
type readOnly struct {
	option           ReadOnlyOption
	pendingReadIndex map[string]*readIndexStatus
	readIndexQueue   []string
}

func newReadOnly(option ReadOnlyOption) *readOnly {
	return &readOnly{
		option:           option,
		pendingReadIndex: make(map[string]*readIndexStatus),
	}
}

// addRequest 添加一个制度请求到readonly
// index 是raft状态机收到消息时的committed index
// m 是来自本地或远程只读请求的原始数据
func (ro *readOnly) addRequest(index uint64, m raftPB.RaftMessage) {
	s := string(m.Entries[0].Data)
	if _, ok := ro.pendingReadIndex[s]; ok {
		return
	}
	ro.pendingReadIndex[s] = &readIndexStatus{index: index, req: m, acks: make(map[uint64]bool)}
	ro.readIndexQueue = append(ro.readIndexQueue, s)
}

func (ro *readOnly) recvAck(id uint64, context []byte) map[uint64]bool {
	rs, ok := ro.pendingReadIndex[string(context)]
	if !ok {
		return nil
	}
	rs.acks[id] = true
	return rs.acks
}

// lastPendingRequestCtx 返回最后添加的只读请求上下文
func (ro *readOnly) lastPendingRequestCtx() string {
	if len(ro.readIndexQueue) == 0 {
		return ""
	}
	return ro.readIndexQueue[len(ro.readIndexQueue)-1]
}
