package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetPlayerInvitationsHandler lists invitations for the logged-in player.
func GetPlayerInvitationsHandler(c *fiber.Ctx) error {
	userID, err := getUserIDFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	statusFilter := strings.ToLower(strings.TrimSpace(c.Query("status")))

	filter := bson.M{
		"type":  models.InviteTypeTeam,
		"to_id": userID,
	}
	if statusFilter != "" {
		filter["status"] = statusFilter
	}

	invitesColl := db.MongoClient.Database("raidx").Collection("invitations")
	teamsColl := db.MongoClient.Database("raidx").Collection("rbac_teams")
	playersColl := db.MongoClient.Database("raidx").Collection("players")
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

	response := make([]fiber.Map, 0, len(invites))
	for _, invite := range invites {
		item := fiber.Map{
			"id":        invite.ID.Hex(),
			"type":      invite.Type,
			"status":    invite.Status,
			"declineReason": invite.DeclineReason,
			"teamId":    "",
			"teamName":  "",
			"ownerName": "",
			"createdAt": invite.CreatedAt,
			"expiresAt": invite.ExpiresAt,
		}

		if invite.TeamID != nil && invite.TeamID != &primitive.NilObjectID {
			item["teamId"] = invite.TeamID.Hex()

			var teamDoc bson.M
			if err := teamsColl.FindOne(ctx, bson.M{"_id": *invite.TeamID}).Decode(&teamDoc); err == nil {
				if name, ok := teamDoc["teamName"].(string); ok && name != "" {
					item["teamName"] = name
				} else if name, ok := teamDoc["team_name"].(string); ok && name != "" {
					item["teamName"] = name
				}

				if ownerOID, ok := teamDoc["owner_id"].(primitive.ObjectID); ok {
					var ownerDoc bson.M
					if err := playersColl.FindOne(ctx, bson.M{"_id": ownerOID}).Decode(&ownerDoc); err == nil {
						if ownerName, ok := ownerDoc["fullName"].(string); ok && ownerName != "" {
							item["ownerName"] = ownerName
						}
					}
				}
			}
		}

		response = append(response, item)
	}

	return c.JSON(response)
}
