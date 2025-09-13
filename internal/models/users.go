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
}

// User represents a player document in the DB
type User struct {
	FullName      string    `bson:"fullName"`
	Email         string    `bson:"email"`
	UserID        string    `bson:"userId"`
	Password      string    `bson:"password"`
	Position      string    `bson:"position"`
	CreatedAt     time.Time `bson:"createdAt"`
	TotalPoints   int       `bson:"totalPoints"`
	RaidPoints    int       `bson:"raidPoints"`
	DefencePoints int       `bson:"defencePoints"`
}
