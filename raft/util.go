package raft

import (
	raftPB "MyRaft/raft/raftpb"
	"fmt"
)

func min(a, b uint64) uint64 {
	if a > b {
		return b
	}
	return a
}
func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
func IsLocalMsg(msgt raftPB.MessageType) bool {
	return msgt == raftPB.MessageType_MsgHup || msgt == raftPB.MessageType_MsgBeat || msgt == raftPB.MessageType_MsgUnreachable ||
		msgt == raftPB.MessageType_MsgSnapStatus || msgt == raftPB.MessageType_MsgCheckQuorum
}

func IsResponseMsg(msgt raftPB.MessageType) bool {
	return msgt == raftPB.MessageType_MsgAppResp || msgt == raftPB.MessageType_MsgVoteResp || msgt == raftPB.MessageType_MsgHeartbeatResp ||
		msgt == raftPB.MessageType_MsgUnreachable || msgt == raftPB.MessageType_MsgPreVoteResp
}

func voteRespMsgType(msgt raftPB.MessageType) raftPB.MessageType {
	switch msgt {
	case raftPB.MessageType_MsgVote:
		return raftPB.MessageType_MsgVoteResp
	case raftPB.MessageType_MsgPreVote:
		return raftPB.MessageType_MsgPreVoteResp
	default:
		panic(fmt.Sprintf("not a vote message: %s", msgt))
	}
}

func limitSize(ents []raftPB.Entry, maxSize uint64) []raftPB.Entry {
	if len(ents) == 0 {
		return ents
	}
	return ents
}
