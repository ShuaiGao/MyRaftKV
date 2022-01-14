package quorum

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
