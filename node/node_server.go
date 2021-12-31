package node

import (
	"MyRaft/logger"
	"MyRaft/node/rpc"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
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
	item := n.GetTailItem()
	reply := &rpc.HeartReply{}
	if in.CommitIndex == uint32(n.CommitIndex+1) && item.Index > n.CommitIndex {
		n.CommitIndex++
		reply.CommitIndex = uint32(n.CommitIndex)
	}
	logger.Logger().Info("rpc Heart",
		zap.String("in_from", in.From),
		zap.Uint32("in_term", in.Term),
		zap.Uint32("in_commit_index", in.CommitIndex),
	)
	return reply, nil
}
func (n *Node) Election(ctx context.Context, in *rpc.ElectionRequest) (*rpc.ElectionReply, error) {
	accept := false
	n.heartBeat = time.Now()
	inTerm := uint(in.Term)
	if n.Term < inTerm {
		preTerm := n.GetTailItem()
		if uint32(preTerm.Index) > in.GetIndex() {
			accept = false
		} else {
			n.Term = inTerm
			n.leaderId = in.NodeId
			accept = true
			n.SetState(StateFollower)
		}
	} else if n.Term == inTerm {
		accept = false
	} else {
		accept = false
	}
	logger.Logger().Info("rpc Election",
		zap.String("in_from", in.NodeId),
		zap.Uint32("in_term", in.Term),
		zap.Bool("accept", accept))
	return &rpc.ElectionReply{Accept: accept, Term: uint32(n.Term), Index: uint32(n.configIndex)}, nil
}
func (n *Node) Append(ctx context.Context, in *rpc.AppendEntryRequest) (reply *rpc.AppendEntryReply, err error) {
	reply = &rpc.AppendEntryReply{Accept: false, AppendIndex: in.GetEntry().GetIndex()}
	item := n.GetTailItem()
	entry := in.GetEntry()
	if item.Term == uint(in.PreTerm) && item.Index == uint(in.PreIndex) && uint32(item.Index+1) == in.GetEntry().GetIndex() {
		newItem := &Item{
			Index: uint(entry.Index),
			Term:  uint(entry.GetTerm()),
			Log:   entry.GetValue(),
		}
		newItemStr, _ := json.Marshal(newItem)
		err = n.rdb.HSet(ctx, n.GetRedisKey(), fmt.Sprintf("%v", entry.Index), newItemStr).Err()
		if err == nil || err == redis.Nil {
			n.ItemList = append(n.ItemList, newItem)
			n.Term = uint(entry.GetTerm())
			reply.Accept = true
			reply.AppendIndex = entry.GetIndex()
			err = nil
		}
	}
	if reply.Accept {
		if n.Id != in.From {
			n.SetState(StateFollower)
		}
		if in.CommitIndex == uint32(n.CommitIndex+1) && item.Index > n.CommitIndex {
			n.CommitIndex++
		}
	} else {
		reply.AppendIndex = uint32(item.Index)
	}
	reply.CommitIndex = uint32(n.CommitIndex)
	entryStr, _ := json.Marshal(in.Entry)
	logger.Logger().Info("rpc Append",
		zap.Uint32("reply_commit_index", reply.CommitIndex),
		zap.Bool("reply_accept", reply.Accept),
		zap.Uint32("in_pre_index", in.PreIndex),
		zap.Uint32("in_pre_term", in.PreTerm),
		zap.Uint32("in_commit_index", in.CommitIndex),
		zap.String("in_from", in.From),
		zap.String("entry", string(entryStr)))
	return reply, err
}
