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

type ownerProfile struct {
    FullName  string    `bson:"fullName"`
    Email     string    `bson:"email"`
    UserID    string    `bson:"userId"`
    CreatedAt time.Time `bson:"createdAt"`
}

func OwnerProfileHandler(c *fiber.Ctx) error {
    ownerID := c.Params("id")
    objID, err := primitive.ObjectIDFromHex(ownerID)
    if err != nil {
        logrus.Warn("OwnerProfileHandler: Invalid owner ID:", err)
        return c.Status(fiber.StatusBadRequest).SendString("Invalid owner ID format")
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    playersColl := db.MongoClient.Database("raidx").Collection("players")
    var profile ownerProfile
    if err := playersColl.FindOne(ctx, bson.M{"_id": objID}).Decode(&profile); err != nil {
        if err == mongo.ErrNoDocuments {
            return c.Status(fiber.StatusNotFound).SendString("Owner not found")
        }
        return c.Status(fiber.StatusInternalServerError).SendString("Error fetching owner data")
    }

    teamsColl := db.MongoClient.Database("raidx").Collection("rbac_teams")
    totalTeams, _ := teamsColl.CountDocuments(ctx, bson.M{"owner_id": objID})

    return c.Render("ownerprofile", fiber.Map{
        "ID":         ownerID,
        "FullName":   profile.FullName,
        "Email":      profile.Email,
        "UserId":     profile.UserID,
        "CreatedAt":  profile.CreatedAt.Format("2006-01-02"),
        "TotalTeams": totalTeams,
    })
}
