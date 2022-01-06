package raft

import (
	"testing"
)

func TestConfig_validate(t *testing.T) {
	type fields struct {
		ID            uint64
		ElectionTick  int
		HeartbeatTick int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{name: "id error", fields: fields{ID: 0}, wantErr: true},
		{name: "heartbeat tick error", fields: fields{ID: 1, HeartbeatTick: 0}, wantErr: true},
		{name: "heartbeat tick error", fields: fields{ID: 1, HeartbeatTick: -1}, wantErr: true},
		{name: "election tick error", fields: fields{ID: 1, HeartbeatTick: 10, ElectionTick: 9}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				ID:            tt.fields.ID,
				ElectionTick:  tt.fields.ElectionTick,
				HeartbeatTick: tt.fields.HeartbeatTick,
			}
			if err := c.validate(); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
