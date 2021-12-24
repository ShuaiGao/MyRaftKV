package node

import (
	"MyRaft/logger"
	"MyRaft/node/rpc"
	"context"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"time"
)

func (n *Node) SendOathRequest(cfg NodeConfig) {
	conn, err := grpc.Dial(cfg.Address(), grpc.WithInsecure())
	if err != nil {
		logger.Logger().Sugar().Fatal("did not connect:%s", cfg.Address())
	}
	defer conn.Close()
	c := rpc.NewElectionServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*300)
	defer cancel()
	r, err := c.Oath(ctx, &rpc.OathRequest{Term: uint32(n.Term), NodeId: n.id})
	if err != nil {
		logger.Logger().Warn("SendOathRequest_oath_error", zap.String("address", cfg.Address()), zap.Error(err))
		return
	}
	if r.Accept {
		n.OathAcceptNum++
		n.otherNodeList = append(n.otherNodeList, WorkNode{
			id:  r.NodeId,
			cfg: cfg,
		})
	}
	logger.Logger().Info("Send OathRequest",
		zap.String("node_id", n.id),
		zap.String("vote_id", n.id),
		zap.String("ticket_from", r.GetNodeId()),
		zap.Bool("accept", r.Accept),
		zap.Int("oath_accept_num", n.OathAcceptNum))
}

var kacp = keepalive.ClientParameters{
	Time:                5 * time.Second,
	Timeout:             time.Second,
	PermitWithoutStream: true,
}

func (n *Node) SendHeart(node WorkNode) {
	// 心跳拨号需要keepalive，因为leader 对 flower 的通讯是频繁的
	conn, err := grpc.Dial(node.cfg.Address(), grpc.WithKeepaliveParams(kacp))
	if err != nil {
		logger.Sugar().Warn("did not connect:%s", zap.String("address", node.cfg.Address()), zap.Error(err))
		return
	}
	defer conn.Close()
	c := rpc.NewElectionServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	heart := &rpc.HeartRequest{From: n.id, Term: uint32(n.Term)}
	if node.commitIndex < node.acceptIndex {
		heart.CommitIndex = uint32(node.commitIndex + 1)
	}
	_, err = c.Heart(ctx, heart)
	if err != nil {
		logger.Sugar().Warn("heart rpc error", zap.String("address", node.cfg.Address()), zap.Error(err))
	}
	logger.Logger().Info("Send Heart",
		zap.String("node_id", n.id),
		zap.Uint("term", n.Term))
}

func (n *Node) AppendEntry(node WorkNode, item *Item) {
	conn, err := grpc.Dial(node.cfg.Address(), grpc.WithInsecure())
	if err != nil {
		logger.Sugar().Warn("AppendEntry did not connect:%s", zap.String("address", node.cfg.Address()), zap.Error(err))
		return
	}
	defer conn.Close()
	c := rpc.NewElectionServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	appendEntry := &rpc.AppendEntryRequest{
		Entry:    &rpc.Entry{},
		PreIndex: uint32(node.acceptIndex),
		PreTerm:  uint32(node.term),
	}
	if node.commitIndex < node.acceptIndex {
		appendEntry.CommitIndex = uint32(node.commitIndex + 1)
	}
	_, err = c.Append(ctx, appendEntry)
	if err != nil {
		logger.Sugar().Warn("AppendEntry rpc error", zap.String("address", node.cfg.Address()), zap.Error(err))
	}
	logger.Logger().Info("append_entry",
		zap.String("node_id", n.id),
		zap.Uint("term", item.Term),
		zap.Uint("index", item.Index),
		zap.String("key", item.Key),
	)
}
