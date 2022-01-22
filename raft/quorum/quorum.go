package quorum

import (
	"math"
	"strconv"
)

//go:generate stringer -type=VoteResult

type VoteResult uint8

const (
	VotePending VoteResult = 1 + iota
	VoteLost
	VoteWon
)

type Index uint64

func (i Index) String() string {
	if i == math.MaxUint64 {
		return "∞"
	}
	return strconv.FormatUint(uint64(i), 10)
}

type AckedIndexer interface {
	AckedIndex(voterID uint64) (idx Index, found bool)
}
