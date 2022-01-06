package raft

import "MyRaft/logger"

type node struct {
	tickc chan struct{}
	done  chan struct{}
	stop  chan struct{}
	rn    *RawNode
}

func newNode() *node {
	return &node{}
}
func (n *node) Tick() {
	select {
	case n.tickc <- struct{}{}:
	case <-n.done:
	default:
		logger.Sugar().Warnf("%v tick error", n.rn.raft.id)
	}
}
func (n *node) Stop() {

}
func (n *node) run() {
	for {
		select {
		case <-n.tickc:
			n.rn.raft.tick()
		case <-n.stop:
			close(n.done)
			return
		}
	}
}
