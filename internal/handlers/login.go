package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/auth"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func LoginHandler(c *fiber.Ctx) error {
	// Get form fields (not JSON!)
	email := strings.ToLower(strings.TrimSpace(c.FormValue("email")))
	password := strings.TrimSpace(c.FormValue("password"))
	encodedPassword := hashAndEncodeBase62(password)

	fmt.Println("Login Attempt => Email:", email, "EncodedPassword:", encodedPassword)

	collection := db.MongoClient.Database("raidx").Collection("players")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.Userr
	err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			logrus.Info("Info:", "LoginHandler:", " Email not registered: %v", err)
			return c.Status(fiber.StatusUnauthorized).SendString("❌ Email not registered")
		}
		logrus.Error("Error:", "LoginHandler:", " DB error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("❌ Server error")
	}

	if user.Password != encodedPassword {
		logrus.Info("Info:", "LoginHandler:", " Incorrect password for email: %s", email)
		return c.Status(fiber.StatusUnauthorized).SendString("❌ Incorrect password")
	}

	// Success
	logrus.Info("Info:", "LoginHandler:", " User logged in successfully: %s", email)
	// Generate JWT token
	token, err := auth.GenerateJWT(user.ID.Hex(), "player")
	if err != nil {
		logrus.Error("Error:", "LoginHandler:", " JWT generation failed: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("❌ Could not generate token")
	}
	// Return HTML with redirect and store JWT in localStorage
	return c.Type("html").SendString(fmt.Sprintf(`
	   <!DOCTYPE html>
	   <html>
	   <head>
	   <script>
		   localStorage.setItem('jwt', '%s');
		   // Optionally set a cookie for server-side auth if needed
		   document.cookie = "jwt=%s; path=/";
		   alert("✅ Login successful!");
		   window.location.href = "/home1/%s?name=%s";
	   </script>
	   </head>
	   <body></body>
	   </html>
	   `, token, token, user.ID.Hex(), user.Name))

}
