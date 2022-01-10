package raft

import (
	raftPB "MyRaft/raft/raftpb"
	"fmt"
	"log"
	"math"
)

const noLimit = math.MaxUint64

type raftLog struct {
	storage  Storage
	unstable unstable

	committed uint64
	// applied 是应用到状态机的最大log id
	// applied <= committed
	applied uint64
	logger  Logger
	// maxNextEntsSize 消息合计最大字节数
	maxNextEntsSize uint64
}

func newLog(storage Storage, logger Logger) *raftLog {
	return newLogWithSize(storage, logger, noLimit)
}
func newLogWithSize(storage Storage, logger Logger, maxNextEntsSize uint64) *raftLog {
	if storage == nil {
		log.Panicln("storage must not be nil")
	}
	log := &raftLog{
		storage:         storage,
		logger:          logger,
		maxNextEntsSize: maxNextEntsSize,
	}
	fitstIndex, err := storage.FirstIndex()
	if err != nil {
		panic(err)
	}
	lastIndex, err := storage.LastIndex()
	if err != nil {
		panic(err)
	}
	log.unstable.offset = lastIndex + 1
	log.unstable.logger = logger
	log.committed = fitstIndex - 1
	log.applied = fitstIndex - 1
	return log
}

func (rl *raftLog) String() string {
	return fmt.Sprintf("committed=%d, applied=%d, unstable.offset=%d, len(unstable.Entries)=%d", rl.committed, rl.applied, rl.unstable.offset, len(rl.unstable.entries))
}

func (rl *raftLog) append(es ...raftPB.Entry) (lastIndex uint64) {
	return rl.lastIndex()
}

func (rl *raftLog) hasPendingSnapshot() bool {
	//return l.unsta
	return false
}
func (rl *raftLog) unstableEntries() []raftPB.Entry {
	if len(rl.unstable.entries) == 0 {
		return nil
	}
	return rl.unstable.entries
}
func (rl *raftLog) nextEnts() (ents []raftPB.Entry) {
	off := max(rl.applied+1, rl.firstIndex())
	if rl.committed+1 > off {
		//ents, err := rl.sl
	}
	return nil
}
func (rl *raftLog) hasNextEnts() bool {
	//off := max(rl.applied+1, rl.)
	return false
}
func (rl *raftLog) firstIndex() uint64 {
	if i, ok := rl.unstable.maybeFirstIndex(); ok {
		return i
	}
	index, err := rl.storage.FirstIndex()
	if err != nil {
		panic(err)
	}
	return index
}
func (rl *raftLog) lastIndex() uint64 {
	if i, ok := rl.unstable.maybeLastIndex(); ok {
		return i
	}
	i, err := rl.storage.LastIndex()
	if err != nil {
		panic(err)
	}
	return i
}
