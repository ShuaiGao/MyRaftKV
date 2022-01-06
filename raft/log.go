package raft

import raftPB "MyRaft/raft/raftpb"

type raftLog struct {
	committed uint64
	// applied 是应用到状态机的最大log id
	// applied <= committed
	applied uint64
	storage Storage
}

func (rl *raftLog) lastIndex() uint64 {
	return 0
}
func (rl *raftLog) append(es ...raftPB.Entry) (lastIndex uint64) {
	return rl.lastIndex()
}
