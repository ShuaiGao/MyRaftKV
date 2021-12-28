package node

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNode_CheckCommitId(t *testing.T) {
	n := Node{CommitIndex: 1, OtherNodeList: []*WorkNode{
		{commitIndex: 2}, {commitIndex: 3},
	}}

	n.CheckCommitId()
	assert.Equal(t, uint(2), n.CommitIndex)
	n.CheckCommitId()
	assert.Equal(t, uint(2), n.CommitIndex)
}

func TestNode_CheckAcceptId(t *testing.T) {
	n := Node{AcceptIndex: 1, OtherNodeList: []*WorkNode{
		{acceptIndex: 2}, {acceptIndex: 3},
	}}

	n.CheckAcceptId()
	assert.Equal(t, uint(2), n.AcceptIndex)
	n.CheckAcceptId()
	assert.Equal(t, uint(2), n.AcceptIndex)
}
