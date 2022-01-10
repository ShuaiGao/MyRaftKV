package raft

import (
	raftPB "MyRaft/raft/raftpb"
	"MyRaft/raft/tracker"
)

type BaseStatus struct {
	ID uint64

	raftPB.HardState
	*SoftState

	Applied        uint64
	LeadTransferee uint64
}
type Status struct {
	BaseStatus
	Config   tracker.Config
	Progress map[uint64]tracker.Progress
}

func getProgressCopy(r *raft) map[uint64]tracker.Progress {
	m := make(map[uint64]tracker.Progress)
	r.trk.Visit(func(id uint64, pr *tracker.Progress) {
		p := *pr
		p.Inflights = pr.Inflights.Clone()
		pr = nil
		m[id] = p
	})
	return m
}

func getBasicStatus(r *raft) BaseStatus {
	s := BaseStatus{
		ID:             r.id,
		LeadTransferee: r.leadTransferee,
	}
	s.HardState = r.hardState()
	s.SoftState = r.softState()
	s.Applied = r.raftLog.applied
	return s
}

func getStatus(r *raft) Status {
	var s Status
	s.BaseStatus = getBasicStatus(r)
	if s.RaftState == StateLeader {
		s.Progress = getProgressCopy(r)
	}
	s.Config = r.trk.Config.Clone()
	return s
}

func (s Status) MarshalJSION() ([]byte, error) {
	return []byte(nil), nil
}
func (s Status) String() string {
	b, err := s.MarshalJSION()
	if err != nil {
		getLogger().Panicf("unexpected error: %v", err)
	}
	return string(b)
}
