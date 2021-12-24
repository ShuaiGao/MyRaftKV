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
	assert.Equal(t, NodeStateWorking, n.GetState())
	assert.Equal(t, "888", n.leaderId)
}

func TestNode_Oath(t *testing.T) {
	type fields struct {
		UnimplementedElectionServiceServer *rpc.UnimplementedElectionServiceServer
		id                                 string
		electing                           bool
		lock                               sync.RWMutex
		Term                               uint
		leaderId                           string
		heartBeat                          time.Time
		state                              NodeState
		role                               NodeRole
		OathAcceptNum                      int
		config                             []NodeConfig
		configIndex                        int
		otherNodeList                      []Node
	}
	ServerNode := fields{
		UnimplementedElectionServiceServer: &rpc.UnimplementedElectionServiceServer{},
		id:                                 "candidate",
		Term:                               0,
		state:                              NodeStateWorking,
	}
	type args struct {
		ctx context.Context
		in  *rpc.OathRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *rpc.OathReply
		wantErr bool
	}{
		{
			name:    "",
			args:    args{ctx: context.Background(), in: &rpc.OathRequest{}},
			fields:  ServerNode,
			want:    &rpc.OathReply{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{
				UnimplementedElectionServiceServer: tt.fields.UnimplementedElectionServiceServer,
				id:                                 tt.fields.id,
				electing:                           tt.fields.electing,
				lock:                               tt.fields.lock,
				Term:                               tt.fields.Term,
				leaderId:                           tt.fields.leaderId,
				heartBeat:                          tt.fields.heartBeat,
				state:                              tt.fields.state,
				role:                               tt.fields.role,
				OathAcceptNum:                      tt.fields.OathAcceptNum,
				config:                             tt.fields.config,
				configIndex:                        tt.fields.configIndex,
			}
			got, err := n.Oath(tt.args.ctx, tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("Oath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Oath() got = %v, want %v", got, tt.want)
			}
		})
	}
}
