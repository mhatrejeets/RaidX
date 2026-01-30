package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mhatrejeets/RaidX/internal/db"
	"go.mongodb.org/mongo-driver/bson"
)

func LogoutAllDevicesHandler(c *fiber.Ctx) error {
	// Extract user ID from token
	token := c.Get("Authorization")
	if token == "" {
		token = c.Cookies("token")
	}
	if token == "" {
		token = c.Query("token")
	}

	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing token"})
	}

	// Remove "Bearer " prefix if present
	if strings.HasPrefix(token, "Bearer ") {
		token = strings.TrimPrefix(token, "Bearer ")
	}

	// Parse JWT to get user ID
	jwtSecret := []byte("your-secret-key") // Should match your JWT secret
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	userID := claims["user_id"].(string)

	// Deactivate all sessions for this user
	sessionColl := db.MongoClient.Database("raidx").Collection("sessions")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := sessionColl.UpdateMany(ctx, bson.M{"user_id": userID}, bson.M{
		"$set": bson.M{
			"active":        false,
			"refresh_token": "", // Clear refresh token
		},
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to logout from all devices"})
	}

	// Clear cookies
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    "",
		HTTPOnly: true,
		Path:     "/",
		MaxAge:   -1,
	})
	c.Cookie(&fiber.Cookie{
		Name:     "refreshToken",
		Value:    "",
		HTTPOnly: true,
		Path:     "/",
		MaxAge:   -1,
	})

	return c.JSON(fiber.Map{
		"message":          "Logged out from all devices",
		"sessions_cleared": result.ModifiedCount,
	})
}
