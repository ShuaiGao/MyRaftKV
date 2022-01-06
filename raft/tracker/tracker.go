package tracker

type ProgressTracker struct {
	Progress    ProgressMap
	Votes       map[uint64]bool
	MaxInflight int
}
