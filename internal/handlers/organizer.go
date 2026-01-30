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
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateEventHandler(c *fiber.Ctx) error {
	organizerID, err := getUserIDFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	var req struct {
		EventName string `json:"event_name" form:"event_name"`
		EventType string `json:"event_type" form:"event_type"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	eventName := strings.TrimSpace(req.EventName)
	if eventName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "event_name is required"})
	}

	eventType := strings.ToLower(strings.TrimSpace(req.EventType))
	if eventType == "" {
		eventType = models.EventTypeMatch
	}
	if eventType != models.EventTypeMatch && eventType != models.EventTypeTournament && eventType != models.EventTypeChampionship {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid event_type"})
	}

	eventsColl := db.MongoClient.Database("raidx").Collection("events")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	event := models.Event{
		OrganizerID:        organizerID,
		EventName:          eventName,
		EventType:          eventType,
		ParticipatingTeams: []models.EventTeamEntry{},
		Status:             models.EventStatusDraft,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	result, err := eventsColl.InsertOne(ctx, event)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create event"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"event_id":   result.InsertedID,
		"event_name": eventName,
		"event_type": eventType,
	})
}

func CreateEventInviteHandler(c *fiber.Ctx) error {
	organizerID, err := getUserIDFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	eventID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid event id"})
	}

	var req struct {
		TeamID        string `json:"team_id" form:"team_id"`
		GenerateLink  bool   `json:"generate_link" form:"generate_link"`
		ExpiresInDays int    `json:"expires_in_days" form:"expires_in_days"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	eventsColl := db.MongoClient.Database("raidx").Collection("events")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ensure event belongs to organizer
	if err := eventsColl.FindOne(ctx, bson.M{"_id": eventID, "organizer_id": organizerID}).Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Event not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch event"})
	}

	teamOID := primitive.NilObjectID
	teamOwnerID := primitive.NilObjectID
	if !req.GenerateLink {
		if strings.TrimSpace(req.TeamID) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "team_id is required unless generate_link is true"})
		}
		var err error
		teamOID, err = primitive.ObjectIDFromHex(req.TeamID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid team_id"})
		}

		teamsColl := db.MongoClient.Database("raidx").Collection("rbac_teams")
		var team models.TeamProfile
		if err := teamsColl.FindOne(ctx, bson.M{"_id": teamOID, "status": models.TeamStatusActive}).Decode(&team); err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Team not found"})
		}
		teamOwnerID = team.OwnerID

		// Add team as invited to event
		_, _ = eventsColl.UpdateOne(ctx, bson.M{"_id": eventID}, bson.M{
			"$addToSet": bson.M{"participating_teams": models.EventTeamEntry{
				TeamID: teamOID,
				Status: models.EventTeamStatusInvited,
			}},
			"$set": bson.M{"updated_at": time.Now()},
		})
	}

	expiresIn := 30
	if req.ExpiresInDays > 0 {
		expiresIn = req.ExpiresInDays
	}

	inviteToken := generateRandomToken()
	invitesColl := db.MongoClient.Database("raidx").Collection("invitations")

	invitation := models.Invitation{
		Type:        models.InviteTypeEvent,
		FromID:      organizerID,
		ToID:        teamOwnerID,
		EventID:     &eventID,
		InviteToken: inviteToken,
		Status:      models.InviteStatusPending,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(time.Duration(expiresIn) * 24 * time.Hour),
	}
	if teamOID != primitive.NilObjectID {
		invitation.TeamID = &teamOID
	}

	result, err := invitesColl.InsertOne(ctx, invitation)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create invitation"})
	}

	inviteURL := c.BaseURL() + "/invite/event/" + inviteToken
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"invitation_id": result.InsertedID,
		"invite_token":  inviteToken,
		"invite_url":    inviteURL,
	})
}

func GetEventTeamsHandler(c *fiber.Ctx) error {
	organizerID, err := getUserIDFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	eventID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid event id"})
	}

	eventsColl := db.MongoClient.Database("raidx").Collection("events")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var event models.Event
	if err := eventsColl.FindOne(ctx, bson.M{"_id": eventID, "organizer_id": organizerID}).Decode(&event); err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Event not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch event"})
	}

	return c.JSON(event.ParticipatingTeams)
}
