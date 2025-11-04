package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Player struct {
	ID          primitive.ObjectID `bson:"_id"`
	FullName    string             `bson:"fullName"`
	TotalPoints int                `bson:"totalPoints"`
	Position    string             `bson:"position"`
}

// PlayerStat represents a player’s stats (dynamic keys in MongoDB)
type PlayerStat struct {
	Name          string `json:"name" bson:"name"`
	ID            string `json:"id" bson:"id"`
	RaidPoints    int    `json:"raidPoints" bson:"raidPoints"`
	DefencePoints int    `json:"defencePoints" bson:"defencePoints"`
	TotalPoints   int    `json:"totalPoints" bson:"totalPoints"`
	Status        string `json:"status" bson:"status"`
}
