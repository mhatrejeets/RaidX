package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	TeamStatusActive    = "active"
	TeamStatusDisbanded = "disbanded"
)

// TeamProfile represents a team owned by a single user with a roster of players.
type TeamProfile struct {
	ID          primitive.ObjectID   `bson:"_id,omitempty"`
	TeamName    string               `bson:"team_name"`
	OwnerID     primitive.ObjectID   `bson:"owner_id"`
	Description string               `bson:"description,omitempty"`
	Players     []primitive.ObjectID `bson:"players"`
	Status      string               `bson:"status"`
	CreatedAt   time.Time            `bson:"created_at"`
	UpdatedAt   time.Time            `bson:"updated_at"`
}
