package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var PlayerProfile struct {
	FullName      string    `bson:"fullName"`
	Email         string    `bson:"email"`
	UserId        string    `bson:"userId"`
	Position      string    `bson:"position"`
	CreatedAt     time.Time `bson:"createdAt"`
	TotalPoints   int       `bson:"totalPoints"`
	RaidPoints    int       `bson:"raidPoints"`
	DefencePoints int       `bson:"defencePoints"`
}

func PlayerProfileHandler(c *fiber.Ctx) error {
	// Get the player ID from the URL
	playerID := c.Params("id")

	// Convert string ID to ObjectId
	objID, err := primitive.ObjectIDFromHex(playerID)
	if err != nil {
		logrus.Warn("Warning:", "PlayerProfileHandler:", " Invalid player ID: %v", err)
		return c.Status(fiber.StatusBadRequest).SendString("Invalid player ID format")
	}

	// Define MongoDB collection
	collection := db.MongoClient.Database("raidx").Collection("players")

	// Query to find the player by ID

	// Find the player by ID
	filter := bson.M{"_id": objID}
	err = collection.FindOne(context.TODO(), filter).Decode(&PlayerProfile)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			logrus.Info("Info:", "PlayerProfileHandler:", " Player not found: %v", err)
			return c.Status(fiber.StatusNotFound).SendString("Player not found")
		}
		logrus.Error("Error:", "PlayerProfileHandler:", " Error fetching player data: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error fetching player data")
	}

	// Render the player profile HTML with the data
	return c.Render("playerprofile", fiber.Map{
		"ID":            playerID,
		"FullName":      PlayerProfile.FullName,
		"Email":         PlayerProfile.Email,
		"UserId":        PlayerProfile.UserId,
		"Position":      PlayerProfile.Position,
		"CreatedAt":     PlayerProfile.CreatedAt.Format("2006-01-02"), // Format for readability
		"TotalPoints":   PlayerProfile.TotalPoints,
		"RaidPoints":    PlayerProfile.RaidPoints,
		"DefencePoints": PlayerProfile.DefencePoints,
	})
}
