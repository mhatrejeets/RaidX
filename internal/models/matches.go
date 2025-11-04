package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Match struct matching your MongoDB document
type Match struct {
	ID   primitive.ObjectID `json:"id" bson:"_id"`
	Type string             `json:"type" bson:"type"`
	Data struct {
		TeamA       TeamStat              `json:"teamA" bson:"teamA"`
		TeamB       TeamStat              `json:"teamB" bson:"teamB"`
		PlayerStats map[string]PlayerStat `json:"playerStats" bson:"playerStats"`
		RaidDetails RaidDetails           `json:"raidDetails" bson:"raidDetails"`
	} `json:"data" bson:"data"`
}

// TeamStat and PlayerStat are defined in other model files (teams.go, players.go)

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
		TeamA           TeamStat              `json:"teamA"`
		TeamB           TeamStat              `json:"teamB"`
		PlayerStats     map[string]PlayerStat `json:"playerStats"`
		RaidDetails     RaidDetails           `json:"raidDetails"`
		RaidNumber      int                   `json:"raidNumber"`
		EmptyRaidCounts struct {
			TeamA int `json:"teamA"`
			TeamB int `json:"teamB"`
		} `json:"emptyRaidCounts"`
	} `json:"data"`
}
