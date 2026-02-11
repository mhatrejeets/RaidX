package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Struct matching MongoDB document
type Userr struct {
	ID       primitive.ObjectID `bson:"_id"`
	Email    string             `bson:"email"`
	Password string             `bson:"password"`
	Name     string             `bson:"fullName"`
	Role     string             `bson:"role"`
}

// User represents a user document in the DB (team_owner, organizer, or player)
type User struct {
	FullName  string    `bson:"fullName"`
	Email     string    `bson:"email"`
	UserID    string    `bson:"userId"`
	Password  string    `bson:"password"`
	Role      string    `bson:"role"` // player, team_owner, organizer
	CreatedAt time.Time `bson:"createdAt"`

	// Player-specific fields (only populated if Role == "player")
	Position      string `bson:"position,omitempty"`
	TotalPoints   int    `bson:"totalPoints,omitempty"`
	RaidPoints    int    `bson:"raidPoints,omitempty"`
	DefencePoints int    `bson:"defencePoints,omitempty"`
}
