package handlers

import (
	"context"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/redisImpl"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func EndGameHandler(c *fiber.Ctx) error {
	ctx := context.Background()
	// 1. Fetch gameStats from Redis
	val, err := redisImpl.RedisClient.Get(ctx, "gameStats").Result()
	logrus.Info("EndGame Handler is invoked")
	if err == redisImpl.RedisNull {
		logrus.Warn("Warning:", "EndGameHandler:", " No game data found in Redis")
		return c.Status(404).SendString("No game data found in Redis")
	} else if err != nil {
		logrus.Error("Error:", "EndGameHandler:", " Redis error: %v", err)
		return c.Status(500).SendString("Redis error: " + err.Error())
	}

	logrus.Debug("EndGame Handler will delete ", val)

	// 2. Parse JSON into generic map
	var gameStats map[string]interface{}
	if err := json.Unmarshal([]byte(val), &gameStats); err != nil {
		logrus.Error("Error:", "EndGameHandler:", " Failed to parse Redis JSON: %v", err)
		return c.Status(500).SendString("Failed to parse Redis JSON: " + err.Error())
	}

	// 3. Insert full gameStats into matches collection
	matchesColl := db.MongoClient.Database("raidx").Collection("matches")
	logrus.Debug("EndGame Handler will insert ", gameStats)
	if _, err := matchesColl.InsertOne(ctx, gameStats); err != nil {
		logrus.Error("Error:", "EndGameHandler:", " Failed to insert into matches: %v", err)
		return c.Status(500).SendString("Failed to insert into matches: " + err.Error())
	}

	// 4. Update each player in players collection
	data := gameStats["data"].(map[string]interface{})
	playerStats := data["playerStats"].(map[string]interface{})
	playersColl := db.MongoClient.Database("raidx").Collection("players")

	for id, raw := range playerStats {
		player := raw.(map[string]interface{})

		update := bson.M{
			"$inc": bson.M{
				"totalPoints":   int(player["totalPoints"].(float64)),
				"raidPoints":    int(player["raidPoints"].(float64)),
				"defencePoints": int(player["defencePoints"].(float64)),
			},
		}

		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			logrus.Error("Error:", "EndGameHandler:", " Invalid player ID: %v", err)
			continue
		}

		_, err = playersColl.UpdateByID(ctx, objID, update)
		if err != nil {
			logrus.Error("Error:", "EndGameHandler:", " Failed to update player %s: %v", id, err)
		}

	}

	// 5. Optional: Clean up Redis key
	if err := redisImpl.RedisClient.Del(ctx, "gameStats").Err(); err != nil {
		logrus.Error("Error:", "EndGameHandler:", " Failed to delete Redis key: %v", err)
	}

	return c.Redirect("/matches") // or return a success message
}
