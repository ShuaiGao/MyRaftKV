package node

import (
	"MyRaft/node/rpc"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestNode_Heart2(t *testing.T) {
	preTime := time.Now()
	n := Node{Term: 1, heartBeat: preTime}
	in := &rpc.HeartRequest{Term: 100, From: "888"}
	time.Sleep(2)
	reply, err := n.Heart(context.Background(), in)
	assert.Nil(t, err)
	assert.NotNil(t, reply)
	assert.Equal(t, uint(100), n.Term)
	assert.Equal(t, StateFollower, n.GetState())
	assert.Equal(t, "888", n.leaderId)
	assert.Greater(t, n.heartBeat.Sub(preTime), time.Duration(1))
}

func TestNode_Election(t *testing.T) {
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
		configIndex   uint
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
			want:    &rpc.ElectionReply{Accept: true, Term: 1},
			wantErr: false,
		},
		{
			name:    "election_fail_by_less_term",
			args:    args{ctx: context.Background(), in: &rpc.ElectionRequest{Term: 1, Index: 100, NodeId: "ddd"}},
			fields:  fields{id: "candidate", Term: 2, state: StateFollower, ItemList: []*Item{{Index: 3, Term: 7, Log: ""}}},
			want:    &rpc.ElectionReply{Accept: false, Term: 2},
			wantErr: false,
		},
		{
			name:    "election_fail_by_less_index",
			args:    args{ctx: context.Background(), in: &rpc.ElectionRequest{Term: 3, Index: 1, NodeId: "ddd"}},
			fields:  fields{id: "candidate", Term: 2, state: StateFollower, ItemList: []*Item{{Index: 3, Term: 7, Log: ""}}},
			want:    &rpc.ElectionReply{Accept: false, Term: 2},
			wantErr: false,
		},
		{
			name:    "election_fail_by_same_term",
			args:    args{ctx: context.Background(), in: &rpc.ElectionRequest{Term: 3, Index: 10, NodeId: "ddd"}},
			fields:  fields{id: "candidate", Term: 3, state: StateFollower, ItemList: []*Item{{Index: 3, Term: 7, Log: ""}}},
			want:    &rpc.ElectionReply{Accept: false, Term: 3},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{
				Id:            tt.fields.id,
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

func TestNode_Gao(t *testing.T) {
	var ctx = context.TODO()

	// get redis.Client and mock
	db, mock := redismock.NewClientMock()

	//the order of commands expected and executed must be the same
	//this is the default value
	mock.MatchExpectationsInOrder(true)

	//simple matching

	//retErr := errors.New("just test")
	//mock.ExpectHSet("key", "ggg", "sss").RedisNil()
	mock.ExpectHSet("key", "ggg", "sss").RedisNil()
	result := db.HSet(ctx, "key", "ggg", "sss").Err()
	assert.Equal(t, result, redis.Nil)
	//db.HSet(ctx, "key", "ggg", "sss").Err(), nil)
	//hget command return error
	mock.ExpectHGet("key", "field").SetErr(errors.New("error"))
	assert.Equal(t, db.HGet(ctx, "key", "field").Err(), errors.New("error"))

	//hget command return value
	mock.ExpectHGet("key", "field").SetVal("test value")
	assert.Equal(t, db.HGet(ctx, "key", "field").Val(), "test value")

	//hget command return redis.Nil
	mock.ExpectHGet("key", "field").RedisNil()
	assert.Equal(t, db.HGet(ctx, "key", "field").Err(), redis.Nil)

	//hget command... do not set return
	mock.ExpectHGet("key", "field")
	assert.NotEqual(t, db.HGet(ctx, "key", "field").Err(), nil)

	//------------

	//clean up all expectations
	//reset expected redis command
	mock.ClearExpect()
}
func TestNode_Append2(t *testing.T) {
	db, mock := redismock.NewClientMock()
	n := Node{Term: 8, Id: "0", state: StateLeader, rdb: db, CommitIndex: 0, ItemList: []*Item{{Index: 0, Term: 0, Log: ""}}}
	in := &rpc.AppendEntryRequest{Entry: &rpc.Entry{Term: 8, Index: 1, Value: "hello"}, PreTerm: 0, PreIndex: 0}
	newItem := &Item{
		Index: uint(in.Entry.Index),
		Term:  uint(in.Entry.GetTerm()),
		Log:   in.Entry.GetValue(),
	}
	newItemStr, _ := json.Marshal(newItem)
	mock.ExpectHSet(n.GetRedisKey(), fmt.Sprintf("%v", newItem.Index), newItemStr).RedisNil()
	//mock.Regexp().ExpectHSet("node_0_1", newItemStr, 0)
	reply, err := n.Append(context.Background(), in)
	assert.Nil(t, err)
	assert.Equal(t, true, reply.GetAccept())
	assert.Equal(t, 2, len(n.ItemList))
	assert.Equal(t, uint(1), n.GetTailItem().Index)
	assert.Equal(t, uint(8), n.GetTailItem().Term)
}
func TestNode_Append3(t *testing.T) {
	db, mock := redismock.NewClientMock()
	n := Node{Term: 8, Id: "0", state: StateFollower, rdb: db, CommitIndex: 0, ItemList: []*Item{{Index: 0, Term: 0, Log: ""}}}
	in := &rpc.AppendEntryRequest{Entry: &rpc.Entry{Term: 8, Index: 1, Value: "hello"}, PreTerm: 0, PreIndex: 0}
	newItem := &Item{
		Index: uint(in.Entry.Index),
		Term:  uint(in.Entry.GetTerm()),
		Log:   in.Entry.GetValue(),
	}
	newItemStr, _ := json.Marshal(newItem)
	mock.ExpectHSet(n.GetRedisKey(), fmt.Sprintf("%v", newItem.Index), newItemStr).RedisNil()
	reply, err := n.Append(context.Background(), in)
	assert.Nil(t, err)
	assert.Equal(t, true, reply.GetAccept())
	assert.Equal(t, 2, len(n.ItemList))
	assert.Equal(t, uint(1), n.GetTailItem().Index)
	assert.Equal(t, uint(8), n.GetTailItem().Term)
	assert.Equal(t, "hello", n.GetTailItem().Log)

	// 新的消息Term更大，使用更大的
	for _, st := range []NodeState{StateFollower, StateLeader, StateCandidate} {
		n = Node{Id: "1", rdb: db, Term: 8, state: st, CommitIndex: 0, ItemList: []*Item{{Index: 10, Term: 2, Log: "vvv"}}}
		in = &rpc.AppendEntryRequest{Entry: &rpc.Entry{Term: 9, Index: 11, Value: "hello"}, PreTerm: 2, PreIndex: 10, From: "8"}
		newItem = &Item{
			Index: uint(in.Entry.Index),
			Term:  uint(in.Entry.GetTerm()),
			Log:   in.Entry.GetValue(),
		}
		newItemStr, _ = json.Marshal(newItem)
		mock.ExpectHSet(n.GetRedisKey(), fmt.Sprintf("%v", newItem.Index), newItemStr).RedisNil()
		reply, err = n.Append(context.Background(), in)
		assert.Nil(t, err)
		assert.Equal(t, true, reply.GetAccept())
		assert.Equal(t, 2, len(n.ItemList))
		assert.Equal(t, uint(11), n.GetTailItem().Index)
		assert.Equal(t, uint(9), n.GetTailItem().Term)
		assert.Equal(t, "hello", n.GetTailItem().Log)
		assert.Equal(t, StateFollower, n.state)
		assert.Equal(t, uint(9), n.Term)
	}
}

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
		configIndex                        uint
		otherNodeList                      []*WorkNode
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
			want:    &rpc.AppendEntryReply{Accept: true, AppendIndex: 1},
			wantErr: false,
		},
		{
			name:   "leader_append_ok_2",
			fields: fields{Term: 8, state: StateLeader, CommitIndex: 3, ItemList: []*Item{{Index: 3, Term: 7, Log: ""}}},
			args: args{ctx: context.Background(), in: &rpc.AppendEntryRequest{
				Entry: &rpc.Entry{Term: 8, Index: 4, Value: "hello"}, PreTerm: 7, PreIndex: 3,
			}},
			want:    &rpc.AppendEntryReply{Accept: true, AppendIndex: 4, CommitIndex: 3},
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
			want:    &rpc.AppendEntryReply{Accept: true, AppendIndex: 4, CommitIndex: 3},
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
			want:    &rpc.AppendEntryReply{Accept: false, AppendIndex: 3, CommitIndex: 3},
			wantErr: false,
		},
		{
			name:   "follower_append_fail_2_PreIndex_error",
			fields: fields{Term: 8, state: StateFollower, CommitIndex: 3, ItemList: []*Item{{Index: 3, Term: 7, Log: ""}}},
			args: args{ctx: context.Background(), in: &rpc.AppendEntryRequest{
				Entry: &rpc.Entry{Term: 8, Index: 5, Value: "hello"}, PreTerm: 7, PreIndex: 4,
			}},
			want:    &rpc.AppendEntryReply{Accept: false, AppendIndex: 3, CommitIndex: 3},
			wantErr: false,
		},
		{
			name:   "follower_append_fail_3_Entry_Index_error",
			fields: fields{Term: 8, state: StateFollower, CommitIndex: 3, ItemList: []*Item{{Index: 3, Term: 7, Log: ""}}},
			args: args{ctx: context.Background(), in: &rpc.AppendEntryRequest{
				Entry: &rpc.Entry{Term: 8, Index: 6, Value: "hello"}, PreTerm: 7, PreIndex: 3,
			}},
			want:    &rpc.AppendEntryReply{Accept: false, AppendIndex: 3, CommitIndex: 3},
			wantErr: false,
		},
	}
	db, mock := redismock.NewClientMock()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{
				UnimplementedElectionServiceServer: tt.fields.UnimplementedElectionServiceServer,
				rdb:                                db,
				Id:                                 tt.fields.id,
				electing:                           tt.fields.electing,
				Term:                               tt.fields.Term,
				leaderId:                           tt.fields.leaderId,
				heartBeat:                          tt.fields.heartBeat,
				state:                              tt.fields.state,
				OathAcceptNum:                      tt.fields.OathAcceptNum,
				config:                             tt.fields.config,
				configIndex:                        tt.fields.configIndex,
				OtherNodeList:                      tt.fields.otherNodeList,
				ItemList:                           tt.fields.ItemList,
				CommitIndex:                        tt.fields.CommitIndex,
			}
			newItem := &Item{
				Index: uint(tt.args.in.Entry.Index),
				Term:  uint(tt.args.in.Entry.GetTerm()),
				Log:   tt.args.in.Entry.GetValue(),
			}
			newItemStr, _ := json.Marshal(newItem)
			mock.ExpectHSet(n.GetRedisKey(), fmt.Sprintf("%v", newItem.Index), newItemStr).RedisNil()
			got, err := n.Append(tt.args.ctx, tt.args.in)
			mock.ClearExpect()
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

func TestNode_Heart(t *testing.T) {
	type fields struct {
		id            string
		electing      bool
		Term          uint
		leaderId      string
		heartBeat     time.Time
		state         NodeState
		OathAcceptNum int
		config        []NodeConfig
		configIndex   uint
		otherNodeList []*WorkNode
		ItemList      []*Item
		CommitIndex   uint
	}
	type args struct {
		ctx context.Context
		in  *rpc.HeartRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *rpc.HeartReply
		wantErr bool
	}{
		{
			name:    "heart_ok",
			args:    args{ctx: context.Background(), in: &rpc.HeartRequest{Term: 1}},
			fields:  fields{id: "candidate", Term: 0, state: StateFollower, ItemList: []*Item{}},
			want:    &rpc.HeartReply{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{
				Id:            tt.fields.id,
				electing:      tt.fields.electing,
				Term:          tt.fields.Term,
				leaderId:      tt.fields.leaderId,
				heartBeat:     tt.fields.heartBeat,
				state:         tt.fields.state,
				OathAcceptNum: tt.fields.OathAcceptNum,
				config:        tt.fields.config,
				configIndex:   tt.fields.configIndex,
				OtherNodeList: tt.fields.otherNodeList,
				ItemList:      tt.fields.ItemList,
				CommitIndex:   tt.fields.CommitIndex,
			}
			got, err := n.Heart(tt.args.ctx, tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("Heart() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Heart() got = %v, want %v", got, tt.want)
			}
		})
	}
}
