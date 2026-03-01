package handlers

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/models"
	"github.com/mhatrejeets/RaidX/internal/redisImpl"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// InitializeChampionshipHandler creates a championship and generates first round fixtures
func InitializeChampionshipHandler(c *fiber.Ctx) error {
	eventID := c.Params("id")
	if eventID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Event ID is required"})
	}

	eventObjID, err := primitive.ObjectIDFromHex(eventID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid event ID"})
	}

	// Check if event exists and has invitations accepted
	var event models.Event
	err = db.EventsCollection.FindOne(context.Background(), bson.M{"_id": eventObjID}).Decode(&event)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Event not found"})
		}
		logrus.Errorf("Error finding event: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to find event"})
	}

	// Extract team IDs from event's participating teams with accepted status
	var teamIDs []primitive.ObjectID
	for _, teamEntry := range event.ParticipatingTeams {
		if teamEntry.Status == models.EventTeamStatusAccepted && teamEntry.TeamID != primitive.NilObjectID {
			teamIDs = append(teamIDs, teamEntry.TeamID)
		}
	}
	teamIDs = uniqueObjectIDs(teamIDs)

	// Validate that we have at least 2 teams
	if len(teamIDs) < 2 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":          "At least 2 teams required for championship",
			"accepted_teams": len(teamIDs),
		})
	}

	// Calculate total rounds needed
	totalRounds := calculateTotalRounds(len(teamIDs))

	// Create championship document
	championship := models.Championship{
		ID:           primitive.NewObjectID(),
		EventID:      eventObjID,
		Status:       models.ChampionshipStatusOngoing,
		CurrentRound: 1,
		TotalRounds:  totalRounds,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err = db.ChampionshipsCollection.InsertOne(context.Background(), championship)
	if err != nil {
		logrus.Errorf("Error inserting championship: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create championship"})
	}

	// Initialize championship stats for all teams
	for _, teamID := range teamIDs {
		stats := models.ChampionshipStats{
			ID:             primitive.NewObjectID(),
			ChampionshipID: championship.ID,
			TeamID:         teamID,
			MatchesPlayed:  0,
			PointsScored:   0,
			PointsConceded: 0,
			NRR:            0.0,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		_, err = db.ChampionshipStatsCollection.InsertOne(context.Background(), stats)
		if err != nil {
			logrus.Errorf("Error inserting championship stats: %v", err)
		}
	}

	// Generate first round fixtures with random bye if odd number of teams
	err = generateRound(championship.ID, 1, teamIDs, true)
	if err != nil {
		logrus.Errorf("Error generating first round: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate first round"})
	}

	// Update event status
	_, err = db.EventsCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": eventObjID},
		bson.M{"$set": bson.M{"status": "ongoing"}},
	)
	if err != nil {
		logrus.Errorf("Error updating event status: %v", err)
	}

	return c.JSON(fiber.Map{
		"message":      "Championship initialized successfully",
		"championship": championship,
	})
}

// calculateTotalRounds determines the number of rounds needed for knockout
func calculateTotalRounds(numTeams int) int {
	return int(math.Ceil(math.Log2(float64(numTeams))))
}

// generateRound creates fixtures for a specific round with bye handling
func generateRound(championshipID primitive.ObjectID, roundNumber int, qualifiedTeams []primitive.ObjectID, isFirstRound bool) error {
	qualifiedTeams = uniqueObjectIDs(qualifiedTeams)
	numTeams := len(qualifiedTeams)
	if numTeams == 0 {
		return nil
	}

	var byeTeamID primitive.ObjectID
	hasBye := false
	matchTeams := make([]primitive.ObjectID, 0, numTeams)

	// Handle odd number of teams
	if numTeams%2 != 0 {
		if isFirstRound {
			// Random bye for first round
			rand.Seed(time.Now().UnixNano())
			byeIndex := rand.Intn(numTeams)
			byeTeamID = qualifiedTeams[byeIndex]
			hasBye = true
			// Remove bye team from match list
			matchTeams = append(matchTeams, qualifiedTeams[:byeIndex]...)
			matchTeams = append(matchTeams, qualifiedTeams[byeIndex+1:]...)
		} else {
			// Highest NRR gets bye in subsequent rounds
			highestNRRTeam, err := getHighestNRRTeam(championshipID, qualifiedTeams)
			if err != nil {
				logrus.Errorf("Error getting highest NRR team: %v", err)
				return err
			}
			byeTeamID = highestNRRTeam
			hasBye = true
			// Remove bye team from match list
			for _, teamID := range qualifiedTeams {
				if teamID == highestNRRTeam {
					continue
				}
				matchTeams = append(matchTeams, teamID)
			}
		}

		// Create bye fixture
		if !hasBye {
			return fmt.Errorf("failed to determine bye team for round %d", roundNumber)
		}
		byeFixture := models.ChampionshipFixture{
			ID:             primitive.NewObjectID(),
			ChampionshipID: championshipID,
			RoundNumber:    roundNumber,
			Team1ID:        byeTeamID,
			Team2ID:        nil, // nil indicates bye
			IsBye:          true,
			Status:         models.ChampionshipFixtureStatusCompleted,
			WinnerID:       &byeTeamID,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		_, err := db.ChampionshipFixturesCollection.InsertOne(context.Background(), byeFixture)
		if err != nil {
			logrus.Errorf("Error inserting bye fixture: %v", err)
			return err
		}
	} else {
		matchTeams = append(matchTeams, qualifiedTeams...)
	}

	if len(matchTeams)%2 != 0 {
		return fmt.Errorf("invalid team count for pairing in round %d: %d", roundNumber, len(matchTeams))
	}

	// Shuffle teams for random pairing
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(matchTeams), func(i, j int) {
		matchTeams[i], matchTeams[j] = matchTeams[j], matchTeams[i]
	})

	// Create regular fixtures
	for i := 0; i < len(matchTeams); i += 2 {
		fixture := models.ChampionshipFixture{
			ID:             primitive.NewObjectID(),
			ChampionshipID: championshipID,
			RoundNumber:    roundNumber,
			Team1ID:        matchTeams[i],
			Team2ID:        &matchTeams[i+1],
			IsBye:          false,
			Status:         models.ChampionshipFixtureStatusPending,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		_, err := db.ChampionshipFixturesCollection.InsertOne(context.Background(), fixture)
		if err != nil {
			logrus.Errorf("Error inserting fixture: %v", err)
			return err
		}
	}

	return nil
}

// getHighestNRRTeam finds the team with the highest NRR among qualified teams
func getHighestNRRTeam(championshipID primitive.ObjectID, teamIDs []primitive.ObjectID) (primitive.ObjectID, error) {
	cursor, err := db.ChampionshipStatsCollection.Find(
		context.Background(),
		bson.M{
			"championshipId": championshipID,
			"teamId":         bson.M{"$in": teamIDs},
		},
	)
	if err != nil {
		return primitive.NilObjectID, err
	}
	defer cursor.Close(context.Background())

	var stats []models.ChampionshipStats
	if err := cursor.All(context.Background(), &stats); err != nil {
		return primitive.NilObjectID, err
	}

	if len(stats) == 0 {
		// If no stats found, return first team
		return teamIDs[0], nil
	}

	// Find team with highest NRR
	highestNRRTeam := stats[0].TeamID
	highestNRR := stats[0].NRR
	for _, stat := range stats[1:] {
		if stat.NRR > highestNRR {
			highestNRR = stat.NRR
			highestNRRTeam = stat.TeamID
		}
	}

	return highestNRRTeam, nil
}

func resolveChampionshipObjectID(idParam string) (primitive.ObjectID, error) {
	championshipObjID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return primitive.NilObjectID, err
	}

	var championship models.Championship
	err = db.ChampionshipsCollection.FindOne(
		context.Background(),
		bson.M{"_id": championshipObjID},
	).Decode(&championship)
	if err == nil {
		return championship.ID, nil
	}
	if err != mongo.ErrNoDocuments {
		return primitive.NilObjectID, err
	}

	err = db.ChampionshipsCollection.FindOne(
		context.Background(),
		bson.M{"eventId": championshipObjID},
	).Decode(&championship)
	if err != nil {
		return primitive.NilObjectID, err
	}

	return championship.ID, nil
}

func uniqueObjectIDs(ids []primitive.ObjectID) []primitive.ObjectID {
	seen := make(map[primitive.ObjectID]struct{}, len(ids))
	unique := make([]primitive.ObjectID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

// GetChampionshipFixturesHandler returns all fixtures for a championship
func GetChampionshipFixturesHandler(c *fiber.Ctx) error {
	championshipID := c.Params("id")
	championshipObjID, err := resolveChampionshipObjectID(championshipID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Championship not found"})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid championship ID"})
	}

	// Fetch fixtures
	cursor, err := db.ChampionshipFixturesCollection.Find(
		context.Background(),
		bson.M{"championshipId": championshipObjID},
	)
	if err != nil {
		logrus.Errorf("Error finding fixtures: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch fixtures"})
	}
	defer cursor.Close(context.Background())

	var fixtures []models.ChampionshipFixture
	if err := cursor.All(context.Background(), &fixtures); err != nil {
		logrus.Errorf("Error decoding fixtures: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode fixtures"})
	}

	// Populate team details for each fixture
	type FixtureWithTeams struct {
		models.ChampionshipFixture `bson:",inline"`
		Team1                      *models.Team `json:"team1,omitempty"`
		Team2                      *models.Team `json:"team2,omitempty"`
	}

	var fixturesWithTeams []FixtureWithTeams
	for _, fixture := range fixtures {
		fwt := FixtureWithTeams{ChampionshipFixture: fixture}

		// Get Team1
		var team1 models.Team
		err := db.TeamsCollection.FindOne(context.Background(), bson.M{"_id": fixture.Team1ID}).Decode(&team1)
		if err == nil {
			fwt.Team1 = &team1
		}

		// Get Team2 if not a bye
		if fixture.Team2ID != nil {
			var team2 models.Team
			err := db.TeamsCollection.FindOne(context.Background(), bson.M{"_id": *fixture.Team2ID}).Decode(&team2)
			if err == nil {
				fwt.Team2 = &team2
			}
		}

		fixturesWithTeams = append(fixturesWithTeams, fwt)
	}

	return c.JSON(fixturesWithTeams)
}

// GetChampionshipStatsHandler returns NRR stats for all teams
func GetChampionshipStatsHandler(c *fiber.Ctx) error {
	championshipID := c.Params("id")
	championshipObjID, err := resolveChampionshipObjectID(championshipID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Championship not found"})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid championship ID"})
	}

	cursor, err := db.ChampionshipStatsCollection.Find(
		context.Background(),
		bson.M{"championshipId": championshipObjID},
	)
	if err != nil {
		logrus.Errorf("Error finding stats: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch stats"})
	}
	defer cursor.Close(context.Background())

	var stats []models.ChampionshipStats
	if err := cursor.All(context.Background(), &stats); err != nil {
		logrus.Errorf("Error decoding stats: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode stats"})
	}

	// Populate team details
	type StatsWithTeam struct {
		models.ChampionshipStats `bson:",inline"`
		Team                     *models.Team `json:"team,omitempty"`
	}

	var statsWithTeams []StatsWithTeam
	for _, stat := range stats {
		swt := StatsWithTeam{ChampionshipStats: stat}

		var team models.Team
		err := db.TeamsCollection.FindOne(context.Background(), bson.M{"_id": stat.TeamID}).Decode(&team)
		if err == nil {
			swt.Team = &team
		}

		statsWithTeams = append(statsWithTeams, swt)
	}

	return c.JSON(statsWithTeams)
}

// StartChampionshipMatchHandler starts a match and redirects to player selection
func StartChampionshipMatchHandler(c *fiber.Ctx) error {
	championshipID := c.Params("id")
	fixtureID := c.Params("fixtureId")

	championshipObjID, err := primitive.ObjectIDFromHex(championshipID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid championship ID"})
	}

	fixtureObjID, err := primitive.ObjectIDFromHex(fixtureID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid fixture ID"})
	}

	// Get fixture
	var fixture models.ChampionshipFixture
	err = db.ChampionshipFixturesCollection.FindOne(
		context.Background(),
		bson.M{"_id": fixtureObjID},
	).Decode(&fixture)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Fixture not found"})
	}

	if fixture.Status != models.ChampionshipFixtureStatusPending {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Match already started or completed"})
	}

	// Create match ID for this fixture (used for stats lookup)
	matchID := primitive.NewObjectID()

	// Update fixture status to ongoing
	_, err = db.ChampionshipFixturesCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": fixtureObjID},
		bson.M{
			"$set": bson.M{
				"status":    models.ChampionshipFixtureStatusOngoing,
				"matchId":   matchID,
				"updatedAt": time.Now(),
			},
		},
	)
	if err != nil {
		logrus.Errorf("Error updating fixture status: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update fixture"})
	}

	return c.JSON(fiber.Map{
		"message":        "Match started successfully",
		"championshipId": championshipObjID.Hex(),
		"team1Id":        fixture.Team1ID.Hex(),
		"team2Id":        (*fixture.Team2ID).Hex(),
		"fixtureId":      fixtureID,
		"matchId":        matchID.Hex(),
	})
}

func ContinueChampionshipMatchHandler(c *fiber.Ctx) error {
	championshipID := c.Params("id")
	fixtureID := c.Params("fixtureId")

	championshipObjID, err := primitive.ObjectIDFromHex(championshipID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid championship ID"})
	}

	fixtureObjID, err := primitive.ObjectIDFromHex(fixtureID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid fixture ID"})
	}

	var fixture models.ChampionshipFixture
	err = db.ChampionshipFixturesCollection.FindOne(
		context.Background(),
		bson.M{"_id": fixtureObjID, "championshipId": championshipObjID},
	).Decode(&fixture)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Fixture not found"})
	}
	if fixture.Status != models.ChampionshipFixtureStatusOngoing || fixture.MatchID == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Fixture is not in ongoing state"})
	}
	if fixture.Team2ID == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot continue a bye fixture"})
	}

	forceScorerTakeover(fixture.MatchID.Hex(), fmt.Sprintf("/organizer/championship?id=%s", championshipObjID.Hex()))

	return c.JSON(fiber.Map{
		"matchId": fixture.MatchID.Hex(),
		"redirectUrl": fmt.Sprintf(
			"/scorer?team1_id=%s&team2_id=%s&championship_id=%s&championship_fixture_id=%s&match_id=%s&resume=1",
			fixture.Team1ID.Hex(),
			fixture.Team2ID.Hex(),
			championshipObjID.Hex(),
			fixtureObjID.Hex(),
			fixture.MatchID.Hex(),
		),
	})
}

func RestartChampionshipMatchHandler(c *fiber.Ctx) error {
	championshipID := c.Params("id")
	fixtureID := c.Params("fixtureId")

	championshipObjID, err := primitive.ObjectIDFromHex(championshipID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid championship ID"})
	}

	fixtureObjID, err := primitive.ObjectIDFromHex(fixtureID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid fixture ID"})
	}

	var fixture models.ChampionshipFixture
	err = db.ChampionshipFixturesCollection.FindOne(
		context.Background(),
		bson.M{"_id": fixtureObjID, "championshipId": championshipObjID},
	).Decode(&fixture)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Fixture not found"})
	}
	if fixture.Team2ID == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot restart a bye fixture"})
	}

	if fixture.MatchID != nil {
		_ = redisImpl.DeleteRedisKey("gameStats:" + fixture.MatchID.Hex())
		_ = redisImpl.DeleteRedisKey("scorer_lock:" + fixture.MatchID.Hex())
		_, _ = db.MongoClient.Database("raidx").Collection("match_snapshots").DeleteOne(context.Background(), bson.M{"matchId": fixture.MatchID.Hex()})
	}

	newMatchID := primitive.NewObjectID()
	_, err = db.ChampionshipFixturesCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": fixtureObjID},
		bson.M{
			"$set": bson.M{
				"status":     models.ChampionshipFixtureStatusOngoing,
				"matchId":    newMatchID,
				"winnerId":   nil,
				"team1Score": 0,
				"team2Score": 0,
				"updatedAt":  time.Now(),
			},
		},
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to restart fixture"})
	}

	return c.JSON(fiber.Map{
		"matchId": newMatchID.Hex(),
		"redirectUrl": fmt.Sprintf(
			"/organizer/playerselection/match?team1_id=%s&team2_id=%s&match_id=%s&championship_id=%s&championship_fixture_id=%s",
			fixture.Team1ID.Hex(),
			fixture.Team2ID.Hex(),
			newMatchID.Hex(),
			championshipObjID.Hex(),
			fixtureObjID.Hex(),
		),
	})
}

// updateChampionshipAfterMatch updates championship state after a match completes
func updateChampionshipAfterMatch(ctx context.Context, championshipID, fixtureID string, gameStats map[string]interface{}) error {
	fixtureObjID, err := primitive.ObjectIDFromHex(fixtureID)
	if err != nil {
		return err
	}

	// Get fixture directly by ID
	var fixture models.ChampionshipFixture
	err = db.ChampionshipFixturesCollection.FindOne(ctx, bson.M{"_id": fixtureObjID}).Decode(&fixture)
	if err != nil {
		return err
	}

	// Extract scores from gameStats
	dataVal, ok := gameStats["data"]
	if !ok || dataVal == nil {
		return fmt.Errorf("missing gameStats.data")
	}
	data, ok := dataVal.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid gameStats.data format")
	}

	team1Score := 0
	team2Score := 0
	var winnerID *primitive.ObjectID
	isDraw := false

	// Prefer teamStats if present (legacy), otherwise use teamA/teamB
	if teamStatsVal, ok := data["teamStats"]; ok && teamStatsVal != nil {
		teamStatsRaw, ok := teamStatsVal.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid gameStats.data.teamStats format")
		}
		for teamIDStr, statsRaw := range teamStatsRaw {
			stats, ok := statsRaw.(map[string]interface{})
			if !ok {
				continue
			}
			score := int(stats["score"].(float64))
			teamObjID, _ := primitive.ObjectIDFromHex(teamIDStr)
			if teamObjID == fixture.Team1ID {
				team1Score = score
			} else if fixture.Team2ID != nil && teamObjID == *fixture.Team2ID {
				team2Score = score
			}
		}
	} else {
		teamAVal, okA := data["teamA"].(map[string]interface{})
		teamBVal, okB := data["teamB"].(map[string]interface{})
		if !okA || !okB {
			return fmt.Errorf("missing gameStats.data.teamA/teamB")
		}
		if scoreA, ok := teamAVal["score"].(float64); ok {
			team1Score = int(scoreA)
		}
		if scoreB, ok := teamBVal["score"].(float64); ok {
			team2Score = int(scoreB)
		}
	}

	// Determine winner
	if team1Score > team2Score {
		winnerID = &fixture.Team1ID
	} else if team2Score > team1Score && fixture.Team2ID != nil {
		winnerID = fixture.Team2ID
	} else {
		isDraw = true
	}

	if isDraw {
		var championship models.Championship
		err = db.ChampionshipsCollection.FindOne(
			ctx,
			bson.M{"_id": fixture.ChampionshipID},
		).Decode(&championship)
		if err != nil {
			return err
		}

		isSemifinalOrFinal := fixture.RoundNumber >= championship.TotalRounds-1
		if isSemifinalOrFinal {
			return fmt.Errorf("%w: championship semifinal/final match cannot end in a tie", ErrKnockoutTieNotAllowed)
		}
	}

	// Update fixture with scores and winner
	updateDoc := bson.M{
		"team1Score": team1Score,
		"team2Score": team2Score,
		"status":     models.ChampionshipFixtureStatusCompleted,
		"updatedAt":  time.Now(),
	}
	if winnerID != nil {
		updateDoc["winnerId"] = *winnerID
	}

	_, err = db.ChampionshipFixturesCollection.UpdateOne(
		ctx,
		bson.M{"_id": fixture.ID},
		bson.M{"$set": updateDoc},
	)
	if err != nil {
		logrus.Errorf("Error updating championship fixture: %v", err)
		return err
	}

	// Update championship stats for both teams
	updateChampionshipStats(fixture.ChampionshipID, fixture.Team1ID, team1Score, team2Score)
	if fixture.Team2ID != nil {
		updateChampionshipStats(fixture.ChampionshipID, *fixture.Team2ID, team2Score, team1Score)
	}

	// Check if round is complete and generate next round
	checkAndGenerateNextRound(fixture.ChampionshipID, fixture.RoundNumber)

	return nil
}

// updateChampionshipStats updates NRR for a team
func updateChampionshipStats(championshipID, teamID primitive.ObjectID, scored, conceded int) {
	var stats models.ChampionshipStats
	err := db.ChampionshipStatsCollection.FindOne(
		context.Background(),
		bson.M{"championshipId": championshipID, "teamId": teamID},
	).Decode(&stats)
	if err != nil {
		logrus.Errorf("Error finding championship stats: %v", err)
		return
	}

	stats.MatchesPlayed++
	stats.PointsScored += scored
	stats.PointsConceded += conceded

	// Calculate NRR
	if stats.MatchesPlayed > 0 {
		stats.NRR = (float64(stats.PointsScored) / float64(stats.MatchesPlayed)) -
			(float64(stats.PointsConceded) / float64(stats.MatchesPlayed))
	}

	_, err = db.ChampionshipStatsCollection.UpdateOne(
		context.Background(),
		bson.M{"championshipId": championshipID, "teamId": teamID},
		bson.M{
			"$set": bson.M{
				"matchesPlayed":  stats.MatchesPlayed,
				"pointsScored":   stats.PointsScored,
				"pointsConceded": stats.PointsConceded,
				"nrr":            stats.NRR,
				"updatedAt":      time.Now(),
			},
		},
	)
	if err != nil {
		logrus.Errorf("Error updating championship stats: %v", err)
	}
}

// checkAndGenerateNextRound checks if current round is complete and generates next round
func checkAndGenerateNextRound(championshipID primitive.ObjectID, currentRound int) {
	// Check if all fixtures in current round are completed
	count, err := db.ChampionshipFixturesCollection.CountDocuments(
		context.Background(),
		bson.M{
			"championshipId": championshipID,
			"roundNumber":    currentRound,
			"status":         bson.M{"$ne": models.ChampionshipFixtureStatusCompleted},
		},
	)
	if err != nil || count > 0 {
		// Round not complete yet
		return
	}

	// Get all winners from current round
	cursor, err := db.ChampionshipFixturesCollection.Find(
		context.Background(),
		bson.M{
			"championshipId": championshipID,
			"roundNumber":    currentRound,
		},
	)
	if err != nil {
		logrus.Errorf("Error finding completed fixtures: %v", err)
		return
	}
	defer cursor.Close(context.Background())

	var fixtures []models.ChampionshipFixture
	if err := cursor.All(context.Background(), &fixtures); err != nil {
		logrus.Errorf("Error decoding fixtures: %v", err)
		return
	}

	var qualifiedTeams []primitive.ObjectID
	for _, fixture := range fixtures {
		if fixture.WinnerID != nil {
			qualifiedTeams = append(qualifiedTeams, *fixture.WinnerID)
		}
	}
	qualifiedTeams = uniqueObjectIDs(qualifiedTeams)

	// If only one team left, championship is complete
	if len(qualifiedTeams) == 1 {
		_, err := db.ChampionshipsCollection.UpdateOne(
			context.Background(),
			bson.M{"_id": championshipID},
			bson.M{
				"$set": bson.M{
					"status":    models.ChampionshipStatusCompleted,
					"winnerId":  qualifiedTeams[0],
					"updatedAt": time.Now(),
				},
			},
		)
		if err != nil {
			logrus.Errorf("Error updating championship status: %v", err)
		}

		// Update event status to completed
		var championship models.Championship
		err = db.ChampionshipsCollection.FindOne(
			context.Background(),
			bson.M{"_id": championshipID},
		).Decode(&championship)
		if err == nil {
			_, err = db.EventsCollection.UpdateOne(
				context.Background(),
				bson.M{"_id": championship.EventID},
				bson.M{
					"$set": bson.M{
						"status":   "completed",
						"winnerId": qualifiedTeams[0],
					},
				},
			)
			if err != nil {
				logrus.Errorf("Error updating event status: %v", err)
			}
		}

		logrus.Infof("Championship %s completed! Winner: %s", championshipID.Hex(), qualifiedTeams[0].Hex())
		return
	}

	// Generate next round
	nextRound := currentRound + 1
	err = generateRound(championshipID, nextRound, qualifiedTeams, false)
	if err != nil {
		logrus.Errorf("Error generating next round: %v", err)
		return
	}

	// Update championship current round
	_, err = db.ChampionshipsCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": championshipID},
		bson.M{
			"$set": bson.M{
				"currentRound": nextRound,
				"updatedAt":    time.Now(),
			},
		},
	)
	if err != nil {
		logrus.Errorf("Error updating championship current round: %v", err)
	}

	logrus.Infof("Generated round %d for championship %s with %d teams", nextRound, championshipID.Hex(), len(qualifiedTeams))
}

// GetChampionshipByIDHandler returns championship details
func GetChampionshipByIDHandler(c *fiber.Ctx) error {
	idParam := c.Params("id")

	// Try to resolve by championship ID first
	championshipObjID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid championship ID"})
	}

	var championship models.Championship
	err = db.ChampionshipsCollection.FindOne(
		context.Background(),
		bson.M{"_id": championshipObjID},
	).Decode(&championship)

	// If not found by ID, try by eventId
	if err == mongo.ErrNoDocuments {
		err = db.ChampionshipsCollection.FindOne(
			context.Background(),
			bson.M{"eventId": championshipObjID},
		).Decode(&championship)
	}

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Championship not found"})
		}
		logrus.Errorf("Error finding championship: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to find championship"})
	}

	return c.JSON(championship)
}
