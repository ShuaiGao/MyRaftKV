package tracker

import (
	"fmt"
	"sort"
	"strings"
)

type Progress struct {
	Match, Next uint64
	State       StateType
	// PendingSnapshot 在StateSnapshot状态时使用
	// 如果当前在发送快照，pendingSnapshot将被设置成快照索引
	// 如果 pendingSnapshot 被设置，该进程的复制操作将暂停
	// raft 将不会再发送快照，直到上一次发送失败
	PendingSnapshot uint64
	// RecentActive 进程最近活跃时会被设置成true
	// 当收到对应follower的任何消息时，表明进程时活跃的
	// 当选举超时时，应该被设置成false
	// leader进程，应该被设置成true
	RecentActive bool
	// ProbeSent 当follower状态为StateProbe时使用此值
	// 当 ProbeSent 为true时，raft应当停止对该进程的日志复制
	// 直到 ProbeSent 被重置
	ProbeSent bool
	// Inflights 是一个发送中消息记录的滑动窗口
	// 每一个消息包含一个或多个log entry
	// 每一个消息包含的entry数量定义在raft配置（MaxSizePerMsg）
	// inflight 有效的限制了发送中的消息数量和每个进程能够使用的带宽
	// 当 inflights 满了的时候，不应当再发送消息
	// 当leader 发送一个消息，发送消息的最后一个entry的索引，应该被添加到inflights
	// index 的添加必须是有序的
	// 当 leader收到一个回复，之前的inflights应当被释放，通过使用最后接收到的entry的index调用inflights.FreeLE
	Inflights *Inflights
	// IsLeader 当前进程为learner时，设置为true
	IsLearner bool
}

func (pr *Progress) ResetState(state StateType) {
	pr.ProbeSent = false
	pr.PendingSnapshot = 0
	pr.State = state
	pr.Inflights.reset()
}
func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b uint64) uint64 {
	if a > b {
		return b
	}
	return a
}
func (pr *Progress) ProbeAcked() {
	pr.ProbeSent = false
}
func (pr *Progress) BecomeProbe() {
	if pr.State == StateSnapshot {
		pendingSnapshot := pr.PendingSnapshot
		pr.ResetState(StateProbe)
		pr.Next = max(pr.Match+1, pendingSnapshot+1)
	} else {
		pr.ResetState(StateProbe)
		pr.Next = pr.Match + 1
	}
}
func (pr *Progress) BecomeReplicate() {
	pr.ResetState(StateReplicate)
	pr.Next = pr.Match + 1
}
func (pr *Progress) BecomeSnapshot(snapshoti uint64) {
	pr.ResetState(StateSnapshot)
	pr.PendingSnapshot = snapshoti
}

// MaybeUpdate 再MsgAppResp 收到follower返回时调用，参数index时该节点接受的index
// 当索引参数过期时，返回false
// 当索引参数未过期时，更新当前进程数据，并返回true
func (pr *Progress) MaybeUpdate(n uint64) bool {
	var updated bool
	if pr.Match < n {
		pr.Match = n
		updated = true
		pr.ProbeAcked()
	}
	pr.Next = max(pr.Next, n+1)
	return updated
}

// OptimisticUpdate 乐观更新
func (pr *Progress) OptimisticUpdate(n uint64) {
	pr.Next = n + 1
}

// MaybeDecrTo 用于再收到MsgApp拒绝时，调整Progress
// 参数为被拒绝的index和想要减到的index
//
// 拒绝发生再消息重复或消息乱序时
// 当拒绝一个已经应用的index时，返回false，并且不做任何修改
//
// 当拒绝确实发生时，需要合理的降低Next，清空正在发送的log entries
func (pr *Progress) MaybeDecrTo(rejected, matchHint uint64) bool {
	if pr.State == StateReplicate {
		if rejected <= pr.Match {
			return false
		}
		pr.Next = pr.Match + 1
		return true
	}
	if pr.Next-1 != rejected {
		return false
	}
	pr.Next = max(min(rejected, matchHint+1), 1)
	pr.ProbeSent = false
	return true
}

// IsPaused 确定log entries 复制，是否应该被限流
func (pr Progress) IsPaused() bool {
	switch pr.State {
	case StateProbe:
		return pr.ProbeSent
	case StateReplicate:
		return pr.Inflights.Full()
	case StateSnapshot:
		return true
	default:
		panic("unexpected state")
	}
}

func (pr *Progress) String() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "%s match=%d next=%d", pr.State, pr.Match, pr.Next)
	if pr.IsLearner {
		fmt.Fprintf(&buf, " learner")
	}
	if pr.IsPaused() {
		fmt.Fprintf(&buf, " paused")
	}
	if pr.PendingSnapshot > 0 {
		fmt.Fprintf(&buf, " pendingSnap=%d", pr.PendingSnapshot)
	}
	if !pr.RecentActive {
		fmt.Fprintf(&buf, " inactive")
	}
	if n := pr.Inflights.Count(); n > 0 {
		fmt.Fprintf(&buf, " inflight=%d", n)
		if pr.Inflights.Full() {
			fmt.Fprintf(&buf, "[full]")
		}
	}
	return buf.String()
}

type ProgressMap map[uint64]*Progress

func (m ProgressMap) String() string {
	ids := make([]uint64, 0, len(m))
	for k := range m {
		ids = append(ids, k)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})
	var buf strings.Builder
	for _, id := range ids {
		fmt.Fprintf(&buf, "%d: %s\n", id, m[id])
	}
	return buf.String()
}
