package quorum

//go:generate stringer -type=VoteResult

type VoteResult uint8

const (
	VotePending VoteResult = 1 + iota
	VoteLost
	VoteWon
)
