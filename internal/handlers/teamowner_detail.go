package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// GetOwnerTeams returns all teams owned by the authenticated user
func GetOwnerTeams(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	// Convert string userID to ObjectID
	ownerID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	teamsCollection := db.MongoClient.Database("raidx").Collection("rbac_teams")
	cursor, err := teamsCollection.Find(ctx, bson.M{"owner_id": ownerID})
	if err != nil {
		logrus.Error("GetOwnerTeams: Failed to fetch teams:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch teams"})
	}
	defer cursor.Close(ctx)

	// Use raw BSON to avoid decode errors with corrupted player arrays
	var teamsRaw []bson.M
	if err = cursor.All(ctx, &teamsRaw); err != nil {
		logrus.Error("GetOwnerTeams: Failed to decode teams:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode teams"})
	}

	// Convert to response format
	var teams []fiber.Map
	for _, teamRaw := range teamsRaw {
		teamID := teamRaw["_id"].(primitive.ObjectID)

		// Handle both "teamName" and "team_name" field names
		teamName, ok := teamRaw["teamName"].(string)
		if !ok {
			teamName, _ = teamRaw["team_name"].(string)
		}

		status, _ := teamRaw["status"].(string)
		createdAt, _ := teamRaw["createdAt"].(time.Time)
		updatedAt, _ := teamRaw["updatedAt"].(time.Time)

		// Count valid players
		playerCount := 0
		if playersArray, ok := teamRaw["players"].(primitive.A); ok {
			playerCount = len(playersArray)
		}

		teams = append(teams, fiber.Map{
			"ID":        teamID.Hex(),
			"TeamName":  teamName,
			"OwnerID":   teamRaw["owner_id"],
			"Players":   playerCount,
			"Status":    status,
			"CreatedAt": createdAt,
			"UpdatedAt": updatedAt,
		})
	}

	return c.JSON(teams)
}

// GetTeamByIDDetail returns team details by ID
func GetTeamByIDDetail(c *fiber.Ctx) error {
	teamID := c.Params("id")
	userID := c.Locals("user_id").(string)

	// Convert teamID string to ObjectID
	teamOID, err := primitive.ObjectIDFromHex(teamID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid team ID"})
	}

	// Convert userID string to ObjectID
	ownerID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	teamsCollection := db.MongoClient.Database("raidx").Collection("rbac_teams")

	// Use raw BSON to avoid decoding issues with the players array
	var teamRaw bson.M
	err = teamsCollection.FindOne(ctx, bson.M{"_id": teamOID}).Decode(&teamRaw)
	if err == mongo.ErrNoDocuments {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Team not found"})
	}
	if err != nil {
		logrus.Error("GetTeamByIDDetail: Failed to fetch team:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch team"})
	}

	// Extract basic fields
	teamOIDResult := teamRaw["_id"].(primitive.ObjectID)

	// Handle both "teamName" and "team_name" field names
	teamName, ok := teamRaw["teamName"].(string)
	if !ok {
		teamName, _ = teamRaw["team_name"].(string)
	}

	ownerIDResult := teamRaw["owner_id"].(primitive.ObjectID)
	status, _ := teamRaw["status"].(string)
	createdAt, _ := teamRaw["createdAt"].(time.Time)
	updatedAt, _ := teamRaw["updatedAt"].(time.Time)

	// Verify ownership (for edit operations)
	if ownerIDResult != ownerID && c.Get("Authorization") != "" {
		// Allow viewing other teams' details but restrict modifications
		if c.Method() == fiber.MethodPut || c.Method() == fiber.MethodDelete {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unauthorized"})
		}
	}

	// Get owner name
	playersCollection := db.MongoClient.Database("raidx").Collection("players")
	var owner models.User
	playersCollection.FindOne(ctx, bson.M{"_id": ownerIDResult}).Decode(&owner)

	// Extract and fetch player details - handle both string and ObjectID formats
	var players []fiber.Map
	if playersArray, ok := teamRaw["players"].(primitive.A); ok {
		for _, playerItem := range playersArray {
			var playerOID primitive.ObjectID
			var playerID string

			// Try to handle both ObjectID and string formats
			switch v := playerItem.(type) {
			case primitive.ObjectID:
				playerOID = v
				playerID = v.Hex()
			case string:
				// Try to convert string to ObjectID
				var err error
				playerOID, err = primitive.ObjectIDFromHex(v)
				if err != nil {
					logrus.Warn("GetTeamByIDDetail: Invalid player ID format:", v)
					continue
				}
				playerID = v
			default:
				logrus.Warn("GetTeamByIDDetail: Unknown player ID type")
				continue
			}

			var player models.User
			if err := playersCollection.FindOne(ctx, bson.M{"_id": playerOID}).Decode(&player); err != nil {
				logrus.Warn("GetTeamByIDDetail: Could not fetch player details:", playerID)
				continue
			}
			players = append(players, fiber.Map{
				"_id":      playerID,
				"userId":   player.UserID,
				"fullName": player.FullName,
				"email":    player.Email,
				"position": player.Position,
			})
		}
	}

	// Fetch pending/declined invites for this team
	invitesCollection := db.MongoClient.Database("raidx").Collection("invitations")
	invitesCursor, err := invitesCollection.Find(ctx, bson.M{
		"team_id": teamOID,
		"type":    models.InviteTypeTeam,
		"status":  bson.M{"$in": []string{models.InviteStatusPending, models.InviteStatusDeclined}},
	})
	if err != nil {
		logrus.Warn("GetTeamByIDDetail: Failed to fetch invites:", err)
	}
	var inviteItems []fiber.Map
	if invitesCursor != nil {
		defer invitesCursor.Close(ctx)
		var invites []models.Invitation
		if err := invitesCursor.All(ctx, &invites); err != nil {
			logrus.Warn("GetTeamByIDDetail: Failed to decode invites:", err)
		} else {
			for _, invite := range invites {
				item := fiber.Map{
					"id":     invite.ID.Hex(),
					"status": invite.Status,
				}
				if invite.ToID != primitive.NilObjectID {
					var invited models.User
					if err := playersCollection.FindOne(ctx, bson.M{"_id": invite.ToID}).Decode(&invited); err == nil {
						item["playerId"] = invite.ToID.Hex()
						item["userId"] = invited.UserID
						item["fullName"] = invited.FullName
						item["email"] = invited.Email
					}
				}
				inviteItems = append(inviteItems, item)
			}
		}
	}

	// Return enriched response
	return c.JSON(fiber.Map{
		"ID":        teamOIDResult.Hex(),
		"TeamName":  teamName,
		"OwnerID":   ownerIDResult.Hex(),
		"OwnerName": owner.FullName,
		"Players":   players,
		"Invites":   inviteItems,
		"Status":    status,
		"CreatedAt": createdAt,
		"UpdatedAt": updatedAt,
	})
}

// UpdateTeam updates team information
func UpdateTeam(c *fiber.Ctx) error {
	teamID := c.Params("id")
	userID := c.Locals("user_id").(string)

	// Convert string userID to ObjectID
	ownerID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	// Convert teamID string to ObjectID
	teamOID, err := primitive.ObjectIDFromHex(teamID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid team ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify ownership
	teamsCollection := db.MongoClient.Database("raidx").Collection("rbac_teams")
	var team models.TeamProfile
	err = teamsCollection.FindOne(ctx, bson.M{"_id": teamOID}).Decode(&team)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Team not found"})
	}

	if team.OwnerID != ownerID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Parse request body
	var updateData struct {
		TeamName string `json:"teamName"`
		City     string `json:"city"`
	}
	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Update team
	_, err = teamsCollection.UpdateOne(ctx, bson.M{"_id": teamOID}, bson.M{
		"$set": bson.M{
			"team_name": updateData.TeamName,
			"city":      updateData.City,
		},
	})
	if err != nil {
		logrus.Error("UpdateTeam: Failed to update team:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update team"})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Team updated"})
}

// AddPlayerToTeam adds a player to a team (by ID or username)
func AddPlayerToTeam(c *fiber.Ctx) error {
	teamID := c.Params("id")
	userID := c.Locals("user_id").(string)

	// Convert teamID string to ObjectID
	teamOID, err := primitive.ObjectIDFromHex(teamID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid team ID"})
	}

	// Convert userID string to ObjectID
	ownerID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify team ownership
	teamsCollection := db.MongoClient.Database("raidx").Collection("rbac_teams")
	var team models.TeamProfile
	err = teamsCollection.FindOne(ctx, bson.M{"_id": teamOID}).Decode(&team)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Team not found"})
	}

	if team.OwnerID != ownerID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Check if team is in any active events
	hasActiveEvents, eventNames, err := checkTeamActiveEvents(ctx, teamOID)
	if err != nil {
		logrus.Error("AddPlayerToTeam: Failed to check active events:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to verify team status"})
	}
	if hasActiveEvents {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Cannot modify team while participating in active events",
			"events": eventNames,
		})
	}

	// Parse request
	var req struct {
		PlayerIdentifier string `json:"playerIdentifier"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Find player by ID or username
	playersCollection := db.MongoClient.Database("raidx").Collection("players")
	var playerRaw bson.M
	err = playersCollection.FindOne(ctx, bson.M{
		"$or": []bson.M{
			{"userId": req.PlayerIdentifier},
			{"email": req.PlayerIdentifier},
		},
	}).Decode(&playerRaw)
	if err != nil {
		logrus.Warn("AddPlayerToTeam: Player not found:", req.PlayerIdentifier)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Player not found"})
	}

	// Extract player's ObjectID from _id field
	playerOID, ok := playerRaw["_id"].(primitive.ObjectID)
	if !ok {
		logrus.Warn("AddPlayerToTeam: Player has invalid _id format")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid player ID in database"})
	}

	// Add player to team
	_, err = teamsCollection.UpdateOne(ctx, bson.M{"_id": teamOID}, bson.M{
		"$addToSet": bson.M{"players": playerOID},
	})
	if err != nil {
		logrus.Error("AddPlayerToTeam: Failed to add player:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to add player"})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Player added to team"})
}

// RemovePlayerFromTeam removes a player from a team
func RemovePlayerFromTeam(c *fiber.Ctx) error {
	teamID := c.Params("id")
	playerID := c.Params("playerId")
	userID := c.Locals("user_id").(string)

	// Convert teamID string to ObjectID
	teamOID, err := primitive.ObjectIDFromHex(teamID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid team ID"})
	}

	// Convert playerID string to ObjectID
	playerOID, err := primitive.ObjectIDFromHex(playerID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid player ID"})
	}

	// Convert userID string to ObjectID
	ownerID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify team ownership
	teamsCollection := db.MongoClient.Database("raidx").Collection("rbac_teams")
	var team models.TeamProfile
	err = teamsCollection.FindOne(ctx, bson.M{"_id": teamOID}).Decode(&team)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Team not found"})
	}

	if team.OwnerID != ownerID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Check if team is in any active events
	hasActiveEvents, eventNames, err := checkTeamActiveEvents(ctx, teamOID)
	if err != nil {
		logrus.Error("RemovePlayerFromTeam: Failed to check active events:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to verify team status"})
	}
	if hasActiveEvents {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Cannot modify team while participating in active events",
			"events": eventNames,
		})
	}

	// Remove player from team
	_, err = teamsCollection.UpdateOne(ctx, bson.M{"_id": teamOID}, bson.M{
		"$pull": bson.M{"players": playerOID},
	})
	if err != nil {
		logrus.Error("RemovePlayerFromTeam: Failed to remove player:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to remove player"})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Player removed from team"})
}

// DeleteTeam deletes a team
func DeleteTeam(c *fiber.Ctx) error {
	teamID := c.Params("id")
	userID := c.Locals("user_id").(string)

	// Convert teamID string to ObjectID
	teamOID, err := primitive.ObjectIDFromHex(teamID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid team ID"})
	}

	// Convert userID string to ObjectID
	ownerID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify team ownership
	teamsCollection := db.MongoClient.Database("raidx").Collection("rbac_teams")
	var team models.TeamProfile
	err = teamsCollection.FindOne(ctx, bson.M{"_id": teamOID}).Decode(&team)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Team not found"})
	}

	if team.OwnerID != ownerID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Delete team
	_, err = teamsCollection.DeleteOne(ctx, bson.M{"_id": teamOID})
	if err != nil {
		logrus.Error("DeleteTeam: Failed to delete team:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete team"})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Team deleted"})
}

// GetOwnerTournamentRequests returns tournaments and pending requests for team owner
func GetOwnerTournamentRequests(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	// Convert string userID to ObjectID
	ownerID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get all teams owned by user - use raw BSON to avoid decode errors
	teamsCollection := db.MongoClient.Database("raidx").Collection("rbac_teams")
	cursor, err := teamsCollection.Find(ctx, bson.M{"owner_id": ownerID})
	if err != nil {
		logrus.Error("GetOwnerTournamentRequests: Failed to fetch teams:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch teams"})
	}
	defer cursor.Close(ctx)

	var teamsRaw []bson.M
	if err = cursor.All(ctx, &teamsRaw); err != nil {
		logrus.Error("GetOwnerTournamentRequests: Failed to decode teams:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode teams"})
	}

	// Extract team IDs
	teamIDs := make([]string, len(teamsRaw))
	for i, teamRaw := range teamsRaw {
		if teamID, ok := teamRaw["_id"].(primitive.ObjectID); ok {
			teamIDs[i] = teamID.Hex()
		}
	}

	// Get tournaments where user's teams participate
	eventsCollection := db.MongoClient.Database("raidx").Collection("rbac_events")
	cursor, err = eventsCollection.Find(ctx, bson.M{
		"participating_teams": bson.M{"$in": teamIDs},
	})
	if err != nil {
		logrus.Error("GetOwnerTournamentRequests: Failed to fetch tournaments:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch tournaments"})
	}
	defer cursor.Close(ctx)

	var tournaments []models.Event
	if err = cursor.All(ctx, &tournaments); err != nil {
		logrus.Error("GetOwnerTournamentRequests: Failed to decode tournaments:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode tournaments"})
	}

	// Get pending approvals for this owner's teams
	approvalsCollection := db.MongoClient.Database("raidx").Collection("pending_approvals")
	cursor, err = approvalsCollection.Find(ctx, bson.M{
		"type":   models.InviteLinkTypeEvent,
		"fromId": bson.M{"$ne": userID}, // Requests from organizers, not this owner
		"status": "pending",
	})
	if err != nil {
		logrus.Error("GetOwnerTournamentRequests: Failed to fetch pending requests:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch requests"})
	}
	defer cursor.Close(ctx)

	var pendingRequests []models.PendingApproval
	if err = cursor.All(ctx, &pendingRequests); err != nil {
		logrus.Error("GetOwnerTournamentRequests: Failed to decode requests:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode requests"})
	}

	// Get event names and available teams for each pending request
	var enrichedRequests []fiber.Map
	for _, req := range pendingRequests {
		var event models.Event
		eventsCollection.FindOne(ctx, bson.M{"_id": req.EventID}).Decode(&event)

		// Get available teams (owned by this user)
		enrichedRequests = append(enrichedRequests, fiber.Map{
			"_id":            req.ID.Hex(),
			"eventName":      event.EventName,
			"createdAt":      req.CreatedAt,
			"availableTeams": teamsRaw,
		})
	}

	return c.JSON(fiber.Map{
		"tournaments":     tournaments,
		"pendingRequests": enrichedRequests,
	})
}

// checkTeamActiveEvents checks if a team is participating in any active or ongoing events
func checkTeamActiveEvents(ctx context.Context, teamID primitive.ObjectID) (bool, []string, error) {
	invitationsCollection := db.MongoClient.Database("raidx").Collection("invitations")

	// Find all accepted invitations for this team
	cursor, err := invitationsCollection.Find(ctx, bson.M{
		"team_id": teamID,
		"status":  models.InviteStatusAccepted,
		"type":    models.InviteTypeEvent,
	})
	if err != nil {
		return false, nil, err
	}
	defer cursor.Close(ctx)

	var invitations []models.Invitation
	if err = cursor.All(ctx, &invitations); err != nil {
		return false, nil, err
	}

	if len(invitations) == 0 {
		return false, nil, nil
	}

	// Check if any of the events are active or ongoing
	eventsCollection := db.MongoClient.Database("raidx").Collection("events")
	eventNames := []string{}

	for _, inv := range invitations {
		if inv.EventID == nil {
			continue
		}

		var event models.Event
		err := eventsCollection.FindOne(ctx, bson.M{
			"_id":    *inv.EventID,
			"status": bson.M{"$in": []string{"active", "ongoing"}},
		}).Decode(&event)

		if err == nil {
			// Event found with active/ongoing status
			eventNames = append(eventNames, fmt.Sprintf("%s (%s)", event.EventName, event.Status))
		}
	}

	if len(eventNames) > 0 {
		return true, eventNames, nil
	}

	return false, nil, nil
}
