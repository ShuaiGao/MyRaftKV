package node

import (
	"MyRaft/node/rpc"
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"log"
	"net"
	"reflect"
	"testing"
	"time"
)

type mockNodeServer struct {
	rpc.UnimplementedElectionServiceServer
}

func (*mockNodeServer) Election(ctx context.Context, in *rpc.ElectionRequest) (*rpc.ElectionReply, error) {
	if in.GetNodeId() == "0" {
		return &rpc.ElectionReply{Accept: true, NodeId: "1", Term: in.Term, Index: in.Index}, nil
	}
	return &rpc.ElectionReply{Accept: false, NodeId: "1", Term: in.Term + 1, Index: in.Index + 1}, nil
}

func (*mockNodeServer) Heart(context.Context, *rpc.HeartRequest) (*rpc.HeartReply, error) {
	return &rpc.HeartReply{}, nil
}
func (*mockNodeServer) Append(ctx context.Context, in *rpc.AppendEntryRequest) (*rpc.AppendEntryReply, error) {
	if in.GetPreIndex() < 0 {
		return nil, fmt.Errorf("PreIndex(%d) less than 0", in.GetPreIndex())
	}
	if in.GetPreTerm() < 0 {
		return nil, fmt.Errorf("PreTerm(%d) less than 0", in.GetPreTerm())
	}
	if in.GetEntry().GetIndex() != in.GetPreIndex()+1 {
		return nil, fmt.Errorf("Entry.Index(%d) != PreIndex(%d)+1", in.GetEntry().GetIndex(), in.GetPreIndex())
	}
	if in.GetEntry().GetTerm() < in.GetPreTerm() {
		return nil, fmt.Errorf("Entry.Term(%d) is greater than PreTerm(%d)", in.GetEntry().GetTerm(), in.GetPreTerm())
	}
	if in.GetPreIndex() == 5 {
		return &rpc.AppendEntryReply{Accept: false, AppendIndex: 2, CommitIndex: 3}, nil
	}
	return &rpc.AppendEntryReply{Accept: true, AppendIndex: 2, CommitIndex: 3}, nil
}

func dialer() func(context.Context, string) (net.Conn, error) {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	rpc.RegisterElectionServiceServer(server, &mockNodeServer{})
	go func() {
		if err := server.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()
	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}
func TestClient_SendHeart(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "", grpc.WithInsecure(), grpc.WithContextDialer(dialer()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	tests := []struct {
		name     string
		node     *Node
		workNode *WorkNode
		err      error
	}{
		{
			name:     "send_heart_ok",
			node:     &Node{Id: "0", Term: 3},
			workNode: &WorkNode{id: 0, acceptIndex: 10, commitIndex: 8, cfg: NodeConfig{Port: 1990, Host: "localhost"}},
			err:      nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newClient(conn, time.Second).SendHeart(tt.node, tt.workNode)
			if err != nil && errors.Is(err, tt.err) {
				t.Error("error: expected", tt.err, "received", err)
			}
		})
	}
}

func TestClient_SendElectionRequest(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "", grpc.WithInsecure(), grpc.WithContextDialer(dialer()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	tests := []struct {
		name     string
		node     *Node
		wantNode *Node
		wantErr  bool
	}{
		{
			name:     "send_election_accept",
			node:     &Node{Id: "0", Term: 3, AcceptIndex: 2, OathAcceptNum: 0, state: StateCandidate},
			wantNode: &Node{Id: "0", Term: 3, AcceptIndex: 2, OathAcceptNum: 1, state: StateCandidate},
			wantErr:  false,
		},
		{
			name:     "send_election_accept_2",
			node:     &Node{Id: "0", Term: 3, AcceptIndex: 2, OathAcceptNum: 0, state: StateCandidate},
			wantNode: &Node{Id: "0", Term: 3, AcceptIndex: 2, OathAcceptNum: 1, state: StateCandidate},
			wantErr:  false,
		},
		{
			name:     "send_election_not_accept",
			node:     &Node{Id: "1", Term: 3, AcceptIndex: 2, OathAcceptNum: 0, state: StateCandidate},
			wantNode: &Node{Id: "1", Term: 3, AcceptIndex: 2, OathAcceptNum: 0, state: StateFollower},
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newClient(conn, time.Second).SendElectionRequest(tt.node)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendAppendEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(tt.node, tt.wantNode) {
				t.Errorf("Append() got = %v, want %v", tt.node, tt.wantNode)
			}
		})
	}
}

//func TestClient_SendAppendEntry(t *testing.T) {
//	ctx := context.Background()
//	conn, err := grpc.DialContext(ctx, "", grpc.WithInsecure(), grpc.WithContextDialer(dialer()))
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer conn.Close()
//	tests := []struct {
//		name     string
//		node     *Node
//		workNode *WorkNode
//		want     *rpc.AppendEntryReply
//		wantErr  bool
//	}{
//		{
//			name:     "send_append_entry_accept",
//			node:     &Node{Id: "0", Term: 3, ItemList: []*Item{{Index: 1, Term: 3, Log: ""}}},
//			workNode: &WorkNode{id: 0, acceptIndex: 10, commitIndex: 8},
//			want: &rpc.AppendEntryReply{id: 0, acceptIndex: 2, commitIndex: 3},
//			wantErr:  false,
//		},
//		{
//			name:     "send_append_entry_accept_2",
//			node:     &Node{Id: "0", Term: 3, ItemList: []*Item{{Index: 1, Term: 4, Log: ""}}},
//			workNode: &WorkNode{id: 0, acceptIndex: 0, commitIndex: 0},
//			want: &WorkNode{id: 0, acceptIndex: 2, commitIndex: 3},
//			wantErr:  false,
//		},
//		{
//			name:     "send_append_entry_not_accept",
//			node:     &Node{Id: "0", Term: 3, ItemList: []*Item{{Index: 5, Term: 2, Log: ""}, {Index: 6, Term: 2, Log: ""}}},
//			workNode: &WorkNode{id: 0, acceptIndex: 4, commitIndex: 8},
//			want: &WorkNode{id: 0, acceptIndex: 4, commitIndex: 8},
//			wantErr:  false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			reply, err := newClient(conn, time.Second).SendAppendEntry(tt.node, tt.workNode)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("SendAppendEntry() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(tt.workNode, tt.wantNode) {
//				t.Errorf("Append() got = %v, want %v", tt.workNode, tt.wantNode)
//			}
//		})
//	}
//}

//func TestClient_SendAppendEntry(t *testing.T) {
