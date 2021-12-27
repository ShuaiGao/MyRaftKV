package node

import (
	"MyRaft/node/rpc"
	"context"
	"github.com/stretchr/testify/assert"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestNode_Heart(t *testing.T) {
	n := Node{Term: 1}
	in := &rpc.HeartRequest{Term: 100, From: "888"}
	reply, err := n.Heart(context.Background(), in)
	assert.Nil(t, err)
	assert.NotNil(t, reply)
	assert.Equal(t, uint(100), n.Term)
	assert.Equal(t, StateFollower, n.GetState())
	assert.Equal(t, "888", n.leaderId)
}

func TestNode_Oath(t *testing.T) {
	type fields struct {
		id            string
		electing      bool
		lock          sync.RWMutex
		Term          uint
		leaderId      string
		heartBeat     time.Time
		state         NodeState
		role          NodeRole
		OathAcceptNum int
		config        []NodeConfig
		configIndex   int
		otherNodeList []Node
		ItemList      []*Item
	}
	type args struct {
		ctx context.Context
		in  *rpc.ElectionRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *rpc.ElectionReply
		wantErr bool
	}{
		{
			name:    "election_success_the_first_time",
			args:    args{ctx: context.Background(), in: &rpc.ElectionRequest{Term: 1, Index: 0, NodeId: "ddd"}},
			fields:  fields{id: "candidate", Term: 0, state: StateFollower, ItemList: []*Item{}},
			want:    &rpc.ElectionReply{Accept: true},
			wantErr: false,
		},
		{
			name:    "election_fail_by_less_term",
			args:    args{ctx: context.Background(), in: &rpc.ElectionRequest{Term: 1, Index: 100, NodeId: "ddd"}},
			fields:  fields{id: "candidate", Term: 2, state: StateFollower, ItemList: []*Item{{Index: 3, Term: 7, Log: ""}}},
			want:    &rpc.ElectionReply{Accept: false},
			wantErr: false,
		},
		{
			name:    "election_fail_by_less_index",
			args:    args{ctx: context.Background(), in: &rpc.ElectionRequest{Term: 3, Index: 1, NodeId: "ddd"}},
			fields:  fields{id: "candidate", Term: 2, state: StateFollower, ItemList: []*Item{{Index: 3, Term: 7, Log: ""}}},
			want:    &rpc.ElectionReply{Accept: false},
			wantErr: false,
		},
		{
			name:    "election_fail_by_same_term",
			args:    args{ctx: context.Background(), in: &rpc.ElectionRequest{Term: 3, Index: 10, NodeId: "ddd"}},
			fields:  fields{id: "candidate", Term: 3, state: StateFollower, ItemList: []*Item{{Index: 3, Term: 7, Log: ""}}},
			want:    &rpc.ElectionReply{Accept: false},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{
				id:            tt.fields.id,
				electing:      tt.fields.electing,
				Term:          tt.fields.Term,
				leaderId:      tt.fields.leaderId,
				heartBeat:     tt.fields.heartBeat,
				state:         tt.fields.state,
				OathAcceptNum: tt.fields.OathAcceptNum,
				config:        tt.fields.config,
				configIndex:   tt.fields.configIndex,
				ItemList:      tt.fields.ItemList,
			}
			got, err := n.Election(tt.args.ctx, tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("Election() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Election() got = %v, want %v", got, tt.want)
			}
		})
	}
}

//func TestNode_Append(t *testing.T) {
//	n := Node{Term: 8, state: StateLeader, CommitIndex: 0, ItemList: []*Item{}}
//	in := &rpc.AppendEntryRequest{Entry: &rpc.Entry{Term: 8, Index: 1, Value: "hello"}, PreTerm: 0, PreIndex: 0}
//	reply, err := n.Append(context.Background(), in)
//	assert.Nil(t, err)
//	assert.Equal(t, true, reply.GetAccept())
//	assert.Equal(t, 1, len(n.ItemList))
//	assert.Equal(t, uint(1), n.GetPreItem().Index)
//	assert.Equal(t, uint(8), n.GetPreItem().Term)
//}

func TestNode_Append(t *testing.T) {
	type fields struct {
		UnimplementedElectionServiceServer *rpc.UnimplementedElectionServiceServer
		id                                 string
		electing                           bool
		lock                               sync.RWMutex
		Term                               uint
		leaderId                           string
		heartBeat                          time.Time
		state                              NodeState
		OathAcceptNum                      int
		config                             []NodeConfig
		configIndex                        int
		otherNodeList                      []WorkNode
		ItemList                           []*Item
		CommitIndex                        uint
	}
	type args struct {
		ctx context.Context
		in  *rpc.AppendEntryRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *rpc.AppendEntryReply
		wantErr bool
	}{
		{
			name:   "leader_append_ok_1",
			fields: fields{Term: 8, state: StateLeader, CommitIndex: 0, ItemList: []*Item{}},
			args: args{ctx: context.Background(), in: &rpc.AppendEntryRequest{
				Entry: &rpc.Entry{Term: 8, Index: 1, Value: "hello"}, PreTerm: 0, PreIndex: 0,
			}},
			want:    &rpc.AppendEntryReply{Accept: true, AppendIndex: 1, CommitIndex: 1},
			wantErr: false,
		},
		{
			name:   "leader_append_ok_2",
			fields: fields{Term: 8, state: StateLeader, CommitIndex: 3, ItemList: []*Item{{Index: 3, Term: 7, Log: ""}}},
			args: args{ctx: context.Background(), in: &rpc.AppendEntryRequest{
				Entry: &rpc.Entry{Term: 8, Index: 4, Value: "hello"}, PreTerm: 7, PreIndex: 3,
			}},
			want:    &rpc.AppendEntryReply{Accept: true, AppendIndex: 4, CommitIndex: 4},
			wantErr: false,
		},
		{
			name:   "follower_append_ok_1",
			fields: fields{Term: 8, state: StateFollower, CommitIndex: 0, ItemList: []*Item{}},
			args: args{ctx: context.Background(), in: &rpc.AppendEntryRequest{
				Entry: &rpc.Entry{Term: 9, Index: 1, Value: "hello"}, PreTerm: 0, PreIndex: 0,
			}},
			want:    &rpc.AppendEntryReply{Accept: true, AppendIndex: 1, CommitIndex: 0},
			wantErr: false,
		},
		{
			name:   "follower_append_ok_2",
			fields: fields{Term: 8, state: StateFollower, CommitIndex: 3, ItemList: []*Item{{Index: 3, Term: 7, Log: ""}}},
			args: args{ctx: context.Background(), in: &rpc.AppendEntryRequest{
				Entry: &rpc.Entry{Term: 8, Index: 4, Value: "hello"}, PreTerm: 7, PreIndex: 3,
			}},
			want:    &rpc.AppendEntryReply{Accept: true, AppendIndex: 4, CommitIndex: 0},
			wantErr: false,
		},
		{
			name:   "follower_append_ok_3_with_commit",
			fields: fields{Term: 8, state: StateFollower, CommitIndex: 2, ItemList: []*Item{{Index: 3, Term: 7, Log: ""}}},
			args: args{ctx: context.Background(), in: &rpc.AppendEntryRequest{
				Entry: &rpc.Entry{Term: 8, Index: 4, Value: "hello"}, PreTerm: 7, PreIndex: 3, CommitIndex: 3,
			}},
			want:    &rpc.AppendEntryReply{Accept: true, AppendIndex: 4, CommitIndex: 3},
			wantErr: false,
		},
		{
			name:   "follower_append_fail_1_PreTerm_error",
			fields: fields{Term: 8, state: StateFollower, CommitIndex: 3, ItemList: []*Item{{Index: 3, Term: 7, Log: ""}}},
			args: args{ctx: context.Background(), in: &rpc.AppendEntryRequest{
				Entry: &rpc.Entry{Term: 8, Index: 4, Value: "hello"}, PreTerm: 8, PreIndex: 3,
			}},
			want:    &rpc.AppendEntryReply{Accept: false, AppendIndex: 4, CommitIndex: 0},
			wantErr: false,
		},
		{
			name:   "follower_append_fail_2_PreIndex_error",
			fields: fields{Term: 8, state: StateFollower, CommitIndex: 3, ItemList: []*Item{{Index: 3, Term: 7, Log: ""}}},
			args: args{ctx: context.Background(), in: &rpc.AppendEntryRequest{
				Entry: &rpc.Entry{Term: 8, Index: 5, Value: "hello"}, PreTerm: 7, PreIndex: 4,
			}},
			want:    &rpc.AppendEntryReply{Accept: false, AppendIndex: 5, CommitIndex: 0},
			wantErr: false,
		},
		{
			name:   "follower_append_fail_3_Entry_Index_error",
			fields: fields{Term: 8, state: StateFollower, CommitIndex: 3, ItemList: []*Item{{Index: 3, Term: 7, Log: ""}}},
			args: args{ctx: context.Background(), in: &rpc.AppendEntryRequest{
				Entry: &rpc.Entry{Term: 8, Index: 6, Value: "hello"}, PreTerm: 7, PreIndex: 3,
			}},
			want:    &rpc.AppendEntryReply{Accept: false, AppendIndex: 6, CommitIndex: 0},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{
				UnimplementedElectionServiceServer: tt.fields.UnimplementedElectionServiceServer,
				id:                                 tt.fields.id,
				electing:                           tt.fields.electing,
				Term:                               tt.fields.Term,
				leaderId:                           tt.fields.leaderId,
				heartBeat:                          tt.fields.heartBeat,
				state:                              tt.fields.state,
				OathAcceptNum:                      tt.fields.OathAcceptNum,
				config:                             tt.fields.config,
				configIndex:                        tt.fields.configIndex,
				otherNodeList:                      tt.fields.otherNodeList,
				ItemList:                           tt.fields.ItemList,
				CommitIndex:                        tt.fields.CommitIndex,
			}
			got, err := n.Append(tt.args.ctx, tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("Append() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Append() got = %v, want %v", got, tt.want)
			}
		})
	}
}
