package node

import (
	"MyRaft/logger"
	"MyRaft/node/rpc"
	"context"
	"go.uber.org/zap"
	"time"
)

func (n *Node) Heart(ctx context.Context, in *rpc.HeartRequest) (*rpc.HeartReply, error) {
	n.heartBeat = time.Now()
	if uint(in.Term) >= n.Term {
		n.Term = uint(in.Term)
		n.leaderId = in.From
		n.SetState(StateFollower)
	}
	item := n.GetPreItem()
	reply := &rpc.HeartReply{}
	if in.CommitIndex == uint32(n.CommitIndex+1) && item.Index > n.CommitIndex {
		n.CommitIndex++
		reply.CommitIndex = uint32(n.CommitIndex)
	}
	logger.Logger().Info("rpc Heart", zap.String("from", in.From), zap.Uint32("term", in.Term))
	return reply, nil
}
func (n *Node) Election(ctx context.Context, in *rpc.ElectionRequest) (*rpc.ElectionReply, error) {
	accept := false
	n.heartBeat = time.Now()
	inTerm := uint(in.Term)
	if n.Term < inTerm {
		preTerm := n.GetPreItem()
		if uint32(preTerm.Index) > in.GetIndex() {
			accept = false
		} else {
			n.Term = inTerm
			n.leaderId = in.NodeId
			accept = true
		}
	} else if n.Term == inTerm {
		accept = false
	} else {
		accept = false
	}
	logger.Logger().Info("rpc Election",
		zap.String("from", in.NodeId),
		zap.Uint32("term", in.Term),
		zap.Bool("accept", accept))
	return &rpc.ElectionReply{Accept: accept}, nil
}
func (n *Node) Append(ctx context.Context, in *rpc.AppendEntryRequest) (*rpc.AppendEntryReply, error) {
	reply := &rpc.AppendEntryReply{AppendIndex: in.GetEntry().GetIndex()}
	item := n.GetPreItem()
	if item.Term == uint(in.PreTerm) && item.Index == uint(in.PreIndex) && uint32(item.Index+1) == in.GetEntry().GetIndex() {
		entry := in.GetEntry()
		n.ItemList = append(n.ItemList, &Item{
			Index: uint(entry.Index),
			Term:  uint(entry.GetTerm()),
			Log:   entry.GetValue()},
		)
		reply.Accept = true
		reply.AppendIndex = in.GetEntry().GetIndex()
	} else {
		reply.Accept = false
	}
	if reply.Accept {
		if n.GetState() == StateLeader {
			n.CommitIndex++
			reply.CommitIndex = uint32(n.CommitIndex)
		} else if in.CommitIndex == uint32(n.CommitIndex+1) && item.Index > n.CommitIndex {
			n.CommitIndex++
			reply.CommitIndex = uint32(n.CommitIndex)
		}
	}
	return reply, nil
}
