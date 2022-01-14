package quorum

type JointConfig [2]MajorityConfig

func (c JointConfig) IDs() map[uint64]struct{} {
	m:=

}
func (c JointConfig) VoteResult(votes map[uint64]bool) VoteResult {
	r1 := c[0].VoteResult(votes)
	r2 := c[1].VoteResult(votes)
	if r1 == r2{
		return r1
	}
	if r1 == VoteLost || r2 == VoteLost{
		return VoteLost
	}
	return VotePending
}