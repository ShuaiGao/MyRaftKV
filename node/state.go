package node

import "MyRaft/logger"

type NodeState int32
type NodeRole int32

func (ns NodeState) String() string {
	switch ns {
	case StateFollower:
		return "FOLLOWER"
	case StateCandidate:
		return "CANDIDATE"
	case StateLeader:
		return "LEADER"
	default:
		logger.Sugar().Errorf("unknown node state:%d", ns)
		return "INVALID_STATE"
	}
}

const (
	StateFollower  NodeState = iota // 正常工作状态
	StateCandidate                  // 选举中
	StateLeader                     // leader状态
)

var Tern = 0
