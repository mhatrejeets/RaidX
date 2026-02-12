package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func getUserIDFromLocals(c *fiber.Ctx) (primitive.ObjectID, error) {
	userIDVal := c.Locals("user_id")
	userIDStr, ok := userIDVal.(string)
	if !ok || strings.TrimSpace(userIDStr) == "" {
		return primitive.NilObjectID, errors.New("missing user_id")
	}
	return primitive.ObjectIDFromHex(userIDStr)
}

func CreateTeamHandler(c *fiber.Ctx) error {
	var req struct {
		TeamName    string `json:"team_name" form:"team_name"`
		Description string `json:"description" form:"description"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if strings.TrimSpace(req.TeamName) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "team_name is required"})
	}

	ownerID, err := getUserIDFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	teamColl := db.MongoClient.Database("raidx").Collection("rbac_teams")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	team := models.TeamProfile{
		TeamName:    strings.TrimSpace(req.TeamName),
		OwnerID:     ownerID,
		Description: strings.TrimSpace(req.Description),
		Players:     []primitive.ObjectID{},
		Status:      models.TeamStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	result, err := teamColl.InsertOne(ctx, team)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create team"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"team_id":   result.InsertedID,
		"team_name": team.TeamName,
	})
}

func CreateTeamInviteHandler(c *fiber.Ctx) error {
	ownerID, err := getUserIDFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	teamID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid team id"})
	}

	teamColl := db.MongoClient.Database("raidx").Collection("rbac_teams")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ensure team belongs to this owner
	var team models.TeamProfile
	if err := teamColl.FindOne(ctx, bson.M{"_id": teamID, "owner_id": ownerID}).Decode(&team); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Team not found or not owned by user"})
	}

	var req struct {
		PlayerID      string `json:"player_id" form:"player_id"`
		Username      string `json:"username" form:"username"`
		GenerateLink  bool   `json:"generate_link" form:"generate_link"`
		ExpiresInDays int    `json:"expires_in_days" form:"expires_in_days"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	toID := primitive.NilObjectID
	if !req.GenerateLink {
		playerID := strings.TrimSpace(req.PlayerID)
		username := strings.TrimSpace(req.Username)
		if playerID == "" && username == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "player_id or username is required unless generate_link is true"})
		}

		playersColl := db.MongoClient.Database("raidx").Collection("players")
		var player struct {
			ID     primitive.ObjectID `bson:"_id"`
			UserID string             `bson:"userId"`
			Name   string             `bson:"fullName"`
		}

		// Try by ObjectID
		if playerID != "" {
			if oid, err := primitive.ObjectIDFromHex(playerID); err == nil {
				if err := playersColl.FindOne(ctx, bson.M{"_id": oid}).Decode(&player); err == nil {
					toID = player.ID
				}
			}
		}

		// Try by userId, email, or fullName
		if toID == primitive.NilObjectID && username != "" {
			_ = playersColl.FindOne(ctx, bson.M{"$or": []bson.M{{"userId": username}, {"email": username}, {"fullName": username}}}).Decode(&player)
			if player.ID != primitive.NilObjectID {
				toID = player.ID
			}
		}

		if toID == primitive.NilObjectID {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Player not found"})
		}
	}

	expiresIn := 30
	if req.ExpiresInDays > 0 {
		expiresIn = req.ExpiresInDays
	}

	inviteToken := generateRandomToken()
	invitesColl := db.MongoClient.Database("raidx").Collection("invitations")
	// Prevent duplicate invitations for same team and player (allow if previously declined)
	if toID != primitive.NilObjectID {
		dupErr := invitesColl.FindOne(ctx, bson.M{
			"type":    models.InviteTypeTeam,
			"team_id": teamID,
			"to_id":   toID,
			"status":  bson.M{"$ne": models.InviteStatusDeclined},
		}).Err()
		if dupErr == nil {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Invitation already exists for this team and player"})
		}
		if dupErr != mongo.ErrNoDocuments {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to check existing invitations"})
		}
	}

	invitation := models.Invitation{
		Type:        models.InviteTypeTeam,
		FromID:      ownerID,
		ToID:        toID,
		TeamID:      &teamID,
		InviteToken: inviteToken,
		Status:      models.InviteStatusPending,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(time.Duration(expiresIn) * 24 * time.Hour),
	}

	result, err := invitesColl.InsertOne(ctx, invitation)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create invitation"})
	}

	inviteURL := c.BaseURL() + "/invite/team/" + inviteToken
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"invitation_id": result.InsertedID,
		"invite_token":  inviteToken,
		"invite_url":    inviteURL,
	})
}

func GetTeamInvitesHandler(c *fiber.Ctx) error {
	ownerID, err := getUserIDFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	teamID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid team id"})
	}

	teamColl := db.MongoClient.Database("raidx").Collection("rbac_teams")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := teamColl.FindOne(ctx, bson.M{"_id": teamID, "owner_id": ownerID}).Err(); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Team not found or not owned by user"})
	}

	invitesColl := db.MongoClient.Database("raidx").Collection("invitations")
	cursor, err := invitesColl.Find(ctx, bson.M{
		"team_id": teamID,
		"type":    models.InviteTypeTeam,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch invites"})
	}

	var invites []models.Invitation
	if err := cursor.All(ctx, &invites); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode invites"})
	}

	return c.JSON(invites)
}

func GetOwnerEventInvitesHandler(c *fiber.Ctx) error {
	ownerUserID, err := getUserIDFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	statusFilter := strings.ToLower(strings.TrimSpace(c.Query("status")))

	filter := bson.M{
		"type":  models.InviteTypeEvent,
		"to_id": ownerUserID,
	}
	if statusFilter != "" {
		filter["status"] = statusFilter
	}

	invitesColl := db.MongoClient.Database("raidx").Collection("invitations")
	playersColl := db.MongoClient.Database("raidx").Collection("players")
	eventsColl := db.MongoClient.Database("raidx").Collection("events")
	teamsColl := db.MongoClient.Database("raidx").Collection("rbac_teams")

	// Debug: Check all event invitations in database
	allCursor, _ := invitesColl.Find(ctx, bson.M{"type": models.InviteTypeEvent})
	var allInvites []models.Invitation
	if allCursor != nil {
		allCursor.All(ctx, &allInvites)

	}

	cursor, err := invitesColl.Find(ctx, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch event invites"})
	}

	var invites []models.Invitation
	if err := cursor.All(ctx, &invites); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode event invites"})
	}

	// Ensure we return an empty array instead of null
	if invites == nil {
		invites = []models.Invitation{}
	}

	var ownerName string
	var owner struct {
		FullName string `bson:"fullName"`
	}
	if err := playersColl.FindOne(ctx, bson.M{"_id": ownerUserID}).Decode(&owner); err == nil {
		ownerName = owner.FullName
	}

	response := make([]fiber.Map, 0, len(invites))
	for _, invite := range invites {
		item := fiber.Map{
			"id":     invite.ID.Hex(),
			"status": invite.Status,
		}
		if invite.DeclineReason != "" {
			item["declineReason"] = invite.DeclineReason
		}
		if ownerName != "" {
			item["ownerName"] = ownerName
		}
		if invite.EventID != nil {
			item["eventId"] = invite.EventID.Hex()
			var event struct {
				EventName string `bson:"event_name"`
				EventType string `bson:"event_type"`
			}
			if err := eventsColl.FindOne(ctx, bson.M{"_id": *invite.EventID}).Decode(&event); err == nil {
				item["eventName"] = event.EventName
				item["eventType"] = event.EventType
			}
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
		response = append(response, item)
	}

	return c.JSON(response)
}

func UpdateInvitationStatusHandler(c *fiber.Ctx) error {
	userID, err := getUserIDFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	roleVal := c.Locals("role")
	roleStr, _ := roleVal.(string)
	roleStr = strings.ToLower(strings.TrimSpace(roleStr))

	invID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid invitation id"})
	}

	var req struct {
		Status string `json:"status" form:"status"`
		TeamID string `json:"team_id" form:"team_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status != models.InviteStatusAccepted && status != models.InviteStatusDeclined {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "status must be accepted or declined"})
	}

	invitesColl := db.MongoClient.Database("raidx").Collection("invitations")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var invite models.Invitation
	if err := invitesColl.FindOne(ctx, bson.M{"_id": invID}).Decode(&invite); err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Invitation not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch invitation"})
	}

	if invite.Type == models.InviteTypeTeam {
		if roleStr != models.RolePlayer {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only players can accept team invites"})
		}
		if invite.ToID != primitive.NilObjectID && invite.ToID != userID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Invitation not assigned to this user"})
		}
	} else if invite.Type == models.InviteTypeEvent {
		if roleStr != models.RoleTeamOwner {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only team owners can accept event invites"})
		}

		if invite.ToID != primitive.NilObjectID && invite.ToID != userID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Invitation not assigned to this user"})
		}
	}

	update := bson.M{"$set": bson.M{"status": status}}
	if invite.ToID == primitive.NilObjectID {
		update["$set"].(bson.M)["to_id"] = userID
	}
	if status == models.InviteStatusAccepted {
		update["$set"].(bson.M)["decline_reason"] = ""
	}
	if invite.Type == models.InviteTypeEvent && strings.TrimSpace(req.TeamID) != "" {
		if oid, err := primitive.ObjectIDFromHex(req.TeamID); err == nil {
			update["$set"].(bson.M)["team_id"] = oid
		}
	}

	if invite.Type == models.InviteTypeEvent && status == models.InviteStatusAccepted {
		if strings.TrimSpace(req.TeamID) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "team_id is required for event acceptance"})
		}
		teamOID, err := primitive.ObjectIDFromHex(req.TeamID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid team_id"})
		}
		teamsColl := db.MongoClient.Database("raidx").Collection("rbac_teams")
		var team struct {
			OwnerID primitive.ObjectID `bson:"owner_id"`
			Players []primitive.ObjectID `bson:"players"`
		}
		if err := teamsColl.FindOne(ctx, bson.M{"_id": teamOID}).Decode(&team); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Team not found"})
		}
		if team.OwnerID != userID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Team not owned by user"})
		}
		if len(team.Players) < 7 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Team must have at least 7 players"})
		}

		if invite.EventID != nil {
			eventsColl := db.MongoClient.Database("raidx").Collection("events")
			var event models.Event
			if err := eventsColl.FindOne(ctx, bson.M{"_id": *invite.EventID}).Decode(&event); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch event"})
			}
			maxAllowed := 0
			switch event.EventType {
			case models.EventTypeMatch:
				maxAllowed = 2
			case models.EventTypeTournament, models.EventTypeChampionship:
				maxAllowed = event.MaxTeams
			}
			if maxAllowed > 0 {
				acceptedCount := 0
				for _, entry := range event.ParticipatingTeams {
					if entry.Status == models.EventTeamStatusAccepted {
						acceptedCount++
					}
				}
				if acceptedCount >= maxAllowed {
					label := event.EventType
					if label == "" {
						label = "event"
					}
					reason := fmt.Sprintf("Maximum number of teams for the %s reached", label)
					declineUpdate := bson.M{"$set": bson.M{
						"status":          models.InviteStatusDeclined,
						"decline_reason":  reason,
						"to_id":           userID,
						"team_id":         teamOID,
					}}
					_, _ = invitesColl.UpdateOne(ctx, bson.M{"_id": invID}, declineUpdate)
					return c.JSON(fiber.Map{
						"status":  models.InviteStatusDeclined,
						"reason":  reason,
						"message": reason,
					})
				}
			}
		}
	}

	if _, err := invitesColl.UpdateOne(ctx, bson.M{"_id": invID}, update); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update invitation"})
	}

	if status == models.InviteStatusAccepted && invite.Type == models.InviteTypeTeam && invite.TeamID != nil {
		if invite.Source == "invite_link" {
			if err := createPendingApprovalFromInvite(ctx, invite, userID); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create pending approval"})
			}
			_, _ = invitesColl.UpdateOne(ctx, bson.M{"_id": invite.ID}, bson.M{
				"$set": bson.M{"status": models.InviteStatusInvitedViaLink},
			})
			return c.JSON(fiber.Map{"status": models.InviteStatusInvitedViaLink})
		}
		teamsColl := db.MongoClient.Database("raidx").Collection("rbac_teams")
		_, _ = teamsColl.UpdateOne(ctx, bson.M{"_id": *invite.TeamID}, bson.M{
			"$addToSet": bson.M{"players": userID},
			"$set":      bson.M{"updated_at": time.Now()},
		})
	}

	if status == models.InviteStatusAccepted && invite.Type == models.InviteTypeEvent && invite.EventID != nil {
		if invite.Source == "invite_link" {
			if err := createPendingApprovalFromInvite(ctx, invite, userID); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create pending approval"})
			}
			_, _ = invitesColl.UpdateOne(ctx, bson.M{"_id": invite.ID}, bson.M{
				"$set": bson.M{"status": models.InviteStatusInvitedViaLink},
			})
			return c.JSON(fiber.Map{"status": models.InviteStatusInvitedViaLink})
		}
		teamsColl := db.MongoClient.Database("raidx").Collection("rbac_teams")
		teamID := primitive.NilObjectID
		if invite.TeamID != nil {
			teamID = *invite.TeamID
		} else if strings.TrimSpace(req.TeamID) != "" {
			if oid, err := primitive.ObjectIDFromHex(req.TeamID); err == nil {
				teamID = oid
			}
		}
		if teamID == primitive.NilObjectID {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "team_id is required to accept event invite"})
		}

		// Validate team owner
		var team models.TeamProfile
		if err := teamsColl.FindOne(ctx, bson.M{"_id": teamID, "owner_id": userID}).Decode(&team); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Team not found or not owned by user"})
		}

		eventsColl := db.MongoClient.Database("raidx").Collection("events")
		updateRes, _ := eventsColl.UpdateOne(ctx, bson.M{
			"_id":                         *invite.EventID,
			"participating_teams.team_id": teamID,
		}, bson.M{"$set": bson.M{"participating_teams.$.status": models.EventTeamStatusAccepted, "updated_at": time.Now()}})
		if updateRes == nil || updateRes.MatchedCount == 0 {
			_, _ = eventsColl.UpdateOne(ctx, bson.M{"_id": *invite.EventID}, bson.M{
				"$addToSet": bson.M{"participating_teams": models.EventTeamEntry{
					TeamID: teamID,
					Status: models.EventTeamStatusAccepted,
				}},
				"$set": bson.M{"updated_at": time.Now()},
			})
		}
	}

	return c.JSON(fiber.Map{"status": status})
}

func createPendingApprovalFromInvite(ctx context.Context, invite models.Invitation, userID primitive.ObjectID) error {
	linksCollection := db.MongoClient.Database("raidx").Collection("invite_links")
	var link models.InviteLink
	if err := linksCollection.FindOne(ctx, bson.M{
		"token":    invite.InviteToken,
		"isActive": true,
	}).Decode(&link); err != nil {
		return err
	}

	approvalsCollection := db.MongoClient.Database("raidx").Collection("pending_approvals")
	// Avoid duplicates
	existingErr := approvalsCollection.FindOne(ctx, bson.M{
		"inviteLinkId": link.ID,
		"acceptorId":   userID.Hex(),
		"status":       bson.M{"$in": []string{"pending", "invited_via_link"}},
	}).Err()
	if existingErr == nil {
		return nil
	}

	playersCollection := db.MongoClient.Database("raidx").Collection("players")
	var account models.User
	playerName := userID.Hex()
	playerUsername := userID.Hex()
	if err := playersCollection.FindOne(ctx, bson.M{"_id": userID}).Decode(&account); err == nil {
		if account.FullName != "" {
			playerName = account.FullName
		}
		if account.UserID != "" {
			playerUsername = account.UserID
		}
	}

	approval := models.PendingApproval{
		ID:               primitive.NewObjectID(),
		InviteLinkID:     link.ID,
		FromID:           link.FromID,
		AcceptorID:       userID.Hex(),
		AcceptorUsername: playerUsername,
		AcceptorName:     playerName,
		Status:           "invited_via_link",
		CreatedAt:        time.Now(),
	}

	if invite.Type == models.InviteTypeTeam {
		approval.Type = models.InviteLinkTypeTeam
		approval.TeamID = link.TeamID
		approval.AcceptorRole = models.RolePlayer
	}

	if invite.Type == models.InviteTypeEvent {
		approval.Type = models.InviteLinkTypeEvent
		approval.EventID = link.EventID
		approval.AcceptorRole = models.RoleTeamOwner
	}

	if _, err := approvalsCollection.InsertOne(ctx, approval); err != nil {
		return err
	}

	_, _ = linksCollection.UpdateOne(ctx, bson.M{"_id": link.ID}, bson.M{
		"$inc": bson.M{"usedCount": 1},
	})

	return nil
}
