package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func PlayerProfileHandler(c *fiber.Ctx) error {
	// Get the player ID from the URL
	playerID := c.Params("id")

	// Convert string ID to ObjectId
	objID, err := primitive.ObjectIDFromHex(playerID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid player ID format")
	}

	// Define MongoDB collection
	collection := db.MongoClient.Database("raidx").Collection("players")

	// Find the player by ID
	filter := bson.M{"_id": objID}
	err = collection.FindOne(context.TODO(), filter).Decode(&models.Player)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).SendString("Player not found")
		}
		return c.Status(fiber.StatusInternalServerError).SendString("Error fetching player data")
	}

	// Render the player profile HTML with the data
	return c.Render("playerprofile", fiber.Map{
		"ID":            playerID,
		"FullName":      models.Player.FullName,
		"Email":         models.Player.Email,
		"UserId":        models.Player.UserId,
		"Position":      models.Player.Position,
		"CreatedAt":     models.Player.CreatedAt.Format("2006-01-02"), // Format for readability
		"TotalPoints":   models.Player.TotalPoints,
		"RaidPoints":    models.Player.RaidPoints,
		"DefencePoints": models.Player.DefencePoints,
	})
}
