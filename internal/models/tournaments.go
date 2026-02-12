package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Tournament phases
const (
	TournamentPhaseLeague    = "league"
	TournamentPhaseSemifinal = "semifinal"
	TournamentPhaseFinal     = "final"
)

// Tournament status
const (
	TournamentStatusOngoing   = "ongoing"
	TournamentStatusCompleted = "completed"
)

// Match types for fixtures
const (
	FixtureTypeLeague    = "league"
	FixtureTypeSemifinal = "semifinal"
	FixtureTypeFinal     = "final"
)

// Fixture status
const (
	FixtureStatusPending   = "pending"
	FixtureStatusOngoing   = "ongoing"
	FixtureStatusCompleted = "completed"
)

// Tournament represents the state and metadata of a tournament
type Tournament struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	EventID   primitive.ObjectID `json:"eventId" bson:"eventId"`
	Phase     string             `json:"phase" bson:"phase"`   // league, semifinal, final
	Status    string             `json:"status" bson:"status"` // ongoing, completed
	WinnerID  primitive.ObjectID `json:"winnerId,omitempty" bson:"winnerId,omitempty"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// Fixture represents a match fixture in a tournament
type Fixture struct {
	ID           primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	TournamentID primitive.ObjectID  `json:"tournamentId" bson:"tournamentId"`
	Team1ID      primitive.ObjectID  `json:"team1Id" bson:"team1Id"`
	Team2ID      primitive.ObjectID  `json:"team2Id" bson:"team2Id"`
	MatchType    string              `json:"matchType" bson:"matchType"`                 // league, semifinal, final
	Status       string              `json:"status" bson:"status"`                       // pending, ongoing, completed
	MatchID      *primitive.ObjectID `json:"matchId,omitempty" bson:"matchId,omitempty"` // Reference to actual match when started
	WinnerID     *primitive.ObjectID `json:"winnerId,omitempty" bson:"winnerId,omitempty"`
	Team1Score   int                 `json:"team1Score,omitempty" bson:"team1Score,omitempty"`
	Team2Score   int                 `json:"team2Score,omitempty" bson:"team2Score,omitempty"`
	IsDraw       bool                `json:"isDraw" bson:"isDraw"`
	CreatedAt    time.Time           `json:"createdAt" bson:"createdAt"`
	UpdatedAt    time.Time           `json:"updatedAt" bson:"updatedAt"`
}

// PointsTableEntry represents a team's standing in the tournament
type PointsTableEntry struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TournamentID   primitive.ObjectID `json:"tournamentId" bson:"tournamentId"`
	TeamID         primitive.ObjectID `json:"teamId" bson:"teamId"`
	MatchesPlayed  int                `json:"matchesPlayed" bson:"matchesPlayed"`
	Wins           int                `json:"wins" bson:"wins"`
	Losses         int                `json:"losses" bson:"losses"`
	Draws          int                `json:"draws" bson:"draws"`
	Points         int                `json:"points" bson:"points"` // 2 for win, 1 for draw, 0 for loss
	PointsScored   int                `json:"pointsScored" bson:"pointsScored"`
	PointsConceded int                `json:"pointsConceded" bson:"pointsConceded"`
	NRR            float64            `json:"nrr" bson:"nrr"` // Net Run Rate
	CreatedAt      time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt      time.Time          `json:"updatedAt" bson:"updatedAt"`
}
