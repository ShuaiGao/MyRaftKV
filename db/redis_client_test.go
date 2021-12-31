package db

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestConnectRedis(t *testing.T) {
	type args struct {
		cfg RedisConfig
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "just_connect", args: args{cfg: RedisConfig{Address: "localhost:6379", DB: 0}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ConnectRedis(tt.args.cfg)
			rdb := GetRedisDB()
			ctx := context.Background()
			for {
				err := rdb.Set(ctx, "test_key", 1, 0).Err()
				if err == nil {
					fmt.Println("set ok")
				} else {
					fmt.Println("set error")
					fmt.Printf("set error:%v", err)
				}
				time.Sleep(time.Second)
			}
		})
	}
}
