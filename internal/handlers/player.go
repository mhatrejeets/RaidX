package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/models"
	"go.mongodb.org/mongo-driver/bson"
)

// GetPlayerInvitationsHandler lists invitations for the logged-in player.
func GetPlayerInvitationsHandler(c *fiber.Ctx) error {
	userID, err := getUserIDFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	statusFilter := strings.ToLower(strings.TrimSpace(c.Query("status")))

	filter := bson.M{
		"type": models.InviteTypeTeam,
		"to_id": userID,
	}
	if statusFilter != "" {
		filter["status"] = statusFilter
	}

	invitesColl := db.MongoClient.Database("raidx").Collection("invitations")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := invitesColl.Find(ctx, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch invitations"})
	}

	var invites []models.Invitation
	if err := cursor.All(ctx, &invites); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode invitations"})
	}

	return c.JSON(invites)
}
