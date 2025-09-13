package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetTeamByID(c *fiber.Ctx) error {
	teamID := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(teamID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid team ID"})
	}

	collection := db.MongoClient.Database("raidx").Collection("teams")

	var result struct {
		ID       primitive.ObjectID `bson:"_id" json:"id"`
		TeamName string             `bson:"team_name" json:"team_name"`
		Players  []struct {
			ID   primitive.ObjectID `bson:"id" json:"id"`
			Name string             `bson:"name" json:"name"`
		} `bson:"players" json:"players"`
	}

	err = collection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&result)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Team not found"})
	}

	// Convert ObjectID to string for frontend
	type Player struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	players := make([]Player, len(result.Players))
	for i, p := range result.Players {
		players[i] = Player{
			ID:   p.ID.Hex(),
			Name: p.Name,
		}
	}

	return c.JSON(fiber.Map{
		"id":        result.ID.Hex(),
		"team_name": result.TeamName,
		"players":   players,
	})
}
