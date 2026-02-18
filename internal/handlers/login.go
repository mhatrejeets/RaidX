package handlers

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func LoginHandler(c *fiber.Ctx) error {

	// Support both JSON and form data
	var loginData struct {
		Identifier string `json:"identifier" form:"identifier"`
		Email      string `json:"email" form:"email"`
		Password   string `json:"password" form:"password"`
	}

	// Try parsing JSON first
	if err := c.BodyParser(&loginData); err != nil {
		// If JSON parsing fails, try form data
		loginData.Identifier = c.FormValue("identifier")
		loginData.Email = c.FormValue("email")
		loginData.Password = c.FormValue("password")
	}

	identifier := strings.TrimSpace(loginData.Identifier)
	if identifier == "" {
		identifier = strings.TrimSpace(loginData.Email)
	}

	// Validate required fields
	if identifier == "" || loginData.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Username/email and password are required",
		})
	}

	email := strings.ToLower(identifier)
	password := strings.TrimSpace(loginData.Password)
	encodedPassword := hashAndEncodeBase62(password)

	collection := db.MongoClient.Database("raidx").Collection("players")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.Userr
	err := collection.FindOne(ctx, bson.M{"$or": []bson.M{
		{"email": email},
		{"userId": bson.M{"$regex": fmt.Sprintf("^%s$", regexp.QuoteMeta(identifier)), "$options": "i"}},
	}}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Username or email not registered"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Server error"})
	}
	if user.Password != encodedPassword {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Incorrect password"})
	}

	role := strings.ToLower(strings.TrimSpace(user.Role))
	if role == "" {
		role = models.RolePlayer
	}

	// Extract device information
	userAgent := c.Get("User-Agent")
	ipAddress := c.IP()
	deviceID := hashDeviceFingerprint(userAgent) // Create device fingerprint

	sessionColl := db.MongoClient.Database("raidx").Collection("sessions")

	// Check if this device already has an active session for this user
	existingSession := models.Session{}
	err = sessionColl.FindOne(ctx, bson.M{
		"user_id":   user.ID.Hex(),
		"device_id": deviceID,
		"active":    true,
	}).Decode(&existingSession)

	if err == nil && time.Now().Before(existingSession.RefreshExpiryTime) {
		// Existing valid session found for this device
		// But check if JWT token is still valid (not expired)
		if time.Now().Before(existingSession.ExpiryTime) {
			// JWT token still valid - reuse existing session
			_, err := sessionColl.UpdateOne(ctx, bson.M{"_id": existingSession.ID},
				bson.M{"$set": bson.M{"last_used_at": time.Now()}})
			if err == nil {
				// Return existing tokens
				c.Cookie(&fiber.Cookie{
					Name:     "token",
					Value:    existingSession.JWTToken,
					HTTPOnly: true,
					Path:     "/",
					SameSite: "Lax",
					Expires:  existingSession.ExpiryTime,
				})
				c.Cookie(&fiber.Cookie{
					Name:     "refreshToken",
					Value:    existingSession.RefreshToken,
					HTTPOnly: true,
					Path:     "/",
					SameSite: "Lax",
					Expires:  existingSession.RefreshExpiryTime,
				})

				return c.JSON(fiber.Map{
					"token":         existingSession.JWTToken,
					"refresh_token": existingSession.RefreshToken,
					"user_id":       user.ID.Hex(),
					"name":          user.Name,
					"role":          user.Role,
					"expires":       existingSession.ExpiryTime.Unix(),
					"reused":        true,
				})
			}
		}
		// JWT token expired but refresh token valid - generate new JWT
	}

	// New device: Check active session count for this user (max 5)
	activeSessions, err := sessionColl.CountDocuments(ctx, bson.M{
		"user_id": user.ID.Hex(),
		"active":  true,
	})
	if err == nil && activeSessions >= 5 {
		// Invalidate oldest session
		oldestSession := models.Session{}
		opts := options.FindOne().SetSort(bson.M{"created_at": 1})
		sessionColl.FindOne(ctx, bson.M{
			"user_id": user.ID.Hex(),
			"active":  true,
		}, opts).Decode(&oldestSession)

		if oldestSession.ID != primitive.NilObjectID {
			sessionColl.UpdateOne(ctx, bson.M{"_id": oldestSession.ID},
				bson.M{"$set": bson.M{"active": false}})
		}
	}

	// Generate JWT
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	sessionID := primitive.NewObjectID().Hex()
	expiry := time.Now().Add(time.Hour)
	claims := jwt.MapClaims{
		"user_id":    user.ID.Hex(),
		"role":       role,
		"session_id": sessionID,
		"exp":        expiry.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(jwtSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate JWT"})
	}

	// Generate refresh token
	refreshToken := primitive.NewObjectID().Hex()
	refreshExpiry := time.Now().Add(7 * 24 * time.Hour) // 7 days

	// Create new session with device tracking
	session := models.Session{
		SessionID:         sessionID,
		UserID:            user.ID.Hex(),
		JWTToken:          tokenStr,
		LoginTime:         time.Now(),
		ExpiryTime:        expiry,
		RefreshToken:      refreshToken,
		RefreshExpiryTime: refreshExpiry,
		Active:            true,
		DeviceID:          deviceID,
		UserAgent:         userAgent,
		IPAddress:         ipAddress,
		CreatedAt:         time.Now(),
		LastUsedAt:        time.Now(),
	}
	_, err = sessionColl.InsertOne(ctx, session)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to store session"})
	}

	// Success: set access JWT as HttpOnly cookie and return both tokens
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    tokenStr,
		HTTPOnly: true,
		Path:     "/",
		SameSite: "Lax",
		Expires:  expiry,
	})

	// Also set refresh token cookie (HttpOnly)
	c.Cookie(&fiber.Cookie{
		Name:     "refreshToken",
		Value:    refreshToken,
		HTTPOnly: true,
		Path:     "/",
		SameSite: "Lax",
		Expires:  refreshExpiry,
	})

	return c.JSON(fiber.Map{
		"token":         tokenStr,
		"refresh_token": refreshToken,
		"user_id":       user.ID.Hex(),
		"name":          user.Name,
		"role":          user.Role,
		"expires":       expiry.Unix(),
	})

}

// hashDeviceFingerprint creates a unique device identifier from user agent and other factors
func hashDeviceFingerprint(userAgent string) string {
	hash := md5.Sum([]byte(userAgent))
	return fmt.Sprintf("%x", hash)
}
