package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	InviteTypeTeam  = "team_invite"
	InviteTypeEvent = "event_invite"

	InviteStatusPending        = "pending"
	InviteStatusAccepted       = "accepted"
	InviteStatusDeclined       = "declined"
	InviteStatusInvitedViaLink = "invited_via_link"
)

// Invitation represents a team or event invitation.
type Invitation struct {
	ID            primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	Type          string              `bson:"type" json:"type"`
	FromID        primitive.ObjectID  `bson:"from_id" json:"from_id"`
	ToID          primitive.ObjectID  `bson:"to_id" json:"to_id"`
	TeamID        *primitive.ObjectID `bson:"team_id,omitempty" json:"team_id,omitempty"`
	EventID       *primitive.ObjectID `bson:"event_id,omitempty" json:"event_id,omitempty"`
	InviteToken   string              `bson:"invite_token" json:"invite_token"`
	Status        string              `bson:"status" json:"status"`
	DeclineReason string              `bson:"decline_reason,omitempty" json:"decline_reason,omitempty"`
	Source        string              `bson:"source,omitempty" json:"source,omitempty"`
	CreatedAt     time.Time           `bson:"created_at" json:"created_at"`
	ExpiresAt     time.Time           `bson:"expires_at" json:"expires_at"`
}
