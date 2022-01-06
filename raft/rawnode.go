package raft

type RawNode struct {
	raft *raft
}

func NewRawNode(config *Config) (*RawNode, error) {
	r := newRaft(config)
	rn := &RawNode{
		raft: r,
	}
	return rn, nil
}
