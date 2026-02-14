package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Match struct matching your MongoDB document
type Match struct {
	ID        primitive.ObjectID  `json:"id" bson:"_id"`
	MatchID   string              `json:"matchId" bson:"matchId"`
	Type      string              `json:"type" bson:"type"`
	EventType string              `json:"eventType" bson:"event_type"`                 // "match" | "tournament" | "championship"
	EventID   *primitive.ObjectID `json:"eventId,omitempty" bson:"event_id,omitempty"` // Reference to event, tournament, or championship
	Data      struct {
		TeamA            TeamStat              `json:"teamA" bson:"teamA"`
		TeamB            TeamStat              `json:"teamB" bson:"teamB"`
		PlayerStats      map[string]PlayerStat `json:"playerStats" bson:"playerStats"`
		RaidDetails      RaidDetails           `json:"raidDetails" bson:"raidDetails"`
		TossWinner       string                `json:"tossWinner,omitempty" bson:"tossWinner,omitempty"`
		TossDecision     string                `json:"tossDecision,omitempty" bson:"tossDecision,omitempty"`
		FirstRaidingTeam string                `json:"firstRaidingTeam,omitempty" bson:"firstRaidingTeam,omitempty"`
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
	AllOut       bool     `json:"allOut,omitempty"`     // Indicates if this raid resulted in an all-out
	AllOutTeam   string   `json:"allOutTeam,omitempty"` // Which team got all-out (A or B)
}

type EnhancedStatsMessage struct {
	Type string `json:"type"`
	Data struct {
		TeamA            TeamStat              `json:"teamA"`
		TeamB            TeamStat              `json:"teamB"`
		PlayerStats      map[string]PlayerStat `json:"playerStats"`
		RaidDetails      RaidDetails           `json:"raidDetails"`
		RaidNumber       int                   `json:"raidNumber"`
		TeamAPlayerIDs   []string              `json:"teamAPlayerIds" bson:"teamAPlayerIds"`
		TeamBPlayerIDs   []string              `json:"teamBPlayerIds" bson:"teamBPlayerIds"`
		TossWinner       string                `json:"tossWinner,omitempty" bson:"tossWinner,omitempty"`
		TossDecision     string                `json:"tossDecision,omitempty" bson:"tossDecision,omitempty"`
		FirstRaidingTeam string                `json:"firstRaidingTeam,omitempty" bson:"firstRaidingTeam,omitempty"`
		EmptyRaidCounts  struct {
			TeamA int `json:"teamA"`
			TeamB int `json:"teamB"`
		} `json:"emptyRaidCounts"`
	} `json:"data"`
}
