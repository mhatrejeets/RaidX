package handlers

import (
	"context"
	"math/rand"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mhatrejeets/RaidX/internal/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func RefreshTokenHandler(c *fiber.Ctx) error {
	// Get refresh token from cookie or query param
	refreshToken := c.Cookies("refreshToken")
	if refreshToken == "" {
		refreshToken = c.Query("refresh_token")
	}
	if refreshToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing refresh token"})
	}

	sessionColl := db.MongoClient.Database("raidx").Collection("sessions")

	// Find session by refresh token
	var session struct {
		SessionID         string    `bson:"session_id"`
		UserID            string    `bson:"user_id"`
		RefreshToken      string    `bson:"refresh_token"`
		RefreshExpiryTime time.Time `bson:"refresh_expiry_time"`
		Active            bool      `bson:"active"`
	}

	err := sessionColl.FindOne(context.TODO(), bson.M{"refresh_token": refreshToken}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid refresh token"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database error"})
	}

	// Validate session is active and refresh token not expired
	if !session.Active {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Session is inactive"})
	}
	if time.Now().After(session.RefreshExpiryTime) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Refresh token expired"})
	}

	// Issue new access JWT (15-minute duration)
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	newExpiry := time.Now().Add(15 * time.Minute)
	claims := jwt.MapClaims{
		"user_id":    session.UserID,
		"role":       "user",
		"session_id": session.SessionID,
		"exp":        newExpiry.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	newTokenStr, err := token.SignedString(jwtSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate JWT"})
	}

	// Issue new refresh token (rotate it)
	newRefreshToken := generateRandomToken()
	newRefreshExpiry := time.Now().Add(7 * 24 * time.Hour)

	// Update session with new tokens and last_used_at timestamp
	update := bson.M{
		"$set": bson.M{
			"jwt_token":           newTokenStr,
			"expiry_time":         newExpiry,
			"refresh_token":       newRefreshToken,
			"refresh_expiry_time": newRefreshExpiry,
			"last_used_at":        time.Now(),
		},
	}
	_, err = sessionColl.UpdateOne(context.TODO(), bson.M{"session_id": session.SessionID}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update session"})
	}

	// Set new access token cookie
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    newTokenStr,
		HTTPOnly: true,
		Path:     "/",
		SameSite: "Lax",
		Expires:  newExpiry,
	})

	// Set new refresh token cookie
	c.Cookie(&fiber.Cookie{
		Name:     "refreshToken",
		Value:    newRefreshToken,
		HTTPOnly: true,
		Path:     "/",
		SameSite: "Lax",
		Expires:  newRefreshExpiry,
	})

	return c.JSON(fiber.Map{
		"token":         newTokenStr,
		"refresh_token": newRefreshToken,
		"expires":       newExpiry.Unix(),
	})
}

// generateRandomToken creates a random alphanumeric token
func generateRandomToken() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 32)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
