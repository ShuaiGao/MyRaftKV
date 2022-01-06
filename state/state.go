package state

import "MyRaft/logger"

type StateEnum int

const (
	StateFollower  StateEnum = iota // 正常工作状态
	StateCandidate                  // 选举中
	StateLeader                     // leader状态
)

func (se StateEnum) String() string {
	switch se {
	case StateFollower:
		return "FOLLOWER"
	case StateCandidate:
		return "CANDIDATE"
	case StateLeader:
		return "LEADER"
	default:
		logger.Sugar().Errorf("unknown node state:%d", se)
		return "INVALID_STATE"
	}
}

type StateMachine struct {
	tickC chan struct{}
}
