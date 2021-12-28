package node

type NodeState int32
type NodeRole int32

const (
	StateFollower    NodeState = 0 // 正常工作状态
	StateNoAvailable NodeState = 1 // 不可用状态
	StateCandidate   NodeState = 2 // 选举中
	StateLeader      NodeState = 3 // leader状态

	NodeRoleCandidate NodeRole = 0
	NodeRoleLeader    NodeRole = 1
	NodeRoleFollower  NodeRole = 2
)

var Tern = 0
