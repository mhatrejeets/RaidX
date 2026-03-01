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
		TeamA             TeamStat              `json:"teamA" bson:"teamA"`
		TeamB             TeamStat              `json:"teamB" bson:"teamB"`
		PlayerStats       map[string]PlayerStat `json:"playerStats" bson:"playerStats"`
		RaidDetails       RaidDetails           `json:"raidDetails" bson:"raidDetails"`
		RaidLog           []RaidLogEntry        `json:"raidLog,omitempty" bson:"raidLog,omitempty"`
		PendingLobby      LobbyState            `json:"pendingLobby,omitempty" bson:"pendingLobby,omitempty"`
		Awards            MatchAwards           `json:"awards,omitempty" bson:"awards,omitempty"`
		TossWinner        string                `json:"tossWinner,omitempty" bson:"tossWinner,omitempty"`
		TossDecision      string                `json:"tossDecision,omitempty" bson:"tossDecision,omitempty"`
		FirstRaidingTeam  string                `json:"firstRaidingTeam,omitempty" bson:"firstRaidingTeam,omitempty"`
		LastScoreChangeAt int64                 `json:"lastScoreChangeAt,omitempty" bson:"lastScoreChangeAt,omitempty"`
	} `json:"data" bson:"data"`
}

// TeamStat and PlayerStat are defined in other model files (teams.go, players.go)

type RaidDetails struct {
	Type           string   `json:"type"`
	Raider         string   `json:"raider"`
	Defenders      []string `json:"defenders,omitempty"`
	PointsGained   int      `json:"pointsGained,omitempty"`
	BonusTaken     bool     `json:"bonusTaken,omitempty"`
	SuperRaid      bool     `json:"superRaid,omitempty"`
	SuperTackle    bool     `json:"superTackle,omitempty"`
	DoOrDie        bool     `json:"doOrDie,omitempty"`
	LobbyRaider    bool     `json:"lobbyRaider,omitempty"`
	LobbyDefenders []string `json:"lobbyDefenders,omitempty"`
	AllOut         bool     `json:"allOut,omitempty"`     // Indicates if this raid resulted in an all-out
	AllOutTeam     string   `json:"allOutTeam,omitempty"` // Which team got all-out (A or B)
}

type LobbyEvent struct {
	TouchedPlayerId string `json:"touchedPlayerId" bson:"touchedPlayerId"`
	IsRaider        bool   `json:"isRaider" bson:"isRaider"`
	ScoringTeam     string `json:"scoringTeam" bson:"scoringTeam"`
	RaidNumber      int    `json:"raidNumber,omitempty" bson:"raidNumber,omitempty"`
}

type LobbyState struct {
	Events []LobbyEvent `json:"events,omitempty" bson:"events,omitempty"`
}

type RaidLogEntry struct {
	RaidNumber  int          `json:"raidNumber" bson:"raidNumber"`
	RaidingTeam string       `json:"raidingTeam" bson:"raidingTeam"`
	RaiderId    string       `json:"raiderId" bson:"raiderId"`
	DefenderIds []string     `json:"defenderIds,omitempty" bson:"defenderIds,omitempty"`
	Result      string       `json:"result" bson:"result"`
	Points      int          `json:"points" bson:"points"`
	BonusTaken  bool         `json:"bonusTaken,omitempty" bson:"bonusTaken,omitempty"`
	SuperRaid   bool         `json:"superRaid,omitempty" bson:"superRaid,omitempty"`
	SuperTackle bool         `json:"superTackle,omitempty" bson:"superTackle,omitempty"`
	DoOrDie     bool         `json:"doOrDie,omitempty" bson:"doOrDie,omitempty"`
	LobbyEvents []LobbyEvent `json:"lobbyEvents,omitempty" bson:"lobbyEvents,omitempty"`
}

type AwardInfo struct {
	PlayerId string `json:"playerId" bson:"playerId"`
	Name     string `json:"name" bson:"name"`
	Points   int    `json:"points" bson:"points"`
}

type MatchAwards struct {
	MVP          AwardInfo `json:"mvp,omitempty" bson:"mvp,omitempty"`
	BestRaider   AwardInfo `json:"bestRaider,omitempty" bson:"bestRaider,omitempty"`
	BestDefender AwardInfo `json:"bestDefender,omitempty" bson:"bestDefender,omitempty"`
}

type EnhancedStatsMessage struct {
	Type string `json:"type"`
	Data struct {
		TeamA             TeamStat              `json:"teamA"`
		TeamB             TeamStat              `json:"teamB"`
		PlayerStats       map[string]PlayerStat `json:"playerStats"`
		RaidDetails       RaidDetails           `json:"raidDetails"`
		RaidLog           []RaidLogEntry        `json:"raidLog,omitempty"`
		PendingLobby      LobbyState            `json:"pendingLobby,omitempty"`
		Awards            MatchAwards           `json:"awards,omitempty"`
		RaidNumber        int                   `json:"raidNumber"`
		TeamAPlayerIDs    []string              `json:"teamAPlayerIds" bson:"teamAPlayerIds"`
		TeamBPlayerIDs    []string              `json:"teamBPlayerIds" bson:"teamBPlayerIds"`
		TossWinner        string                `json:"tossWinner,omitempty" bson:"tossWinner,omitempty"`
		TossDecision      string                `json:"tossDecision,omitempty" bson:"tossDecision,omitempty"`
		FirstRaidingTeam  string                `json:"firstRaidingTeam,omitempty" bson:"firstRaidingTeam,omitempty"`
		LastScoreChangeAt int64                 `json:"lastScoreChangeAt,omitempty" bson:"lastScoreChangeAt,omitempty"`
		EmptyRaidCounts   struct {
			TeamA int `json:"teamA"`
			TeamB int `json:"teamB"`
		} `json:"emptyRaidCounts"`
	} `json:"data"`
}
