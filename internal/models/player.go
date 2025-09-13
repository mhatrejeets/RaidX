package models

import "time"

// PlayerStat represents a player’s stats (dynamic keys in MongoDB)
type PlayerStatt struct {
	Name          string `json:"name" bson:"name"`
	RaidPoints    int    `json:"raidPoints" bson:"raidPoints"`
	DefencePoints int    `json:"defencePoints" bson:"defencePoints"`
	TotalPoints   int    `json:"totalPoints" bson:"totalPoints"`
	Status        string `json:"status" bson:"status"`
}

// Query to find the player by ID
var Player struct {
	FullName      string    `bson:"fullName"`
	Email         string    `bson:"email"`
	UserId        string    `bson:"userId"`
	Position      string    `bson:"position"`
	CreatedAt     time.Time `bson:"createdAt"`
	TotalPoints   int       `bson:"totalPoints"`
	RaidPoints    int       `bson:"raidPoints"`
	DefencePoints int       `bson:"defencePoints"`
}


type PlayerStat struct {
	Name          string `json:"name"`
	ID            string `json:"id"`
	TotalPoints   int    `json:"totalPoints"`
	RaidPoints    int    `json:"raidPoints"`
	DefencePoints int    `json:"defencePoints"`
	Status        string `json:"status"`
}