package handlers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func resolveTournamentByIDOrEventID(ctx context.Context, id string) (models.Tournament, error) {
	tournamentObjID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return models.Tournament{}, fmt.Errorf("invalid id")
	}

	var tournament models.Tournament
	err = db.TournamentsCollection.FindOne(ctx, bson.M{"_id": tournamentObjID}).Decode(&tournament)
	if err == nil {
		return tournament, nil
	}

	if !errors.Is(err, mongo.ErrNoDocuments) {
		return models.Tournament{}, err
	}

	err = db.TournamentsCollection.FindOne(ctx, bson.M{"eventId": tournamentObjID}).Decode(&tournament)
	if err != nil {
		return models.Tournament{}, err
	}

	return tournament, nil
}

// InitializeTournamentHandler creates tournament, generates fixtures and points table
func InitializeTournamentHandler(c *fiber.Ctx) error {
	eventID := c.Params("id")
	if eventID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Event ID required"})
	}

	eventObjID, err := primitive.ObjectIDFromHex(eventID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid event ID"})
	}

	ctx := context.Background()

	// Get event details
	var event models.Event
	err = db.EventsCollection.FindOne(ctx, bson.M{"_id": eventObjID}).Decode(&event)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Event not found"})
	}

	// Verify event type is tournament
	if event.EventType != "tournament" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Event type must be tournament"})
	}

	// Check if tournament already exists
	var existingTournament models.Tournament
	err = db.TournamentsCollection.FindOne(ctx, bson.M{"eventId": eventObjID}).Decode(&existingTournament)
	if err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":        "Tournament already initialized",
			"tournamentId": existingTournament.ID.Hex(),
		})
	}

	// Get accepted teams
	acceptedTeams, err := getAcceptedTeamsForEvent(ctx, eventObjID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get teams"})
	}

	if len(acceptedTeams) < 2 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Need at least 2 teams to start tournament"})
	}

	// Create tournament
	tournament := models.Tournament{
		ID:        primitive.NewObjectID(),
		EventID:   eventObjID,
		Phase:     models.TournamentPhaseLeague,
		Status:    models.TournamentStatusOngoing,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = db.TournamentsCollection.InsertOne(ctx, tournament)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create tournament"})
	}

	// Generate round-robin fixtures
	fixtures := generateRoundRobinFixtures(tournament.ID, acceptedTeams)
	if len(fixtures) > 0 {
		var fixturesDocs []interface{}
		for _, f := range fixtures {
			fixturesDocs = append(fixturesDocs, f)
		}
		_, err = db.FixturesCollection.InsertMany(ctx, fixturesDocs)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create fixtures"})
		}
	}

	// Initialize points table
	var pointsTableDocs []interface{}
	for _, teamID := range acceptedTeams {
		entry := models.PointsTableEntry{
			ID:             primitive.NewObjectID(),
			TournamentID:   tournament.ID,
			TeamID:         teamID,
			MatchesPlayed:  0,
			Wins:           0,
			Losses:         0,
			Draws:          0,
			Points:         0,
			PointsScored:   0,
			PointsConceded: 0,
			NRR:            0.0,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		pointsTableDocs = append(pointsTableDocs, entry)
	}

	if len(pointsTableDocs) > 0 {
		_, err = db.PointsTableCollection.InsertMany(ctx, pointsTableDocs)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create points table"})
		}
	}

	// Update event status to ongoing
	_, err = db.EventsCollection.UpdateOne(ctx, bson.M{"_id": eventObjID}, bson.M{
		"$set": bson.M{
			"status":    "ongoing",
			"updatedAt": time.Now(),
		},
	})

	return c.JSON(fiber.Map{
		"message":      "Tournament initialized successfully",
		"tournamentId": tournament.ID.Hex(),
		"fixtures":     len(fixtures),
		"teams":        len(acceptedTeams),
	})
}

// GetTournamentFixturesHandler retrieves all fixtures for a tournament
func GetTournamentFixturesHandler(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	if tournamentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tournament ID required"})
	}

	ctx := context.Background()

	// Get tournament by tournament ID or event ID
	tournament, err := resolveTournamentByIDOrEventID(ctx, tournamentID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tournament not found"})
		}
		if err.Error() == "invalid id" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid tournament ID"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load tournament"})
	}

	// Get all fixtures
	cursor, err := db.FixturesCollection.Find(ctx, bson.M{"tournamentId": tournament.ID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get fixtures"})
	}
	defer cursor.Close(ctx)

	var fixtures []models.Fixture
	if err = cursor.All(ctx, &fixtures); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode fixtures"})
	}

	// Enrich fixtures with team names
	enrichedFixtures := make([]fiber.Map, 0)
	for _, fixture := range fixtures {
		team1, _ := getTeamByID(ctx, fixture.Team1ID)
		team2, _ := getTeamByID(ctx, fixture.Team2ID)

		enrichedFixtures = append(enrichedFixtures, fiber.Map{
			"id":         fixture.ID.Hex(),
			"team1Id":    fixture.Team1ID.Hex(),
			"team1Name":  team1.Name,
			"team2Id":    fixture.Team2ID.Hex(),
			"team2Name":  team2.Name,
			"matchType":  fixture.MatchType,
			"status":     fixture.Status,
			"matchId":    getStringFromObjectID(fixture.MatchID),
			"winnerId":   getStringFromObjectID(fixture.WinnerID),
			"team1Score": fixture.Team1Score,
			"team2Score": fixture.Team2Score,
			"isDraw":     fixture.IsDraw,
		})
	}

	return c.JSON(fiber.Map{
		"tournament": fiber.Map{
			"id":      tournament.ID.Hex(),
			"eventId": tournament.EventID.Hex(),
			"phase":   tournament.Phase,
			"status":  tournament.Status,
		},
		"fixtures": enrichedFixtures,
	})
}

// GetTournamentStandingsHandler retrieves points table sorted by points and NRR
func GetTournamentStandingsHandler(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	if tournamentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tournament ID required"})
	}

	ctx := context.Background()

	tournament, err := resolveTournamentByIDOrEventID(ctx, tournamentID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tournament not found"})
		}
		if err.Error() == "invalid id" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid tournament ID"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load tournament"})
	}

	// Get standings sorted by points (desc), then NRR (desc)
	opts := options.Find().SetSort(bson.D{
		{Key: "points", Value: -1},
		{Key: "nrr", Value: -1},
	})

	cursor, err := db.PointsTableCollection.Find(ctx, bson.M{"tournamentId": tournament.ID}, opts)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get standings"})
	}
	defer cursor.Close(ctx)

	var standings []models.PointsTableEntry
	if err = cursor.All(ctx, &standings); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode standings"})
	}

	// Enrich with team names
	enrichedStandings := make([]fiber.Map, 0)
	for i, entry := range standings {
		team, _ := getTeamByID(ctx, entry.TeamID)

		enrichedStandings = append(enrichedStandings, fiber.Map{
			"position":       i + 1,
			"teamId":         entry.TeamID.Hex(),
			"teamName":       team.Name,
			"matchesPlayed":  entry.MatchesPlayed,
			"wins":           entry.Wins,
			"losses":         entry.Losses,
			"draws":          entry.Draws,
			"points":         entry.Points,
			"pointsScored":   entry.PointsScored,
			"pointsConceded": entry.PointsConceded,
			"nrr":            fmt.Sprintf("%.3f", entry.NRR),
		})
	}

	return c.JSON(fiber.Map{"standings": enrichedStandings})
}

// StartTournamentMatchHandler creates a match from a fixture and redirects to player selection
func StartTournamentMatchHandler(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	fixtureID := c.Params("fixtureId")

	if tournamentID == "" || fixtureID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tournament ID and Fixture ID required"})
	}

	fixtureObjID, err := primitive.ObjectIDFromHex(fixtureID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid fixture ID"})
	}

	ctx := context.Background()

	tournament, err := resolveTournamentByIDOrEventID(ctx, tournamentID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tournament not found"})
		}
		if err.Error() == "invalid id" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid tournament ID"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load tournament"})
	}

	// Get fixture
	var fixture models.Fixture
	err = db.FixturesCollection.FindOne(ctx, bson.M{"_id": fixtureObjID, "tournamentId": tournament.ID}).Decode(&fixture)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Fixture not found"})
	}

	// Check if fixture already started
	if fixture.Status != models.FixtureStatusPending {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Fixture already started or completed"})
	}

	// Create match document (similar to how matches are created for events)
	matchID := primitive.NewObjectID()

	// Update fixture status
	_, err = db.FixturesCollection.UpdateOne(ctx, bson.M{"_id": fixtureObjID}, bson.M{
		"$set": bson.M{
			"status":    models.FixtureStatusOngoing,
			"matchId":   matchID,
			"updatedAt": time.Now(),
		},
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update fixture"})
	}

	// Return match ID and fixture info for player selection
	return c.JSON(fiber.Map{
		"matchId":      matchID.Hex(),
		"fixtureId":    fixture.ID.Hex(),
		"tournamentId": tournament.ID.Hex(),
		"eventId":      tournament.EventID.Hex(),
		"team1Id":      fixture.Team1ID.Hex(),
		"team2Id":      fixture.Team2ID.Hex(),
		"redirectUrl": fmt.Sprintf(
			"/organizer/playerselection/%s?team_id=%s&team_key=teamA_selected&team1_id=%s&team2_id=%s&event_id=%s&tournament_id=%s&fixture_id=%s&match_id=%s",
			matchID.Hex(),
			fixture.Team1ID.Hex(),
			fixture.Team1ID.Hex(),
			fixture.Team2ID.Hex(),
			tournament.EventID.Hex(),
			tournament.ID.Hex(),
			fixtureID,
			matchID.Hex(),
		),
	})
}

// Helper functions

func generateRoundRobinFixtures(tournamentID primitive.ObjectID, teams []primitive.ObjectID) []models.Fixture {
	fixtures := make([]models.Fixture, 0)
	n := len(teams)

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			fixture := models.Fixture{
				ID:           primitive.NewObjectID(),
				TournamentID: tournamentID,
				Team1ID:      teams[i],
				Team2ID:      teams[j],
				MatchType:    models.FixtureTypeLeague,
				Status:       models.FixtureStatusPending,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			fixtures = append(fixtures, fixture)
		}
	}

	return fixtures
}

func getAcceptedTeamsForEvent(ctx context.Context, eventID primitive.ObjectID) ([]primitive.ObjectID, error) {
	cursor, err := db.InvitationsCollection.Find(ctx, bson.M{
		"type":     models.InviteTypeEvent,
		"event_id": eventID,
		"status":   models.InviteStatusAccepted,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var invitations []models.Invitation
	if err = cursor.All(ctx, &invitations); err != nil {
		return nil, err
	}

	teamIDs := make([]primitive.ObjectID, 0)
	for _, inv := range invitations {
		if inv.TeamID != nil {
			teamIDs = append(teamIDs, *inv.TeamID)
		}
	}

	return teamIDs, nil
}

func getTeamByID(ctx context.Context, teamID primitive.ObjectID) (models.Team, error) {
	var team models.Team
	err := db.TeamsCollection.FindOne(ctx, bson.M{"_id": teamID}).Decode(&team)
	return team, err
}

func getStringFromObjectID(objID *primitive.ObjectID) string {
	if objID == nil {
		return ""
	}
	return objID.Hex()
}
