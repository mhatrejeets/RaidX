package handlers

import (
	"context"
	"crypto/sha1"
	"encoding/binary"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jcoene/go-base62"
	"github.com/mhatrejeets/RaidX/internal/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// User represents a player document in the DB
type User struct {
	FullName      string    `bson:"fullName"`
	Email         string    `bson:"email"`
	UserID        string    `bson:"userId"`
	Password      string    `bson:"password"`
	Position      string    `bson:"position"`
	CreatedAt     time.Time `bson:"createdAt"`
	TotalPoints   int       `bson:"totalPoints"`
	RaidPoints    int       `bson:"raidPoints"`
	DefencePoints int       `bson:"defencePoints"`
}

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
	form := new(User)
	if err := c.BodyParser(form); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("❌ Failed to parse form data")
	}

	// Check if the passwords match
	if strings.TrimSpace(c.FormValue("password")) != strings.TrimSpace(c.FormValue("confirmPassword")) {
		return c.Status(fiber.StatusBadRequest).SendString("❌ Passwords do not match")
	}

	// Access MongoDB collection
	collection := db.MongoClient.Database("raidx").Collection("players")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if the email already exists
	var existing User
	err := collection.FindOne(ctx, bson.M{"email": form.Email}).Decode(&existing)
	if err != mongo.ErrNoDocuments {
		return c.Status(fiber.StatusConflict).SendString("❌ Email already registered")
	}

	// Encode password with Base62 after hashing it
	encodedPassword := hashAndEncodeBase62(form.Password)

	// Create a new user entry
	newUser := User{
		FullName:      form.FullName,
		Email:         form.Email,
		UserID:        form.UserID,
		Password:      encodedPassword,
		Position:      form.Position,
		CreatedAt:     time.Now(),
		TotalPoints:   0,
		RaidPoints:    0,
		DefencePoints: 0,
	}

	// Insert the new user into the database
	_, err = collection.InsertOne(ctx, newUser)
	if err != nil {
		log.Printf("DB insert error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("❌ Could not store user")
	}

	// Return success message
	return c.Type("html").SendString(`
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
`)

}
