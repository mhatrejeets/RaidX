package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// RaidDetails of the last raid
type RaidDetailss struct {
	Type         string `json:"type" bson:"type"`
	Raider       string `json:"raider" bson:"raider"`
	PointsGained int    `json:"pointsGained" bson:"pointsGained"`
}

// Match struct matching your MongoDB document
type Match struct {
	ID   primitive.ObjectID `json:"id" bson:"_id"`
	Type string             `json:"type" bson:"type"`
	Data struct {
		TeamA       Teamm                  `json:"teamA" bson:"teamA"`
		TeamB       Teamm                  `json:"teamB" bson:"teamB"`
		PlayerStats map[string]PlayerStatt `json:"playerStats" bson:"playerStats"`
		RaidDetails RaidDetailss           `json:"raidDetails" bson:"raidDetails"`
	} `json:"data" bson:"data"`
}

type PlayerStat struct {
	Name          string `json:"name"`
	ID            string `json:"id"`
	TotalPoints   int    `json:"totalPoints"`
	RaidPoints    int    `json:"raidPoints"`
	DefencePoints int    `json:"defencePoints"`
	Status        string `json:"status"`
}

type TeamStats struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

type RaidDetails struct {
	Type         string   `json:"type"`
	Raider       string   `json:"raider"`
	Defenders    []string `json:"defenders,omitempty"`
	PointsGained int      `json:"pointsGained,omitempty"`
	BonusTaken   bool     `json:"bonusTaken,omitempty"`
	SuperTackle  bool     `json:"superTackle,omitempty"`
}

type EnhancedStatsMessage struct {
	Type string `json:"type"`
	Data struct {
		TeamA       TeamStats             `json:"teamA"`
		TeamB       TeamStats             `json:"teamB"`
		PlayerStats map[string]PlayerStat `json:"playerStats"`
		RaidDetails RaidDetails           `json:"raidDetails"`
	} `json:"data"`
	Extra map[string]interface{} `json:"extra,omitempty"`
}
