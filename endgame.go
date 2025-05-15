package main

import (
	"encoding/json"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
)

func EndGameHandler(c *fiber.Ctx) error {
	// 1. Fetch gameStats from Redis
	val, err := redisClient.Get(ctx, "gameStats").Result()
	if err == redis.Nil {
		return c.Status(404).SendString("No game data found in Redis")
	} else if err != nil {
		return c.Status(500).SendString("Redis error: " + err.Error())
	}

	// 2. Parse JSON into generic map
	var gameStats map[string]interface{}
	if err := json.Unmarshal([]byte(val), &gameStats); err != nil {
		return c.Status(500).SendString("Failed to parse Redis JSON: " + err.Error())
	}

	// 3. Insert full gameStats into matches collection
	matchesColl := Client.Database("raidx").Collection("matches")
	if _, err := matchesColl.InsertOne(ctx, gameStats); err != nil {
		return c.Status(500).SendString("Failed to insert into matches: " + err.Error())
	}

	// 4. Update each player in players collection
	data := gameStats["data"].(map[string]interface{})
	playerStats := data["playerStats"].(map[string]interface{})
	playersColl := Client.Database("raidx").Collection("players")

	for id, raw := range playerStats {
		player := raw.(map[string]interface{})

		update := bson.M{
			"$inc": bson.M{
				"totalPoints":   int(player["totalPoints"].(float64)),
				"raidPoints":    int(player["raidPoints"].(float64)),
				"defencePoints": int(player["defencePoints"].(float64)),
			},
		}

		_, err := playersColl.UpdateByID(ctx, id, update)
		if err != nil {
			log.Printf("Warning: Failed to update player %s: %v", id, err)
		}
	}

	// 5. Optional: Clean up Redis key
	if err := redisClient.Del(ctx, "gameStats").Err(); err != nil {
		log.Printf("Warning: Failed to delete Redis key: %v", err)
	}

	return c.Redirect("/viewer") // or return a success message
}
