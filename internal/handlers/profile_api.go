package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetMyProfileHandler returns the authenticated user's profile as JSON.
func GetMyProfileHandler(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	playersColl := db.MongoClient.Database("raidx").Collection("players")
	var doc bson.M
	if err := playersColl.FindOne(ctx, bson.M{"_id": userID}).Decode(&doc); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Profile not found"})
	}

	response := fiber.Map{
		"id":                userID.Hex(),
		"fullName":          getStringField(doc, "fullName"),
		"email":             getStringField(doc, "email"),
		"userId":            getStringField(doc, "userId"),
		"role":              getStringField(doc, "role"),
		"position":          getStringField(doc, "position"),
		"createdAt":         doc["createdAt"],
		"totalPoints":       getIntField(doc, "totalPoints"),
		"raidPoints":        getIntField(doc, "raidPoints"),
		"defencePoints":     getIntField(doc, "defencePoints"),
		"superRaids":        getIntField(doc, "superRaids"),
		"superTackles":      getIntField(doc, "superTackles"),
		"totalRaids":        getIntField(doc, "totalRaids"),
		"successfulRaids":   getIntField(doc, "successfulRaids"),
		"totalTackles":      getIntField(doc, "totalTackles"),
		"successfulTackles": getIntField(doc, "successfulTackles"),
		"matchesPlayed":     getIntField(doc, "matchesPlayed"),
		"mvpCount":          getIntField(doc, "mvpCount"),
		"bestRaiderCount":   getIntField(doc, "bestRaiderCount"),
		"bestDefenderCount": getIntField(doc, "bestDefenderCount"),
	}

	if response["role"] == "" {
		if roleVal, ok := c.Locals("role").(string); ok {
			response["role"] = roleVal
		}
	}

	return c.JSON(response)
}

func getStringField(doc bson.M, key string) string {
	if value, ok := doc[key].(string); ok {
		return value
	}
	return ""
}

func getIntField(doc bson.M, key string) int {
	switch value := doc[key].(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}
