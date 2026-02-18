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
	Name              string `json:"name" bson:"name"`
	ID                string `json:"id" bson:"id"`
	RaidPoints        int    `json:"raidPoints" bson:"raidPoints"`
	DefencePoints     int    `json:"defencePoints" bson:"defencePoints"`
	TotalPoints       int    `json:"totalPoints" bson:"totalPoints"`
	SuperRaids        int    `json:"superRaids" bson:"superRaids"`
	SuperTackles      int    `json:"superTackles" bson:"superTackles"`
	TotalRaids        int    `json:"totalRaids" bson:"totalRaids"`
	SuccessfulRaids   int    `json:"successfulRaids" bson:"successfulRaids"`
	TotalTackles      int    `json:"totalTackles" bson:"totalTackles"`
	SuccessfulTackles int    `json:"successfulTackles" bson:"successfulTackles"`
	Status            string `json:"status" bson:"status"`
}
