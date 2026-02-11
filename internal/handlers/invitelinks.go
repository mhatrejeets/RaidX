package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// generateToken creates a secure random token
func generateToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GenerateTeamInviteLink creates a shareable invite link for a team
func GenerateTeamInviteLink(c *fiber.Ctx) error {
	teamID := c.Params("id")
	userID := c.Locals("user_id").(string)

	// Convert IDs to ObjectID
	teamOID, err := primitive.ObjectIDFromHex(teamID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid team ID"})
	}

	ownerID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify team exists and user is owner
	teamsCollection := db.MongoClient.Database("raidx").Collection("rbac_teams")
	var team models.TeamProfile
	err = teamsCollection.FindOne(ctx, bson.M{"_id": teamOID, "owner_id": ownerID}).Decode(&team)
	if err != nil {
		logrus.Warn("GenerateTeamInviteLink: Team not found or user not owner:", err)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Team not found or unauthorized"})
	}

	// Generate token
	token, err := generateToken()
	if err != nil {
		logrus.Error("GenerateTeamInviteLink: Failed to generate token:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate link"})
	}

	// Create invite link document
	inviteLink := models.InviteLink{
		ID:        primitive.NewObjectID(),
		Token:     token,
		Type:      models.InviteLinkTypeTeam,
		FromID:    ownerID.Hex(),
		TeamID:    teamOID.Hex(),
		TeamName:  team.TeamName,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().AddDate(0, 0, 30), // 30-day expiry
		MaxUses:   0,                            // Unlimited
		UsedCount: 0,
		IsActive:  true,
	}

	linksCollection := db.MongoClient.Database("raidx").Collection("invite_links")
	_, err = linksCollection.InsertOne(ctx, inviteLink)
	if err != nil {
		logrus.Error("GenerateTeamInviteLink: Failed to create invite link:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create link"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"link":    "/invite/team/" + token,
		"token":   token,
	})
}

// GenerateEventInviteLink creates a shareable invite link for an event
func GenerateEventInviteLink(c *fiber.Ctx) error {
	eventID := c.Params("id")
	userID := c.Locals("user_id").(string)

	// Convert IDs to ObjectID
	eventOID, err := primitive.ObjectIDFromHex(eventID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid event ID"})
	}

	organizerID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify event exists and user is organizer
	eventsCollection := db.MongoClient.Database("raidx").Collection("rbac_events")
	var event models.Event
	err = eventsCollection.FindOne(ctx, bson.M{"_id": eventOID, "organizer_id": organizerID}).Decode(&event)
	if err != nil {
		logrus.Warn("GenerateEventInviteLink: Event not found or user not organizer:", err)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Event not found or unauthorized"})
	}

	// Generate token
	token, err := generateToken()
	if err != nil {
		logrus.Error("GenerateEventInviteLink: Failed to generate token:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate link"})
	}

	// Create invite link document
	inviteLink := models.InviteLink{
		ID:        primitive.NewObjectID(),
		Token:     token,
		Type:      models.InviteLinkTypeEvent,
		FromID:    organizerID.Hex(),
		EventID:   eventOID.Hex(),
		EventName: event.EventName,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().AddDate(0, 0, 30), // 30-day expiry
		MaxUses:   0,                            // Unlimited
		UsedCount: 0,
		IsActive:  true,
	}

	linksCollection := db.MongoClient.Database("raidx").Collection("invite_links")
	_, err = linksCollection.InsertOne(ctx, inviteLink)
	if err != nil {
		logrus.Error("GenerateEventInviteLink: Failed to create invite link:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create link"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"link":    "/invite/event/" + token,
		"token":   token,
	})
}

// GetTeamInviteLinkDetails returns basic details for a team invite link
func GetTeamInviteLinkDetails(c *fiber.Ctx) error {
	token := c.Params("token")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	linksCollection := db.MongoClient.Database("raidx").Collection("invite_links")
	var inviteLink models.InviteLink
	err := linksCollection.FindOne(ctx, bson.M{
		"token":    token,
		"type":     models.InviteLinkTypeTeam,
		"isActive": true,
	}).Decode(&inviteLink)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Invalid or expired invite link"})
	}

	if time.Now().After(inviteLink.ExpiresAt) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invite link has expired"})
	}

	return c.JSON(fiber.Map{
		"teamName":     inviteLink.TeamName,
		"teamId":       inviteLink.TeamID,
		"requiredRole": models.RolePlayer,
		"expiresAt":    inviteLink.ExpiresAt,
	})
}

// ClaimTeamInviteLink assigns a team invite link to the logged-in player so it shows on the dashboard.
func ClaimTeamInviteLink(c *fiber.Ctx) error {
	token := c.Params("token")
	userID := c.Locals("user_id").(string)
	roleVal := c.Locals("role")
	roleStr, _ := roleVal.(string)
	roleStr = strings.ToLower(strings.TrimSpace(roleStr))
	if roleStr != models.RolePlayer {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only player accounts can claim team invites"})
	}

	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	linksCollection := db.MongoClient.Database("raidx").Collection("invite_links")
	var inviteLink models.InviteLink
	err = linksCollection.FindOne(ctx, bson.M{
		"token":    token,
		"type":     models.InviteLinkTypeTeam,
		"isActive": true,
	}).Decode(&inviteLink)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Invalid or expired invite link"})
	}

	if time.Now().After(inviteLink.ExpiresAt) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invite link has expired"})
	}

	invitesCollection := db.MongoClient.Database("raidx").Collection("invitations")

	// If an invitation already exists for this token and user, return success.
	var existing models.Invitation
	err = invitesCollection.FindOne(ctx, bson.M{
		"invite_token": token,
		"type":         models.InviteTypeTeam,
		"to_id":        userOID,
	}).Decode(&existing)
	if err == nil {
		return c.JSON(fiber.Map{
			"success":       true,
			"invitation_id": existing.ID.Hex(),
		})
	}

	// If an invitation exists but is unassigned, assign it to the user.
	var unassigned models.Invitation
	err = invitesCollection.FindOne(ctx, bson.M{
		"invite_token": token,
		"type":         models.InviteTypeTeam,
		"to_id":        primitive.NilObjectID,
	}).Decode(&unassigned)
	if err == nil {
		if _, err := invitesCollection.UpdateOne(ctx, bson.M{"_id": unassigned.ID}, bson.M{
			"$set": bson.M{"to_id": userOID},
		}); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to claim invite"})
		}
		return c.JSON(fiber.Map{
			"success":       true,
			"invitation_id": unassigned.ID.Hex(),
		})
	}

	teamOID, err := primitive.ObjectIDFromHex(inviteLink.TeamID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid team ID"})
	}

	fromOID, err := primitive.ObjectIDFromHex(inviteLink.FromID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid owner ID"})
	}

	invitation := models.Invitation{
		ID:          primitive.NewObjectID(),
		Type:        models.InviteTypeTeam,
		FromID:      fromOID,
		ToID:        userOID,
		TeamID:      &teamOID,
		InviteToken: token,
		Status:      models.InviteStatusPending,
		CreatedAt:   time.Now(),
		ExpiresAt:   inviteLink.ExpiresAt,
	}

	if _, err := invitesCollection.InsertOne(ctx, invitation); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to claim invite"})
	}

	return c.JSON(fiber.Map{
		"success":       true,
		"invitation_id": invitation.ID.Hex(),
	})
}

// GetEventInviteLinkDetails returns basic details for an event invite link
func GetEventInviteLinkDetails(c *fiber.Ctx) error {
	token := c.Params("token")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	linksCollection := db.MongoClient.Database("raidx").Collection("invite_links")
	var inviteLink models.InviteLink
	err := linksCollection.FindOne(ctx, bson.M{
		"token":    token,
		"type":     models.InviteLinkTypeEvent,
		"isActive": true,
	}).Decode(&inviteLink)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Invalid or expired invite link"})
	}

	if time.Now().After(inviteLink.ExpiresAt) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invite link has expired"})
	}

	return c.JSON(fiber.Map{
		"eventName":    inviteLink.EventName,
		"eventId":      inviteLink.EventID,
		"requiredRole": models.RoleTeamOwner,
		"expiresAt":    inviteLink.ExpiresAt,
	})
}

// AcceptTeamInviteLink allows a player to accept a team invite link
func AcceptTeamInviteLink(c *fiber.Ctx) error {
	token := c.Params("token")
	userID := c.Locals("user_id").(string)
	roleVal := c.Locals("role")
	roleStr, _ := roleVal.(string)
	roleStr = strings.ToLower(strings.TrimSpace(roleStr))
	if roleStr != models.RolePlayer {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only player accounts can accept team invites"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Find invite link
	linksCollection := db.MongoClient.Database("raidx").Collection("invite_links")
	var inviteLink models.InviteLink
	err := linksCollection.FindOne(ctx, bson.M{
		"token":    token,
		"type":     models.InviteLinkTypeTeam,
		"isActive": true,
	}).Decode(&inviteLink)
	if err != nil {
		logrus.Warn("AcceptTeamInviteLink: Invalid or expired link:", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Invalid or expired invite link"})
	}

	// Check if link is expired
	if time.Now().After(inviteLink.ExpiresAt) {
		logrus.Warn("AcceptTeamInviteLink: Link expired")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invite link has expired"})
	}

	// Check max uses
	if inviteLink.MaxUses > 0 && inviteLink.UsedCount >= inviteLink.MaxUses {
		logrus.Warn("AcceptTeamInviteLink: Max uses exceeded")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invite link has reached max uses"})
	}

	// Get player info (if exists)
	// Get user's full name from players collection
	playersCollection := db.MongoClient.Database("raidx").Collection("players")
	var account models.User
	var playerName string
	err = playersCollection.FindOne(ctx, bson.M{"userId": userID}).Decode(&account)
	if err == nil {
		playerName = account.FullName
	} else {
		// Fallback to userID if not found
		playerName = userID
	}

	// Create pending approval entry
	approvalsCollection := db.MongoClient.Database("raidx").Collection("pending_approvals")
	approval := models.PendingApproval{
		ID:           primitive.NewObjectID(),
		InviteLinkID: inviteLink.ID,
		Type:         models.InviteLinkTypeTeam,
		FromID:       inviteLink.FromID,
		TeamID:       inviteLink.TeamID,
		AcceptorID:   userID,
		AcceptorName: playerName,
		AcceptorRole: models.RolePlayer,
		Status:       "pending",
		CreatedAt:    time.Now(),
	}

	_, err = approvalsCollection.InsertOne(ctx, approval)
	if err != nil {
		logrus.Error("AcceptTeamInviteLink: Failed to create pending approval:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to accept invite"})
	}

	// Increment used count
	linksCollection.UpdateOne(ctx, bson.M{"_id": inviteLink.ID}, bson.M{
		"$inc": bson.M{"usedCount": 1},
	})

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Invite accepted. Waiting for team owner approval.",
	})
}

// AcceptEventInviteLink allows a team owner to accept an event invite link
func AcceptEventInviteLink(c *fiber.Ctx) error {
	token := c.Params("token")
	userID := c.Locals("user_id").(string)
	roleVal := c.Locals("role")
	roleStr, _ := roleVal.(string)
	roleStr = strings.ToLower(strings.TrimSpace(roleStr))
	if roleStr != models.RoleTeamOwner {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only team owner accounts can accept event invites"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Find invite link
	linksCollection := db.MongoClient.Database("raidx").Collection("invite_links")
	var inviteLink models.InviteLink
	err := linksCollection.FindOne(ctx, bson.M{
		"token":    token,
		"type":     models.InviteLinkTypeEvent,
		"isActive": true,
	}).Decode(&inviteLink)
	if err != nil {
		logrus.Warn("AcceptEventInviteLink: Invalid or expired link:", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Invalid or expired invite link"})
	}

	// Check if link is expired
	if time.Now().After(inviteLink.ExpiresAt) {
		logrus.Warn("AcceptEventInviteLink: Link expired")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invite link has expired"})
	}

	// Get team owner info
	playersCollection := db.MongoClient.Database("raidx").Collection("players")
	var teamOwner models.User
	err = playersCollection.FindOne(ctx, bson.M{"userId": userID}).Decode(&teamOwner)
	if err != nil {
		logrus.Warn("AcceptEventInviteLink: Team owner not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	// Create pending approval entry
	approvalsCollection := db.MongoClient.Database("raidx").Collection("pending_approvals")
	approval := models.PendingApproval{
		ID:           primitive.NewObjectID(),
		InviteLinkID: inviteLink.ID,
		Type:         models.InviteLinkTypeEvent,
		FromID:       inviteLink.FromID,
		EventID:      inviteLink.EventID,
		AcceptorID:   userID,
		AcceptorName: teamOwner.FullName,
		AcceptorRole: models.RoleTeamOwner,
		Status:       "pending",
		CreatedAt:    time.Now(),
	}

	_, err = approvalsCollection.InsertOne(ctx, approval)
	if err != nil {
		logrus.Error("AcceptEventInviteLink: Failed to create pending approval:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to accept invite"})
	}

	// Increment used count
	linksCollection.UpdateOne(ctx, bson.M{"_id": inviteLink.ID}, bson.M{
		"$inc": bson.M{"usedCount": 1},
	})

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Invite accepted. Waiting for organizer approval.",
	})
}

// GetPendingApprovalsForTeam returns all pending player approvals for a team
func GetPendingApprovalsForTeam(c *fiber.Ctx) error {
	teamID := c.Params("id")
	userID := c.Locals("user_id").(string)

	// Convert IDs to ObjectID
	teamOID, err := primitive.ObjectIDFromHex(teamID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid team ID"})
	}

	ownerID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify user owns the team
	teamsCollection := db.MongoClient.Database("raidx").Collection("rbac_teams")
	err = teamsCollection.FindOne(ctx, bson.M{"_id": teamOID, "owner_id": ownerID}).Err()
	if err != nil {
		logrus.Warn("GetPendingApprovalsForTeam: Team not found or unauthorized")
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Get pending approvals
	approvalsCollection := db.MongoClient.Database("raidx").Collection("pending_approvals")
	cursor, err := approvalsCollection.Find(ctx, bson.M{
		"teamId": teamOID,
		"status": "pending",
	})
	if err != nil {
		logrus.Error("GetPendingApprovalsForTeam: Failed to fetch approvals:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch approvals"})
	}
	defer cursor.Close(ctx)

	var approvals []models.PendingApproval
	if err = cursor.All(ctx, &approvals); err != nil {
		logrus.Error("GetPendingApprovalsForTeam: Failed to decode approvals:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode approvals"})
	}

	return c.JSON(approvals)
}

// GetPendingApprovalsForEvent returns all pending team approvals for an event
func GetPendingApprovalsForEvent(c *fiber.Ctx) error {
	eventID := c.Params("id")
	userID := c.Locals("user_id").(string)

	// Convert IDs to ObjectID
	eventOID, err := primitive.ObjectIDFromHex(eventID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid event ID"})
	}

	organizerID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify user is the organizer
	eventsCollection := db.MongoClient.Database("raidx").Collection("rbac_events")
	err = eventsCollection.FindOne(ctx, bson.M{"_id": eventOID, "organizer_id": organizerID}).Err()
	if err != nil {
		logrus.Warn("GetPendingApprovalsForEvent: Event not found or unauthorized")
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Get pending approvals
	approvalsCollection := db.MongoClient.Database("raidx").Collection("pending_approvals")
	cursor, err := approvalsCollection.Find(ctx, bson.M{
		"eventId": eventOID,
		"status":  "pending",
	})
	if err != nil {
		logrus.Error("GetPendingApprovalsForEvent: Failed to fetch approvals:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch approvals"})
	}
	defer cursor.Close(ctx)

	var approvals []models.PendingApproval
	if err = cursor.All(ctx, &approvals); err != nil {
		logrus.Error("GetPendingApprovalsForEvent: Failed to decode approvals:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode approvals"})
	}

	return c.JSON(approvals)
}

// ApprovePendingApproval allows team owner/organizer to approve a pending acceptance
func ApprovePendingApproval(c *fiber.Ctx) error {
	approvalID := c.Params("id")
	userID := c.Locals("user_id").(string)

	appID, err := primitive.ObjectIDFromHex(approvalID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid approval ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get pending approval
	approvalsCollection := db.MongoClient.Database("raidx").Collection("pending_approvals")
	var approval models.PendingApproval
	err = approvalsCollection.FindOne(ctx, bson.M{"_id": appID}).Decode(&approval)
	if err != nil {
		logrus.Warn("ApprovePendingApproval: Approval not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Approval not found"})
	}

	// Verify user is owner/organizer
	if approval.FromID != userID {
		logrus.Warn("ApprovePendingApproval: Unauthorized approval attempt")
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Update approval status
	_, err = approvalsCollection.UpdateOne(ctx, bson.M{"_id": appID}, bson.M{
		"$set": bson.M{
			"status":     "approved",
			"approvedAt": time.Now(),
		},
	})
	if err != nil {
		logrus.Error("ApprovePendingApproval: Failed to update approval:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to approve"})
	}

	// Now add the player/team to the team/event based on type
	if approval.Type == models.InviteLinkTypeTeam {
		// Add player to team
		teamsCollection := db.MongoClient.Database("raidx").Collection("rbac_teams")
		_, err = teamsCollection.UpdateOne(ctx, bson.M{"_id": approval.TeamID}, bson.M{
			"$addToSet": bson.M{"players": approval.AcceptorID},
		})
		if err != nil {
			logrus.Error("ApprovePendingApproval: Failed to add player to team:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to add player to team"})
		}
	} else if approval.Type == models.InviteLinkTypeEvent {
		// Add team to event
		eventsCollection := db.MongoClient.Database("raidx").Collection("rbac_events")
		_, err = eventsCollection.UpdateOne(ctx, bson.M{"_id": approval.EventID}, bson.M{
			"$addToSet": bson.M{"participating_teams": approval.AcceptorID},
		})
		if err != nil {
			logrus.Error("ApprovePendingApproval: Failed to add team to event:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to add team to event"})
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Approval confirmed. User added successfully.",
	})
}

// RejectPendingApproval allows team owner/organizer to reject a pending acceptance
func RejectPendingApproval(c *fiber.Ctx) error {
	approvalID := c.Params("id")
	userID := c.Locals("user_id").(string)

	appID, err := primitive.ObjectIDFromHex(approvalID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid approval ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get pending approval
	approvalsCollection := db.MongoClient.Database("raidx").Collection("pending_approvals")
	var approval models.PendingApproval
	err = approvalsCollection.FindOne(ctx, bson.M{"_id": appID}).Decode(&approval)
	if err != nil {
		logrus.Warn("RejectPendingApproval: Approval not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Approval not found"})
	}

	// Verify user is owner/organizer
	if approval.FromID != userID {
		logrus.Warn("RejectPendingApproval: Unauthorized rejection attempt")
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Update approval status
	_, err = approvalsCollection.UpdateOne(ctx, bson.M{"_id": appID}, bson.M{
		"$set": bson.M{"status": "rejected"},
	})
	if err != nil {
		logrus.Error("RejectPendingApproval: Failed to update approval:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to reject"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Request rejected.",
	})
}
