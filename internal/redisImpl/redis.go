package redisImpl

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

var RedisClient *redis.Client
var RedisNull = redis.Nil
var ctx = context.Background()

func InitRedis() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379" // fallback default
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}
	RedisClient = redis.NewClient(opts)

	// Test connection
	pong, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis", pong)
}

func SetRedisKey(key string, value interface{}) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		logrus.Error("Error:", "SetRedisKey:", " Failed to marshal value: %v", err)
		return err
	}

	// Persist the game state without an expiry so it survives client refreshes.
	return RedisClient.Set(ctx, key, jsonData, 0).Err()
}

func GetRedisKey(key string, dest interface{}) error {
	val, err := RedisClient.Get(ctx, key).Result()
	if err != nil {
		// Don't flood logs for the common 'key not found' case (redis.Nil)
		if err == RedisNull {
			return err
		}
		logrus.Error("Error:", "GetRedisKey:", " Failed to get Redis key: %v", err)
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}
