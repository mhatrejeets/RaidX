package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	InviteTypeTeam  = "team_invite"
	InviteTypeEvent = "event_invite"

	InviteStatusPending  = "pending"
	InviteStatusAccepted = "accepted"
	InviteStatusDeclined = "declined"
)

// Invitation represents a team or event invitation.
type Invitation struct {
	ID          primitive.ObjectID  `bson:"_id,omitempty"`
	Type        string              `bson:"type"`
	FromID      primitive.ObjectID  `bson:"from_id"`
	ToID        primitive.ObjectID  `bson:"to_id"`
	TeamID      *primitive.ObjectID `bson:"team_id,omitempty"`
	EventID     *primitive.ObjectID `bson:"event_id,omitempty"`
	InviteToken string              `bson:"invite_token"`
	Status      string              `bson:"status"`
	CreatedAt   time.Time           `bson:"created_at"`
	ExpiresAt   time.Time           `bson:"expires_at"`
}
