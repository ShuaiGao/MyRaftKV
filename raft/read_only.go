package raft

type ReadState struct {
	Index      uint64
	RequestCtx []byte
}
type readOnly struct {
}