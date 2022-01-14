package raft

import (
	raftPB "MyRaft/raft/raftpb"
	"fmt"
	"google.golang.org/grpc/benchmark/latency"
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
func (rl *raftLog) entries(i, maxsize uint64) ([]*raftPB.Entry, error) {
	if i > rl.lastIndex() {
		return nil, nil
	}
	return rl.slice(i, rl.lastIndex()+1, maxsize)
}
func (rl *raftLog) slice(lo, hi, maxSize uint64) ([]*raftPB.Entry, error) {
	err := rl.mustCheckOutOfBounds(lo, hi)
	if err != nil {
		return nil, err
	}
	if lo == hi {
		return nil, nil
	}
	var ents []*raftPB.Entry
	if lo < rl.unstable.offset {
		storedEnts, err := rl.storage.Entries(lo, min(hi, rl.unstable.offset), maxSize)
		if err == ErrCompacted {
			return nil, err
		} else if err == ErrUnavailable {
			rl.logger.Panicf("entries[%d:%d) is unavailable from storage", lo, min(hi, rl.unstable.offset))
		} else if err != nil {
			panic(err)
		}
		if uint64(len(storedEnts)) < min(hi, rl.unstable.offset)-lo {
			return storedEnts, nil
		}
		ents = storedEnts
	}
	if hi > rl.unstable.offset {
		unstable := rl.unstable.slice()
	}
	return nil, nil
}

func (rl *raftLog) snapshot() (raftPB.Snapshot, error) {
	if rl.unstable.snapshot != nil {
		return *rl.unstable.snapshot, nil
	}
	return rl.storage.Snapshot()
}
func (rl *raftLog) append(es ...*raftPB.Entry) (lastIndex uint64) {
	return rl.lastIndex()
}
func (rl *raftLog) maybeCommit(maxIndex, term uint64) bool {
	if maxIndex > rl.committed && rl.zeroTermOnErrCompacted(rl.term(maxIndex)) == term {
		rl.commitTo(maxIndex)
		return true
	}
	return false
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
func (rl *raftLog) nextEnts() (ents []*raftPB.Entry) {
	off := max(rl.applied+1, rl.firstIndex())
	if rl.committed+1 > off {
		//ents, err := rl.sli
	}
	return nil
}
func (rl *raftLog) hasNextEnts() bool {
	off := max(rl.applied+1, rl.firstIndex())
	return rl.committed+1 > off
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
func (rl *raftLog) lastTerm() uint64 {
	t, err := rl.term(rl.lastIndex())
	if err != nil {
		rl.logger.Panicf("unexpected error when getting the last term (%v)", err)
	}
	return t
}

func (rl *raftLog) commitTo(tocommit uint64) {
	if rl.committed < tocommit {
		if rl.lastIndex() < tocommit {
			rl.logger.Panicf("tocommit(%d) is out of range [lastIndex(%d)]. Was the raft log corrupted, truncated, or lost?", tocommit, rl.lastIndex())
		}
		rl.committed = tocommit
	}
}

func (rl *raftLog) maybeAppend(index, logTerm, committed uint64, ents ...*raftPB.Entry) (lastNewi uint64, ok bool) {
	if rl.matchTerm(index, logTerm) {
		lastNewi = index + uint64(len(ents))
		ci := rl.findConflict(ents)
		switch {
		case ci == 0:
		case ci <= rl.committed:
			rl.logger.Panicf("entry %d conflict with committed entry [committed(%d)]", ci, rl.committed)
		default:
			offset := index + 1
			rl.append(ents[ci-offset:]...)
		}
		rl.commitTo(min(committed, lastNewi))
		return lastNewi, true
	}
	return 0, false
}

func (rl *raftLog) findConflict(ents []*raftPB.Entry) uint64 {
	for _, ne := range ents {
		if !rl.matchTerm(ne.Index, ne.Term) {
			if ne.Index < rl.lastIndex() {
				rl.logger.Infof("found conflict at index %d [existing term: %d, conflicting term: %d]",
					ne.Index, rl.zeroTermOnErrCompacted(rl.term(ne.Index)), ne.Term)
			}
			return ne.Index
		}
	}
	return 0
}
func (rl *raftLog) findConflictByTerm(index uint64, term uint64) uint64 {
	if li := rl.lastIndex(); index > li {
		rl.logger.Warnf("index(%d) is out of range [0, lastIndex(%d)] in findConflictByTerm", index, li)
		return index
	}
	for {
		logTerm, err := rl.term(index)
		if logTerm <= term || err != nil {
			break
		}
		index--
	}
	return index
}
func (rl *raftLog) term(i uint64) (uint64, error) {
	dummyIndex := rl.firstIndex() - 1
	if i < dummyIndex || i > rl.lastIndex() {
		return 0, nil
	}
	if t, ok := rl.unstable.maybeTerm(i); ok {
		return t, nil
	}
	t, err := rl.storage.Term(i)
	if err == nil {
		return t, nil
	}
	if err == ErrCompacted || err == ErrUnavailable {
		return 0, err
	}
	panic(err)
}
func (rl *raftLog) matchTerm(i, term uint64) bool {
	t, err := rl.term(i)
	if err != nil {
		return false
	}
	return t == term
}

func (rl *raftLog) zeroTermOnErrCompacted(t uint64, err error) uint64 {
	if err == nil {
		return t
	}
	if err == ErrCompacted {
		return 0
	}
	rl.logger.Panicf("unexpexted error (%v)", err)
	return 0
}

func (rl *raftLog) mustCheckOutOfBounds(lo, hi uint64) error {
	if lo > hi {
		rl.logger.Panicf("invalid slice %d > %d", lo, hi)
	}
	fi := rl.firstIndex()
	if lo < fi {
		return ErrCompacted
	}
	length := rl.lastIndex() + 1 - fi
	if hi > fi+length {
		rl.logger.Panicf("slice[%d,%d) out of bound [%d,%d]", lo, hi, fi, rl.lastIndex())
	}
	return nil
}
func (rl *raftLog) restore(s *raftPB.Snapshot) {
	rl.logger.Infof("log [%s] starts to restore snapshot [index: %d, term: %d]", rl, s.Metadata.Index, s.Metadata.Term)
	rl.committed = s.Metadata.Index
	rl.unstable.restore(s)
}
