package main

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func playerprofileHandler(c *fiber.Ctx) error {
	// Get the player ID from the URL
	playerID := c.Params("id")

	// Convert string ID to ObjectId
	objID, err := primitive.ObjectIDFromHex(playerID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid player ID format")
	}

	// Define MongoDB collection
	collection := Client.Database("raidx").Collection("players")

	// Query to find the player by ID
	var player struct {
		FullName      string    `bson:"fullName"`
		Email         string    `bson:"email"`
		UserId        string    `bson:"userId"`
		Position      string    `bson:"position"`
		CreatedAt     time.Time `bson:"createdAt"`
		TotalPoints   int       `bson:"totalPoints"`
		RaidPoints    int       `bson:"raidPoints"`
		DefencePoints int       `bson:"defencePoints"`
	}

	// Find the player by ID
	filter := bson.M{"_id": objID}
	err = collection.FindOne(context.TODO(), filter).Decode(&player)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).SendString("Player not found")
		}
		return c.Status(fiber.StatusInternalServerError).SendString("Error fetching player data")
	}

	// Render the player profile HTML with the data
	return c.Render("playerprofile", fiber.Map{
		"ID":			playerID,
		"FullName":      player.FullName,
		"Email":         player.Email,
		"UserId":        player.UserId,
		"Position":      player.Position,
		"CreatedAt":     player.CreatedAt.Format("2006-01-02"), // Format for readability
		"TotalPoints":   player.TotalPoints,
		"RaidPoints":    player.RaidPoints,
		"DefencePoints": player.DefencePoints,
	})
}
