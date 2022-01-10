package tracker

import raftPB "MyRaft/raft/raftpb"

type Config struct {
}

func (c *Config) Clone() Config {
	return Config{}
}

type ProgressTracker struct {
	Config
	Progress    ProgressMap
	Votes       map[uint64]bool
	MaxInflight int
}

func MakeProgressTracker(maxInflight int) ProgressTracker {
	p := ProgressTracker{
		MaxInflight: maxInflight,
		Config:      Config{},
		Votes:       map[uint64]bool{},
		Progress:    map[uint64]*Progress{},
	}
	return p
}
func (p *ProgressTracker) ConfState() raftPB.ConfState {
	return raftPB.ConfState{}
}

// IsSingleton 当当前配置只有一个投票成员时，返回true
func (p *ProgressTracker) IsSingleton() bool {
	return true
}
func insertionSort(sl []uint64) {
	a, b := 0, len(sl)
	for i := a + 1; i < b; i++ {
		for j := i; j > a && sl[j] < sl[j-1]; j-- {
			sl[j], sl[j-1] = sl[j-1], sl[j]
		}
	}
}

// Visit 有序地progress闭包调用
func (p *ProgressTracker) Visit(f func(id uint64, pr *Progress)) {
	n := len(p.Progress)
	var sl [7]uint64
	var ids []uint64
	if len(sl) >= n {
		ids = sl[:n]
	} else {
		ids = make([]uint64, n)
	}
	for id := range p.Progress {
		n--
		ids[n] = id
	}
	insertionSort(ids)
	for _, id := range ids {
		f(id, p.Progress[id])
	}

}
func (p *ProgressTracker) ResetVotes() {
	p.Votes = map[uint64]bool{}
}

// RecordVote 记录投票
func (p *ProgressTracker) RecordVote(id uint64, v bool) {
	if _, ok := p.Votes[id]; !ok {
		p.Votes[id] = v
	}
}
func (p *ProgressTracker) TallyVotes() (granted int, rejected int) {
	for id, pr := range p.Progress {
		if pr.IsLearner {
			continue
		}
		v, voted := p.Votes[id]
		if !voted {
			continue
		}
		if v {
			granted++
		} else {
			rejected++
		}
	}
	return granted, rejected
}

type matchAckIndexer map[uint64]*Progress

//func (l matchAckIndexer) AckedIndex(id uint64) (quorum.Index, bool) {
//
//}
