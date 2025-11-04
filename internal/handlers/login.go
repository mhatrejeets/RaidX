package handlers

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func LoginHandler(c *fiber.Ctx) error {
	// Support both JSON and form data
	var loginData struct {
		Email    string `json:"email" form:"email"`
		Password string `json:"password" form:"password"`
	}

	// Try parsing JSON first
	if err := c.BodyParser(&loginData); err != nil {
		// If JSON parsing fails, try form data
		loginData.Email = c.FormValue("email")
		loginData.Password = c.FormValue("password")
	}

	// Validate required fields
	if loginData.Email == "" || loginData.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email and password are required",
		})
	}

	email := strings.ToLower(strings.TrimSpace(loginData.Email))
	password := strings.TrimSpace(loginData.Password)
	encodedPassword := hashAndEncodeBase62(password)

	collection := db.MongoClient.Database("raidx").Collection("players")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.Userr
	err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Email not registered"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Server error"})
	}
	if user.Password != encodedPassword {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Incorrect password"})
	}

	// Generate JWT
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	sessionID := primitive.NewObjectID().Hex()
	expiry := time.Now().Add(time.Hour)
	claims := jwt.MapClaims{
		"user_id":    user.ID.Hex(),
		"role":       "user",
		"session_id": sessionID,
		"exp":        expiry.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(jwtSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate JWT"})
	}

	// Store session in MongoDB
	sessionColl := db.MongoClient.Database("raidx").Collection("sessions")
	session := models.Session{
		SessionID:  sessionID,
		UserID:     user.ID.Hex(),
		JWTToken:   tokenStr,
		LoginTime:  time.Now(),
		ExpiryTime: expiry,
		Active:     true,
	}
	_, err = sessionColl.InsertOne(ctx, session)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to store session"})
	}

	// Success: send JWT to frontend
	return c.JSON(fiber.Map{
		"token":   tokenStr,
		"user_id": user.ID.Hex(),
		"name":    user.Name,
		"expires": expiry.Unix(),
	})

}
