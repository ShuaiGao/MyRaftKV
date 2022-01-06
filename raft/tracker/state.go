package tracker

type StateType uint64

const (
	// StateProbe 表明不知道follower的last index
	StateProbe StateType = iota
	StateReplicate
	// StateSnapshot 表明follower当前log entries不可用，需要一个全量快照
	StateSnapshot
)

var StringMap = [...]string{
	"StateProbe",
	"StateReplicate",
	"StateSnapshot",
}

func (st StateType) String() string {
	return StringMap[st]
}
