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

func TestNode_GetPreItem(t *testing.T) {
	n := Node{AcceptIndex: 1, ItemList: []*Item{
		{Index: 0, Term: 1}, {Index: 1, Term: 1}, {Index: 2, Term: 2},
	}}
	tail := n.GetPreItem(nil)
	assert.Equal(t, n.ItemList[len(n.ItemList)-1], tail)
	pre := n.GetPreItem(tail)
	assert.Equal(t, n.ItemList[len(n.ItemList)-2], pre)
	pre = n.GetPreItem(pre)
	assert.Equal(t, n.ItemList[len(n.ItemList)-3], pre)
	pre = n.GetPreItem(pre)
	assert.Equal(t, uint(0), pre.Index)
	assert.Equal(t, uint(0), pre.Term)
	assert.Equal(t, "", pre.Log)
}

func TestNode_GetTailItem(t *testing.T) {
	n := Node{AcceptIndex: 1, ItemList: []*Item{{Index: 0, Term: 1}, {Index: 1, Term: 1}, {Index: 2, Term: 2}}}
	tail := n.GetTailItem()
	assert.Equal(t, n.ItemList[len(n.ItemList)-1], tail)
}
func TestNode_GetTailItem2(t *testing.T) {
	n := Node{AcceptIndex: 1, ItemList: []*Item{}}
	tail := n.GetTailItem()
	assert.NotNil(t, tail)
	assert.Equal(t, uint(0), tail.Index)
	assert.Equal(t, uint(0), tail.Term)
	assert.Equal(t, "", tail.Log)
}
