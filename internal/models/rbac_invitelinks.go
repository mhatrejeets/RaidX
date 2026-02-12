package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	InviteLinkTypeTeam  = "team"
	InviteLinkTypeEvent = "event"
)

// InviteLink represents a shareable invite link for teams/events
type InviteLink struct {
	ID        primitive.ObjectID `bson:"_id"`
	Token     string             `bson:"token"`             // Unique token for this invite link
	Type      string             `bson:"type"`              // "team" or "event"
	FromID    string             `bson:"fromId"`            // Team owner or organizer ID
	TeamID    string             `bson:"teamId,omitempty"`  // Team being invited to (if type == "team")
	EventID   string             `bson:"eventId,omitempty"` // Event being invited to (if type == "event")
	TeamName  string             `bson:"teamName,omitempty"`
	EventName string             `bson:"eventName,omitempty"`
	CreatedAt time.Time          `bson:"createdAt"`
	ExpiresAt time.Time          `bson:"expiresAt"`
	MaxUses   int                `bson:"maxUses"`   // 0 = unlimited
	UsedCount int                `bson:"usedCount"` // How many times link has been used
	IsActive  bool               `bson:"isActive"`  // Can be deactivated by creator
}

// PendingApproval represents a user/team that accepted an invite link and is waiting for owner/organizer approval
type PendingApproval struct {
	ID               primitive.ObjectID `bson:"_id"`
	InviteLinkID     primitive.ObjectID `bson:"inviteLinkId"`
	Type             string             `bson:"type"`   // "team" or "event"
	FromID           string             `bson:"fromId"` // Team owner or organizer
	TeamID           string             `bson:"teamId,omitempty"`
	EventID          string             `bson:"eventId,omitempty"`
	AcceptorID       string             `bson:"acceptorId"`       // Player or team owner who accepted
	AcceptorUsername string             `bson:"acceptorUsername"` // Username
	AcceptorName     string             `bson:"acceptorName"`     // Full name
	AcceptorRole     string             `bson:"acceptorRole"`     // "player" or "team_owner"
	Status           string             `bson:"status"`           // "pending", "approved", "rejected"
	CreatedAt        time.Time          `bson:"createdAt"`
	ApprovedAt       time.Time          `bson:"approvedAt,omitempty"`
}
