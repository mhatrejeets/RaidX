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

type organizerProfile struct {
	FullName  string    `bson:"fullName"`
	Email     string    `bson:"email"`
	UserID    string    `bson:"userId"`
	CreatedAt time.Time `bson:"createdAt"`
}

func OrganizerProfileHandler(c *fiber.Ctx) error {
	organizerID := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(organizerID)
	if err != nil {
		logrus.Warn("OrganizerProfileHandler: Invalid organizer ID:", err)
		return c.Status(fiber.StatusBadRequest).SendString("Invalid organizer ID format")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	playersColl := db.MongoClient.Database("raidx").Collection("players")
	var profile organizerProfile
	if err := playersColl.FindOne(ctx, bson.M{"_id": objID}).Decode(&profile); err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).SendString("Organizer not found")
		}
		return c.Status(fiber.StatusInternalServerError).SendString("Error fetching organizer data")
	}

	eventsColl := db.MongoClient.Database("raidx").Collection("events")
	totalEvents, _ := eventsColl.CountDocuments(ctx, bson.M{"organizer_id": objID})

	return c.Render("organizerprofile", fiber.Map{
		"ID":          organizerID,
		"FullName":    profile.FullName,
		"Email":       profile.Email,
		"UserId":      profile.UserID,
		"CreatedAt":   profile.CreatedAt.Format("2006-01-02"),
		"TotalEvents": totalEvents,
	})
}
