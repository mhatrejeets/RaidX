package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Struct matching MongoDB document
type Userr struct {
	Email    string `bson:"email"`
	Password string `bson:"password"`
}

func LoginHandler(c *fiber.Ctx) error {
	// Get form fields (not JSON!)
	email := strings.ToLower(strings.TrimSpace(c.FormValue("email")))
	password := strings.TrimSpace(c.FormValue("password"))
	encodedPassword := hashAndEncodeBase62(password)

	fmt.Println("Login Attempt => Email:", email, "EncodedPassword:", encodedPassword)

	collection := Client.Database("raidx").Collection("players")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user Userr
	err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusUnauthorized).SendString("❌ Email not registered")
		}
		log.Printf("DB error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("❌ Server error")
	}

	if user.Password != encodedPassword {
		return c.Status(fiber.StatusUnauthorized).SendString("❌ Incorrect password")
	}

	// Success
	return c.Type("html").SendString(`
		<!DOCTYPE html>
		<html>
		<head>
			<script>
				alert("✅ Login successful!");
				window.location.href = "/start";
			</script>
		</head>
		<body></body>
		</html>
	`)
}
