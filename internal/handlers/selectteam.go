package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

type Team struct {
	ID   string `json:"id" bson:"_id"`
	Name string `json:"team_name" bson:"team_name"`
}

func GetTeams(c *fiber.Ctx) error {
	collection := db.MongoClient.Database("raidx").Collection("teams")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		logrus.Error("Error:", "GetTeams:", " Failed to fetch teams: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch teams"})
	}

	var teams []Team

	if err = cursor.All(ctx, &teams); err != nil {
		logrus.Error("Error:", "GetTeams:", " Failed to decode teams: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to decode teams"})
	}

	return c.JSON(teams)
}
