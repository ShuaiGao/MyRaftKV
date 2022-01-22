package quorum

import "math"

type MajorityConfig map[uint64]struct{}

func (m MajorityConfig) VoteResult(votes map[uint64]bool) VoteResult {
	if len(m) == 0 {
		return VoteWon
	}
	ny := [2]int{}

	var missing int
	for id := range m {
		v, ok := votes[id]
		if !ok {
			missing++
			continue
		}
		if v {
			ny[1]++
		} else {
			ny[0]++
		}
	}
	q := len(m)/2 + 1
	if ny[1] >= q {
		return VoteWon
	}
	if ny[1]+missing >= q {
		return VotePending
	}
	return VoteLost
}

func (m MajorityConfig) CommittedIndex(l AckedIndexer) Index {
	n := len(m)
	if n == 0 {
		return math.MaxUint64
	}
	var stk [7]uint64
	var srt []uint64
	if len(stk) >= n {
		srt = stk[:n]
	} else {
		srt = make([]uint64, n)
	}
	{
		i := n - 1
		for id := range m {
			if idx, ok := l.AckedIndex(id); ok {
				srt[i] = uint64(idx)
				i--
			}
		}
	}
	pos := n - (n/2 + 1)
	return Index(srt[pos])
}
