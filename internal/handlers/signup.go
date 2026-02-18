package handlers

import (
	"context"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jcoene/go-base62"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// hashAndEncodeBase62 hashes the input string and encodes it to base62
func hashAndEncodeBase62(input string) string {
	// Generate a SHA1 hash of the password
	hash := sha1.Sum([]byte(input))
	// Convert the first 8 bytes of the hash to an uint64
	num := binary.BigEndian.Uint64(hash[:8])

	// Cast the uint64 value to int64 (note: this might cause issues if num is too large for int64)
	return base62.Encode(int64(num)) // Ensure num fits within int64 range
}

func SignupHandler(c *fiber.Ctx) error {
	form := new(models.User)
	if err := c.BodyParser(form); err != nil {
		logrus.Error("Error:", "SignupHandler:", " Failed to parse form data: %v", err)
		return c.Status(fiber.StatusBadRequest).SendString("❌ Failed to parse form data")
	}

	form.Email = strings.ToLower(strings.TrimSpace(form.Email))
	form.UserID = strings.TrimSpace(form.UserID)
	if form.UserID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("❌ Username is required")
	}

	// Check if the passwords match
	if strings.TrimSpace(c.FormValue("password")) != strings.TrimSpace(c.FormValue("confirmPassword")) {
		logrus.Warn("Warning:", "SignupHandler:", " Passwords do not match for email: %s", form.Email)
		return c.Status(fiber.StatusBadRequest).SendString("❌ Passwords do not match")
	}

	// Access MongoDB collection
	collection := db.MongoClient.Database("raidx").Collection("players")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if the email already exists
	var existing models.User
	err := collection.FindOne(ctx, bson.M{"email": form.Email}).Decode(&existing)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		logrus.Error("Error:", "SignupHandler:", " Failed to check existing email: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("❌ Could not validate signup details")
	}
	if err == nil {
		logrus.Info("Info:", "SignupHandler:", " Email already registered: %s", form.Email)
		return c.Status(fiber.StatusConflict).SendString("❌ Email already registered")
	}

	usernameRegex := fmt.Sprintf("^%s$", regexp.QuoteMeta(form.UserID))
	err = collection.FindOne(ctx, bson.M{
		"userId": bson.M{"$regex": usernameRegex, "$options": "i"},
	}).Decode(&existing)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		logrus.Error("Error:", "SignupHandler:", " Failed to check existing username: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("❌ Could not validate signup details")
	}
	if err == nil {
		logrus.Info("Info:", "SignupHandler:", " Username already registered: %s", form.UserID)
		return c.Status(fiber.StatusConflict).SendString("❌ Username already taken")
	}

	// Encode password with Base62 after hashing it
	encodedPassword := hashAndEncodeBase62(form.Password)

	role := strings.ToLower(strings.TrimSpace(c.FormValue("role")))
	if role == "" {
		role = models.RolePlayer
	}
	if role != models.RolePlayer && role != models.RoleTeamOwner && role != models.RoleOrganizer {
		role = models.RolePlayer
	}

	// Create a new user entry
	newUser := models.User{
		FullName:  form.FullName,
		Email:     form.Email,
		UserID:    form.UserID,
		Password:  encodedPassword,
		Role:      role,
		CreatedAt: time.Now(),
	}

	// Only populate player-specific fields for players
	if role == models.RolePlayer {
		newUser.Position = form.Position
		newUser.TotalPoints = 0
		newUser.RaidPoints = 0
		newUser.DefencePoints = 0
	}

	// Insert the new user into the database
	_, err = collection.InsertOne(ctx, newUser)
	if err != nil {
		logrus.Error("Error:", "SignupHandler:", " Failed to insert user: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("❌ Could not store user")
	}

	// Return success message
	strr := `
	<!DOCTYPE html>
	<html>
	<head>
		<script>
			alert("✅ Signup successful!");
			window.location.href = "/";
		</script>
	</head>
	<body></body>
	</html>
	`
	return c.Type("html").SendString(strr)

}
