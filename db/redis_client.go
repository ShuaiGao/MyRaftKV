package db

import "github.com/go-redis/redis/v8"

type RedisConfig struct {
	Address  string `json:"address"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

var rdb *redis.Client

func GetRedisDB() *redis.Client {
	return rdb
}

func ConnectRedis(cfg RedisConfig) {
	rdb = redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}
