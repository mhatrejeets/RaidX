package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Championship status
const (
	ChampionshipStatusOngoing   = "ongoing"
	ChampionshipStatusCompleted = "completed"
)

// Championship represents a knockout tournament
type Championship struct {
	ID           primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	EventID      primitive.ObjectID  `json:"eventId" bson:"eventId"`
	Status       string              `json:"status" bson:"status"` // ongoing, completed
	CurrentRound int                 `json:"currentRound" bson:"currentRound"`
	TotalRounds  int                 `json:"totalRounds" bson:"totalRounds"` // calculated based on teams
	WinnerID     *primitive.ObjectID `json:"winnerId,omitempty" bson:"winnerId,omitempty"`
	CreatedAt    time.Time           `json:"createdAt" bson:"createdAt"`
	UpdatedAt    time.Time           `json:"updatedAt" bson:"updatedAt"`
}

// ChampionshipFixture represents a match in the championship
type ChampionshipFixture struct {
	ID             primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	ChampionshipID primitive.ObjectID  `json:"championshipId" bson:"championshipId"`
	RoundNumber    int                 `json:"roundNumber" bson:"roundNumber"`
	Team1ID        primitive.ObjectID  `json:"team1Id" bson:"team1Id"`
	Team2ID        *primitive.ObjectID `json:"team2Id,omitempty" bson:"team2Id,omitempty"` // nil for bye
	IsBye          bool                `json:"isBye" bson:"isBye"`
	Status         string              `json:"status" bson:"status"` // pending, ongoing, completed
	MatchID        *primitive.ObjectID `json:"matchId,omitempty" bson:"matchId,omitempty"`
	WinnerID       *primitive.ObjectID `json:"winnerId,omitempty" bson:"winnerId,omitempty"`
	Team1Score     int                 `json:"team1Score,omitempty" bson:"team1Score,omitempty"`
	Team2Score     int                 `json:"team2Score,omitempty" bson:"team2Score,omitempty"`
	CreatedAt      time.Time           `json:"createdAt" bson:"createdAt"`
	UpdatedAt      time.Time           `json:"updatedAt" bson:"updatedAt"`
}

// ChampionshipStats represents team statistics in championship
type ChampionshipStats struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ChampionshipID primitive.ObjectID `json:"championshipId" bson:"championshipId"`
	TeamID         primitive.ObjectID `json:"teamId" bson:"teamId"`
	MatchesPlayed  int                `json:"matchesPlayed" bson:"matchesPlayed"`
	PointsScored   int                `json:"pointsScored" bson:"pointsScored"`
	PointsConceded int                `json:"pointsConceded" bson:"pointsConceded"`
	NRR            float64            `json:"nrr" bson:"nrr"` // Net Run Rate
	CreatedAt      time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt      time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// Fixture status constants (reuse from tournaments)
const (
	ChampionshipFixtureStatusPending   = "pending"
	ChampionshipFixtureStatusOngoing   = "ongoing"
	ChampionshipFixtureStatusCompleted = "completed"
)
