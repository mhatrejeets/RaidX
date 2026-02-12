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
	"go.mongodb.org/mongo-driver/mongo/options"
)

func CreateEventHandler(c *fiber.Ctx) error {
	organizerID, err := getUserIDFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	var req struct {
		EventName string `json:"event_name" form:"event_name"`
		EventType string `json:"event_type" form:"event_type"`
		MaxTeams  int    `json:"max_teams" form:"max_teams"`
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

	maxTeams := req.MaxTeams
	if eventType == models.EventTypeTournament || eventType == models.EventTypeChampionship {
		if maxTeams < 4 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "max_teams must be at least 4 for tournament or championship"})
		}
	} else {
		maxTeams = 0
	}

	eventsColl := db.MongoClient.Database("raidx").Collection("events")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	event := models.Event{
		OrganizerID:        organizerID,
		EventName:          eventName,
		EventType:          eventType,
		MaxTeams:           maxTeams,
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
		"max_teams":  maxTeams,
	})
}

func MarkEventCompletedHandler(c *fiber.Ctx) error {
	organizerID, err := getUserIDFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	eventID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid event id"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventsColl := db.MongoClient.Database("raidx").Collection("events")
	res, err := eventsColl.UpdateOne(ctx, bson.M{"_id": eventID, "organizer_id": organizerID}, bson.M{
		"$set": bson.M{
			"status":     models.EventStatusCompleted,
			"updated_at": time.Now(),
		},
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update event"})
	}
	if res.MatchedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Event not found"})
	}

	return c.JSON(fiber.Map{"success": true})
}

func UpdateEventHandler(c *fiber.Ctx) error {
	organizerID, err := getUserIDFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	eventID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid event id"})
	}

	var req struct {
		EventName string `json:"event_name" form:"event_name"`
		EventType string `json:"event_type" form:"event_type"`
		MaxTeams  int    `json:"max_teams" form:"max_teams"`
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

	maxTeams := req.MaxTeams
	if eventType == models.EventTypeTournament || eventType == models.EventTypeChampionship {
		if maxTeams < 4 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "max_teams must be at least 4 for tournament or championship"})
		}
	} else {
		maxTeams = 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventsColl := db.MongoClient.Database("raidx").Collection("events")
	var existing models.Event
	if err := eventsColl.FindOne(ctx, bson.M{"_id": eventID, "organizer_id": organizerID}).Decode(&existing); err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Event not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch event"})
	}
	if existing.Status == models.EventStatusActive || existing.Status == models.EventStatusCompleted {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot edit an active or completed event"})
	}

	res, err := eventsColl.UpdateOne(ctx, bson.M{"_id": eventID, "organizer_id": organizerID}, bson.M{
		"$set": bson.M{
			"event_name": eventName,
			"event_type": eventType,
			"max_teams":  maxTeams,
			"updated_at": time.Now(),
		},
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update event"})
	}
	if res.MatchedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Event not found"})
	}

	return c.JSON(fiber.Map{"success": true})
}

func GetOrganizerEventsHandler(c *fiber.Ctx) error {
	organizerID, err := getUserIDFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventsColl := db.MongoClient.Database("raidx").Collection("events")
	findOpts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := eventsColl.Find(ctx, bson.M{"organizer_id": organizerID}, findOpts)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch events"})
	}
	defer cursor.Close(ctx)

	var events []models.Event
	if err := cursor.All(ctx, &events); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode events"})
	}

	// Return empty array instead of nil
	if events == nil {
		events = []models.Event{}
	}

	response := make([]fiber.Map, 0, len(events))
	for _, event := range events {
		accepted := 0
		pending := 0
		declined := 0
		for _, entry := range event.ParticipatingTeams {
			switch entry.Status {
			case models.EventTeamStatusAccepted:
				accepted++
			case models.EventTeamStatusDeclined:
				declined++
			default:
				pending++
			}
		}
		response = append(response, fiber.Map{
			"id":        event.ID.Hex(),
			"eventName": event.EventName,
			"eventType": event.EventType,
			"maxTeams":  event.MaxTeams,
			"status":    event.Status,
			"createdAt": event.CreatedAt,
			"updatedAt": event.UpdatedAt,
			"counts": fiber.Map{
				"accepted": accepted,
				"pending":  pending,
				"declined": declined,
			},
		})
	}

	return c.JSON(response)
}

// GetOrganizerEventDetailHandler returns details for a single event including invite status breakdown.
func GetOrganizerEventDetailHandler(c *fiber.Ctx) error {
	organizerID, err := getUserIDFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	eventID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid event id"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventsColl := db.MongoClient.Database("raidx").Collection("events")
	var event models.Event
	if err := eventsColl.FindOne(ctx, bson.M{"_id": eventID, "organizer_id": organizerID}).Decode(&event); err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Event not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch event"})
	}

	requiredTeams := event.MaxTeams
	if event.EventType == models.EventTypeMatch {
		requiredTeams = 2
	}

	invitesColl := db.MongoClient.Database("raidx").Collection("invitations")
	invitesCursor, err := invitesColl.Find(ctx, bson.M{
		"type":     models.InviteTypeEvent,
		"event_id": eventID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch invitations"})
	}
	defer invitesCursor.Close(ctx)

	var invites []models.Invitation
	if err := invitesCursor.All(ctx, &invites); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode invitations"})
	}

	playersColl := db.MongoClient.Database("raidx").Collection("players")
	teamsColl := db.MongoClient.Database("raidx").Collection("rbac_teams")

	accepted := 0
	pending := 0
	declined := 0
	acceptedList := []fiber.Map{}
	pendingList := []fiber.Map{}
	declinedList := []fiber.Map{}

	for _, invite := range invites {
		status := invite.Status
		item := fiber.Map{
			"teamId":      "",
			"teamName":    "",
			"ownerName":   "",
			"ownerUserId": "",
			"status":      status,
		}

		if invite.TeamID != nil {
			item["teamId"] = invite.TeamID.Hex()
			var team struct {
				TeamName string `bson:"team_name"`
			}
			if err := teamsColl.FindOne(ctx, bson.M{"_id": *invite.TeamID}).Decode(&team); err == nil {
				item["teamName"] = team.TeamName
			}
		}

		if invite.ToID != primitive.NilObjectID {
			var owner struct {
				FullName string `bson:"fullName"`
				UserID   string `bson:"userId"`
				Email    string `bson:"email"`
			}
			if err := playersColl.FindOne(ctx, bson.M{"_id": invite.ToID}).Decode(&owner); err == nil {
				if owner.FullName != "" {
					item["ownerName"] = owner.FullName
				} else {
					item["ownerName"] = owner.Email
				}
				item["ownerUserId"] = owner.UserID
			}
		}

		// Include decline reason if present
		if invite.DeclineReason != "" {
			item["declineReason"] = invite.DeclineReason
		}

		switch status {
		case models.InviteStatusAccepted:
			accepted++
			acceptedList = append(acceptedList, item)
		case models.InviteStatusDeclined:
			declined++
			declinedList = append(declinedList, item)
		default:
			pending++
			pendingList = append(pendingList, item)
		}
	}

	return c.JSON(fiber.Map{
		"id":            event.ID.Hex(),
		"eventName":     event.EventName,
		"eventType":     event.EventType,
		"maxTeams":      event.MaxTeams,
		"status":        event.Status,
		"createdAt":     event.CreatedAt,
		"updatedAt":     event.UpdatedAt,
		"requiredTeams": requiredTeams,
		"counts": fiber.Map{
			"invited":  len(invites),
			"accepted": accepted,
			"pending":  pending,
			"declined": declined,
		},
		"teams": fiber.Map{
			"accepted": acceptedList,
			"pending":  pendingList,
			"declined": declinedList,
		},
	})
}

// GetOrganizerEventMatchStatsHandler returns latest match stats for an event (match-type only).
func GetOrganizerEventMatchStatsHandler(c *fiber.Ctx) error {
	organizerID, err := getUserIDFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	eventID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid event id"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify organizer owns event
	eventsColl := db.MongoClient.Database("raidx").Collection("events")
	var event models.Event
	if err := eventsColl.FindOne(ctx, bson.M{"_id": eventID, "organizer_id": organizerID}).Decode(&event); err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Event not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch event"})
	}

	// Find latest match stats for this event
	matchesColl := db.MongoClient.Database("raidx").Collection("matches")
	var matchDoc bson.M
	findErr := matchesColl.FindOne(ctx, bson.M{"eventId": eventID.Hex()}).Decode(&matchDoc)
	if findErr == mongo.ErrNoDocuments {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Match stats not found"})
	}
	if findErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch match stats"})
	}

	return c.JSON(matchDoc)
}

// StartOrganizerEventHandler marks an event active when it has the required number of accepted teams.
func StartOrganizerEventHandler(c *fiber.Ctx) error {
	organizerID, err := getUserIDFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	eventID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid event id"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventsColl := db.MongoClient.Database("raidx").Collection("events")
	var event models.Event
	if err := eventsColl.FindOne(ctx, bson.M{"_id": eventID, "organizer_id": organizerID}).Decode(&event); err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Event not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch event"})
	}

	requiredTeams := event.MaxTeams
	if event.EventType == models.EventTypeMatch {
		requiredTeams = 2
	}
	if requiredTeams <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Event does not have a valid team limit"})
	}

	invitesColl := db.MongoClient.Database("raidx").Collection("invitations")
	acceptedCount, err := invitesColl.CountDocuments(ctx, bson.M{
		"type":     models.InviteTypeEvent,
		"event_id": eventID,
		"status":   models.InviteStatusAccepted,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to validate event teams"})
	}
	if int(acceptedCount) != requiredTeams {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":         "Accepted teams must equal the required team count",
			"requiredTeams": requiredTeams,
			"acceptedTeams": acceptedCount,
		})
	}

	res, err := eventsColl.UpdateOne(ctx, bson.M{"_id": eventID, "organizer_id": organizerID}, bson.M{
		"$set": bson.M{
			"status":     models.EventStatusActive,
			"updated_at": time.Now(),
		},
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to start event"})
	}
	if res.MatchedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Event not found"})
	}

	return c.JSON(fiber.Map{"success": true})
}

func GetOrganizerEventInvitesHandler(c *fiber.Ctx) error {
	organizerID, err := getUserIDFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	statusFilter := strings.ToLower(strings.TrimSpace(c.Query("status")))
	filter := bson.M{
		"type":    models.InviteTypeEvent,
		"from_id": organizerID,
	}
	if statusFilter != "" {
		filter["status"] = statusFilter
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	invitesColl := db.MongoClient.Database("raidx").Collection("invitations")
	cursor, err := invitesColl.Find(ctx, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch invitations"})
	}
	defer cursor.Close(ctx)

	var invites []models.Invitation
	if err := cursor.All(ctx, &invites); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode invitations"})
	}

	eventsColl := db.MongoClient.Database("raidx").Collection("events")
	playersColl := db.MongoClient.Database("raidx").Collection("players")
	teamsColl := db.MongoClient.Database("raidx").Collection("rbac_teams")
	response := make([]fiber.Map, 0, len(invites))
	for _, invite := range invites {
		item := fiber.Map{
			"id":          invite.ID.Hex(),
			"status":      invite.Status,
			"declineReason": invite.DeclineReason,
			"eventId":     "",
			"eventName":   "",
			"ownerName":   "Unknown",
			"ownerUserId": "",
			"teamId":      "Unassigned",
			"teamName":    "",
			"createdAt":   invite.CreatedAt,
		}
		if invite.EventID != nil {
			item["eventId"] = invite.EventID.Hex()
			var event models.Event
			if err := eventsColl.FindOne(ctx, bson.M{"_id": *invite.EventID}).Decode(&event); err == nil {
				item["eventName"] = event.EventName
			}
		}
		// For event invites, show the team owner being invited
		if invite.ToID != primitive.NilObjectID {
			var owner struct {
				FullName string `bson:"fullName"`
				UserID   string `bson:"userId"`
				Email    string `bson:"email"`
			}
			if err := playersColl.FindOne(ctx, bson.M{"_id": invite.ToID}).Decode(&owner); err == nil {
				item["ownerName"] = owner.FullName
				if owner.FullName == "" {
					item["ownerName"] = owner.Email
				}
				item["ownerUserId"] = owner.UserID
			}
		}
		// Show team details if assigned (when owner accepts and selects team)
		if invite.TeamID != nil {
			item["teamId"] = invite.TeamID.Hex()
			var team struct {
				TeamName string `bson:"team_name"`
			}
			if err := teamsColl.FindOne(ctx, bson.M{"_id": *invite.TeamID}).Decode(&team); err == nil {
				item["teamName"] = team.TeamName
			}
		}
		response = append(response, item)
	}

	return c.JSON(response)
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
		TeamID          string `json:"team_id" form:"team_id"`
		OwnerIdentifier string `json:"ownerIdentifier" form:"ownerIdentifier"`
		GenerateLink    bool   `json:"generate_link" form:"generate_link"`
		ExpiresInDays   int    `json:"expires_in_days" form:"expires_in_days"`
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
		teamIDInput := strings.TrimSpace(req.TeamID)
		ownerIdentifier := strings.TrimSpace(req.OwnerIdentifier)
		if teamIDInput == "" && ownerIdentifier == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ownerIdentifier or team_id is required unless generate_link is true"})
		}

		if teamIDInput != "" {
			var err error
			teamOID, err = primitive.ObjectIDFromHex(teamIDInput)
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
		} else {
			playersColl := db.MongoClient.Database("raidx").Collection("players")
			var owner struct {
				ID       primitive.ObjectID `bson:"_id"`
				UserID   string             `bson:"userId"`
				FullName string             `bson:"fullName"`
				Email    string             `bson:"email"`
				Role     string             `bson:"role"`
			}
			if err := playersColl.FindOne(ctx, bson.M{
				"$or":  []bson.M{{"userId": ownerIdentifier}, {"email": ownerIdentifier}},
				"role": models.RoleTeamOwner,
			}).Decode(&owner); err != nil {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Team owner not found"})
			}

			teamOwnerID = owner.ID
		}
	}

	expiresIn := 30
	if req.ExpiresInDays > 0 {
		expiresIn = req.ExpiresInDays
	}

	inviteToken := generateRandomToken()
	invitesColl := db.MongoClient.Database("raidx").Collection("invitations")
	// Prevent duplicate invitations for same event and owner (allow if previously declined)
	if teamOwnerID != primitive.NilObjectID {
		dupErr := invitesColl.FindOne(ctx, bson.M{
			"type":     models.InviteTypeEvent,
			"event_id": eventID,
			"to_id":    teamOwnerID,
			"status":   bson.M{"$ne": models.InviteStatusDeclined},
		}).Err()
		if dupErr == nil {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Invitation already exists for this event and team owner"})
		}
		if dupErr != mongo.ErrNoDocuments {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to check existing invitations"})
		}
	}

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
