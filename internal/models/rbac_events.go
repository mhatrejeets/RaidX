package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	EventTypeMatch       = "match"
	EventTypeTournament  = "tournament"
	EventTypeChampionship = "championship"

	EventStatusDraft     = "draft"
	EventStatusActive    = "active"
	EventStatusCompleted = "completed"
)

const (
	EventTeamStatusInvited  = "invited"
	EventTeamStatusAccepted = "accepted"
	EventTeamStatusDeclined = "declined"
)

// EventTeamEntry represents a team participation entry in an event.
type EventTeamEntry struct {
	TeamID primitive.ObjectID `bson:"team_id"`
	Status string             `bson:"status"`
}

// Event represents a match/tournament/championship organized by an organizer.
type Event struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty"`
	OrganizerID        primitive.ObjectID `bson:"organizer_id"`
	EventName          string             `bson:"event_name"`
	EventType          string             `bson:"event_type"`
	ParticipatingTeams []EventTeamEntry   `bson:"participating_teams"`
	Status             string             `bson:"status"`
	CreatedAt          time.Time          `bson:"created_at"`
	UpdatedAt          time.Time          `bson:"updated_at"`
}
