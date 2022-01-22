package raft

import (
	raftPB "MyRaft/raft/raftpb"
	"errors"
	"sync"
)

var ErrCompacted = errors.New("requested index is unavailable due to compaction")
var ErrUnavailable = errors.New("requested entry at index is unavailable")
var ErrSnapOutOfDate = errors.New("requested index is older than the existing snapshot")
var ErrSnapshotTemporarilyUnavailable = errors.New("snapshot is temporarily unavailable")

type Storage interface {
	InitialState() (raftPB.HardState, raftPB.ConfState, error)
	Entries(lo, hi, maxSize uint64) ([]*raftPB.Entry, error)
	// Term returns the term of entry i, thich must be in the range
	// [FirstIndex()-1, LastIndex()]. The term of the entry before
	// FirstIndex is retained for matching purposes even though the
	// rest of that entry may not be available.
	Term(i uint64) (uint64, error)
	LastIndex() (uint64, error)
	FirstIndex() (uint64, error)
	Snapshot() (raftPB.Snapshot, error)
}

type MemoryStorage struct {
	sync.Mutex
	hardState raftPB.HardState
	snapshot  raftPB.Snapshot
	// ents[i] has raft log position i+snapshot.Metadata.Index
	ents []raftPB.Entry
}

func newMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		ents: make([]raftPB.Entry, 1),
	}
}
func (ms *MemoryStorage) InitialState() (raftPB.HardState, raftPB.ConfState, error) {
	return ms.hardState, *ms.snapshot.Metadata.ConfState, nil
}

// SetHardState saves the current HardState
func (ms *MemoryStorage) SetHardState(st raftPB.HardState) error {
	ms.Lock()
	defer ms.Unlock()
	ms.hardState = st
	return nil
}
func (ms *MemoryStorage) Entries(lo, hi, maxSize uint64) ([]raftPB.Entry, error) {
	ms.Lock()
	defer ms.Unlock()
	offset := ms.ents[0].Index
	if lo < offset {
		return nil, ErrCompacted
	}
	if hi > ms.lastIndex()+1 {
		getLogger().Panicf("entries hi(%d) is out of bound lastindex(%d)", hi, ms.lastIndex())
	}
	if len(ms.ents) == 1 {
		return nil, ErrUnavailable
	}
	ents := ms.ents[lo-offset : hi-offset]
	return limitSize(ents, maxSize), nil
}

func (ms *MemoryStorage) LastIndex() (uint64, error) {
	ms.Lock()
	defer ms.Unlock()
	return ms.lastIndex(), nil
}

func (ms *MemoryStorage) lastIndex() uint64 {
	return ms.ents[0].Index + uint64(len(ms.ents)) - 1
}
func (ms *MemoryStorage) firstIndex() uint64 {
	return ms.ents[0].Index + 1
}
func (ms *MemoryStorage) FirstIndex() (uint64, error) {
	ms.Lock()
	defer ms.Unlock()
	return ms.firstIndex(), nil
}
func (ms *MemoryStorage) Term(i uint64) (uint64, error) {
	ms.Lock()
	defer ms.Unlock()
	offset := ms.ents[0].Index
	if i < offset {
		return 0, ErrCompacted
	}
	if int(i-offset) >= len(ms.ents) {
		return 0, ErrUnavailable
	}
	return ms.ents[i-offset].Term, nil
}

func (ms *MemoryStorage) Snapshot() (raftPB.Snapshot, error) {
	ms.Lock()
	defer ms.Unlock()
	return ms.snapshot, nil
}

func (ms *MemoryStorage) ApplySnapshot(snap raftPB.Snapshot) error {
	ms.Lock()
	defer ms.Unlock()

	msIndex := ms.snapshot.Metadata.Index
	snapIndex := snap.Metadata.Index
	if msIndex >= snapIndex {
		return ErrSnapOutOfDate
	}
	ms.snapshot = snap
	ms.ents = []raftPB.Entry{{Term: snap.Metadata.Term, Index: snap.Metadata.Index}}
	return nil
}

func (ms *MemoryStorage) CreateSnapshot(i uint64, cs *raftPB.ConfState, data []byte) (raftPB.Snapshot, error) {
	ms.Lock()
	defer ms.Unlock()
	if i < ms.snapshot.Metadata.Index {
		return raftPB.Snapshot{}, ErrSnapOutOfDate
	}
	offset := ms.ents[0].Index
	if i > ms.lastIndex() {
		getLogger().Panicf("snapshot %d is out of bound lastindex(%d)", i, ms.lastIndex())
	}

	ms.snapshot.Metadata.Index = i
	ms.snapshot.Metadata.Term = ms.ents[i-offset].Term
	if cs != nil {
		ms.snapshot.Metadata.ConfState = cs
	}
	ms.snapshot.Data = data
	return ms.snapshot, nil
}

// Compact 删除所有小于compactIndex的日志
// 应用需要保证不压缩索引大于raftLog.applied 的日志
func (ms *MemoryStorage) Compact(compactIndex uint64) error {
	ms.Lock()
	defer ms.Unlock()
	offset := ms.ents[0].Index
	if compactIndex <= offset {
		return ErrCompacted
	}
	if compactIndex > ms.lastIndex() {
		getLogger().Panicf("compact %d is out of bound lastindex(%d)", compactIndex, ms.lastIndex())
	}
	i := compactIndex - offset
	ents := make([]raftPB.Entry, 1, 1+uint64(len(ms.ents))-i)
	ents[0].Index = ms.ents[i].Index
	ents[0].Term = ms.ents[i].Term
	ents = append(ents, ms.ents[i+1:]...)
	ms.ents = ents
	return nil
}

func (ms *MemoryStorage) Append(entries []raftPB.Entry) error {
	if len(entries) == 0 {
		return nil
	}
	ms.Lock()
	defer ms.Unlock()

	first := ms.firstIndex()
	last := entries[0].Index + uint64(len(entries)) - 1
	if last < first {
		return nil
	}
	if first > entries[0].Index {
		entries = entries[first-entries[0].Index:]
	}
	offset := entries[0].Index - ms.ents[0].Index
	switch {
	case uint64(len(ms.ents)) > offset:
		// 节点恢复时回出现这种情况，follower节点与leader节点数据不一致
		ms.ents = append([]raftPB.Entry{}, ms.ents[:offset]...)
		ms.ents = append(ms.ents, entries...)
	case uint64(len(ms.ents)) == offset:
		ms.ents = append(ms.ents, entries...)
	default:
		getLogger().Panicf("missing log entry [last: %d, append at :%d]", ms.lastIndex(), entries[0].Index)
	}
	return nil
}
