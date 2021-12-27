package node

import (
	"MyRaft/node/rpc"
	"context"
)

func (n *Node) DBAppend(ctx context.Context, in *rpc.DBAppendLogRequest) (*rpc.DBAppendLogReply, error) {
	return &rpc.DBAppendLogReply{}, nil
}
