package raft

type Storage interface {
	LastIndex() (uint64, error)
	FirstIndex() (uint64, error)
}
