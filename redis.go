package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client
var ctx = context.Background()

func InitRedis() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis server address
		Password: "",               // No password by default
		DB:       0,                // Default DB
	})

	// Test connection
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis")
}

func SetRedisKey(key string, value interface{}) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return redisClient.Set(ctx, key, jsonData, 10*time.Minute).Err() // Cache for 10 minutes
}

func GetRedisKey(key string, dest interface{}) error {
	val, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}
