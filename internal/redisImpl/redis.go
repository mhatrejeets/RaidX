package redisImpl

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// ListMatchKeys returns all Redis keys for ongoing matches with the given prefix
func ListMatchKeys(prefix string) ([]string, error) {
	if RedisClient == nil {
		return nil, nil
	}
	keys, err := RedisClient.Keys(ctx, prefix+"*").Result()
	return keys, err
}

var RedisClient *redis.Client
var RedisNull = redis.Nil
var ctx = context.Background()

func InitRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis server address
		Password: "",               // No password by default
		DB:       0,                // Default DB
	})

	// Test connection
	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis")
}

func SetRedisKey(key string, value interface{}) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		logrus.Error("Error:", "SetRedisKey:", " Failed to marshal value: %v", err)
		return err
	}

	return RedisClient.Set(ctx, key, jsonData, 10*time.Minute).Err() // Cache for 10 minutes
}

func GetRedisKey(key string, dest interface{}) error {
	val, err := RedisClient.Get(ctx, key).Result()
	if err != nil {
		logrus.Error("Error:", "GetRedisKey:", " Failed to get Redis key: %v", err)
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}
