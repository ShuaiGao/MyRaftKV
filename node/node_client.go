package node

import (
	"MyRaft/logger"
	"MyRaft/node/rpc"
	"context"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"time"
)

type Client struct {
	conn    *grpc.ClientConn
	timeout time.Duration
}

func newClient(conn *grpc.ClientConn, timeout time.Duration) Client {
	return Client{
		conn:    conn,
		timeout: timeout,
	}
}
func (cli Client) SendElectionRequest(n *Node) error {
	c := rpc.NewElectionServiceClient(cli.conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*300)
	defer cancel()
	r, err := c.Election(ctx, &rpc.ElectionRequest{Term: uint32(n.Term), NodeId: n.Id, Index: uint32(n.AcceptIndex)})
	if err != nil {
		logger.Logger().Warn("SendElectionRequest_oath_error", zap.Error(err))
		return err
	}
	if r.Accept {
		n.AddOathAcceptNum(1)
		//n.OtherNodeList = append(n.OtherNodeList, WorkNode{
		//	Id:  r.NodeId,
		//	cfg: cfg,
		//})
	} else {
		if uint(r.Term) > n.Term {
			n.SetState(StateFollower)
		} else if uint(r.Term) == n.Term && uint(r.Index) > n.configIndex {
			n.SetState(StateFollower)
		}
	}
	logger.Logger().Info("Send ElectionRequest",
		zap.String("node_id", n.Id),
		zap.String("vote_id", n.Id),
		zap.String("ticket_from", r.GetNodeId()),
		zap.Bool("accept", r.Accept),
		zap.Int("oath_accept_num", n.GetOathAcceptNum()))
	return nil
}

var kacp = keepalive.ClientParameters{
	Time:                5 * time.Second,
	Timeout:             time.Second,
	PermitWithoutStream: true,
}

func (cli Client) SendHeart(n *Node, node *WorkNode) error {
	// 心跳拨号需要keepalive，因为leader 对 flower 的通讯是频繁的
	//conn, err := grpc.Dial(node.cfg.Address(), grpc.WithKeepaliveParams(kacp))
	c := rpc.NewElectionServiceClient(cli.conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	heart := &rpc.HeartRequest{From: n.Id, Term: uint32(n.Term)}
	if node.commitIndex < node.acceptIndex {
		heart.CommitIndex = uint32(node.commitIndex + 1)
	}
	_, err := c.Heart(ctx, heart)
	if err != nil {
		logger.Sugar().Warn("heart rpc error", zap.String("address", node.cfg.Address()), zap.Error(err))
		return errors.Wrap(err, "rpc_error")
	}
	logger.Logger().Info("Send Heart",
		zap.String("from", n.Id),
		zap.Int("to", node.id),
		zap.Uint("from_term", n.Term),
		zap.Uint("node_commit_index", node.commitIndex),
		zap.Uint("node_accept_index", node.acceptIndex),
	)
	return nil
}

func (cli Client) SendAppendEntry(n *Node, node *WorkNode) error {
	c := rpc.NewElectionServiceClient(cli.conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	item := n.GetTailItem()
	preItem := n.GetPreItem(item)
	appendEntry := &rpc.AppendEntryRequest{
		Entry:    &rpc.Entry{Term: uint32(item.Term), Index: uint32(item.Index), Value: item.Log},
		PreIndex: uint32(preItem.Index),
		PreTerm:  uint32(preItem.Term),
		From:     n.Id,
	}
	if node.commitIndex < node.acceptIndex {
		appendEntry.CommitIndex = uint32(node.commitIndex + 1)
	}
	reply, err := c.Append(ctx, appendEntry)
	if err != nil {
		logger.Sugar().Warn("SendAppendEntry rpc error", zap.String("address", node.cfg.Address()), zap.Error(err))
	} else if reply.GetAccept() {
		node.acceptIndex = uint(reply.AppendIndex)
		node.commitIndex = uint(reply.CommitIndex)
		n.CheckAcceptId()
		n.CheckCommitId()
	} else {
		go func() {
			for preItem != nil {
				item := preItem
				preItem := n.GetPreItem(item)
				appendEntry := &rpc.AppendEntryRequest{
					Entry:    &rpc.Entry{Term: uint32(item.Term), Index: uint32(item.Index), Value: item.Log},
					PreIndex: uint32(preItem.Index),
					PreTerm:  uint32(preItem.Term),
					From:     n.Id,
				}
				if node.commitIndex < node.acceptIndex {
					appendEntry.CommitIndex = uint32(node.commitIndex + 1)
				}
				reply, err := c.Append(ctx, appendEntry)
				if err != nil {
					return
				}
				if reply.GetAccept() {
					preItem = n.GetNextItem(item)
				}
			}
		}()
	}
	logger.Logger().Info("append_entry",
		zap.Int("send_to", node.id),
		zap.String("node_id", n.Id),
		zap.Uint("item_term", item.Term),
		zap.Uint("item_index", item.Index),
		zap.Bool("accept", reply.GetAccept()),
		zap.Uint32("reply_accept_index", reply.GetAppendIndex()),
		zap.Uint32("reply_commit_index", reply.GetCommitIndex()),
		zap.Uint("node_index", node.acceptIndex),
		zap.Uint("node_commit", node.commitIndex),
	)
	return err
}
