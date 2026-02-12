package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/models"
	"github.com/mhatrejeets/RaidX/internal/redisImpl"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func EndGameHandler(c *fiber.Ctx) error {
	ctx := context.Background()

	// Get match_id from query param
	matchId := c.Query("match_id")
	if matchId == "" {
		logrus.Warn("Warning:", "EndGameHandler:", " No match_id provided")
		return c.Status(400).SendString("No match_id provided")
	}

	// Use per-match Redis key format
	redisKey := fmt.Sprintf("gameStats:%s", matchId)

	// 1. Fetch gameStats from Redis for this match
	val, err := redisImpl.RedisClient.Get(ctx, redisKey).Result()
	logrus.Info("EndGame Handler is invoked for match:", matchId)
	if err == redisImpl.RedisNull {
		logrus.Warn("Warning:", "EndGameHandler:", " No game data found in Redis for match:", matchId)
		return c.Status(404).SendString("No game data found in Redis")
	} else if err != nil {
		logrus.Error("Error:", "EndGameHandler:", " Redis error: %v", err)
		return c.Status(500).SendString("Redis error: " + err.Error())
	}

	logrus.Debug("EndGame Handler will delete ", val)

	// 2. Parse JSON into generic map
	var gameStats map[string]interface{}
	if err := json.Unmarshal([]byte(val), &gameStats); err != nil {
		logrus.Error("Error:", "EndGameHandler:", " Failed to parse Redis JSON: %v", err)
		return c.Status(500).SendString("Failed to parse Redis JSON: " + err.Error())
	}

	// Attach eventId if provided
	eventIDParam := c.Query("event_id")
	if eventIDParam != "" {
		gameStats["eventId"] = eventIDParam
		// Best-effort update event status to completed for non-tournament events
		if eventOID, err := primitive.ObjectIDFromHex(eventIDParam); err == nil {
			eventsColl := db.MongoClient.Database("raidx").Collection("events")
			var evt models.Event
			if err := eventsColl.FindOne(ctx, bson.M{"_id": eventOID}).Decode(&evt); err == nil {
				if evt.EventType != models.EventTypeTournament {
					_, _ = eventsColl.UpdateOne(ctx, bson.M{"_id": eventOID}, bson.M{
						"$set": bson.M{
							"status":     models.EventStatusCompleted,
							"updated_at": time.Now(),
						},
					})
				}
			}
		}
	}

	// Ensure matchId is stored in Mongo for shareable lookups
	gameStats["matchId"] = matchId
	if objID, err := primitive.ObjectIDFromHex(matchId); err == nil {
		gameStats["_id"] = objID
	}

	// 3. Insert full gameStats into matches collection
	matchesColl := db.MongoClient.Database("raidx").Collection("matches")
	logrus.Debug("EndGame Handler will insert ", gameStats)
	_, err = matchesColl.InsertOne(ctx, gameStats)
	if err != nil {
		logrus.Error("Error:", "EndGameHandler:", " Failed to insert into matches: %v", err)
		return c.Status(500).SendString("Failed to insert into matches: " + err.Error())
	}

	// 4. Update each player in players collection
	data := gameStats["data"].(map[string]interface{})
	playerStats := data["playerStats"].(map[string]interface{})
	playersColl := db.MongoClient.Database("raidx").Collection("players")

	for id, raw := range playerStats {
		player := raw.(map[string]interface{})

		update := bson.M{
			"$inc": bson.M{
				"totalPoints":   int(player["totalPoints"].(float64)),
				"raidPoints":    int(player["raidPoints"].(float64)),
				"defencePoints": int(player["defencePoints"].(float64)),
			},
		}

		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			logrus.Error("Error:", "EndGameHandler:", " Invalid player ID: %v", err)
			continue
		}

		_, err = playersColl.UpdateByID(ctx, objID, update)
		if err != nil {
			logrus.Error("Error:", "EndGameHandler:", " Failed to update player %s: %v", id, err)
		}

	}

	// 5. Clean up Redis key for this match
	if err := redisImpl.RedisClient.Del(ctx, redisKey).Err(); err != nil {
		logrus.Error("Error:", "EndGameHandler:", " Failed to delete Redis key for match %s: %v", matchId, err)
	}

	// 6. Handle tournament match completion
	tournamentIDParam := c.Query("tournament_id")
	fixtureIDParam := c.Query("fixture_id")

	if tournamentIDParam != "" && fixtureIDParam != "" {
		if err := updateTournamentAfterMatch(ctx, tournamentIDParam, fixtureIDParam, gameStats); err != nil {
			logrus.Error("Error:", "EndGameHandler:", " Failed to update tournament: %v", err)
			// Don't fail the whole endgame, just log the error
		}
	}

	// 7. Handle championship match completion
	championshipIDParam := c.Query("championship_id")
	championshipFixtureIDParam := c.Query("championship_fixture_id")

	if championshipIDParam != "" && championshipFixtureIDParam != "" {
		if err := updateChampionshipAfterMatch(ctx, championshipIDParam, championshipFixtureIDParam, gameStats); err != nil {
			logrus.Error("Error:", "EndGameHandler:", " Failed to update championship: %v", err)
			// Don't fail the whole endgame, just log the error
		}
	}

	return c.JSON(fiber.Map{"success": true, "matchId": matchId})
}

// updateTournamentAfterMatch updates points table, NRR, and checks for playoff generation
func updateTournamentAfterMatch(ctx context.Context, tournamentID, fixtureID string, gameStats map[string]interface{}) error {
	tournamentObjID, err := primitive.ObjectIDFromHex(tournamentID)
	if err != nil {
		return err
	}

	fixtureObjID, err := primitive.ObjectIDFromHex(fixtureID)
	if err != nil {
		return err
	}

	// Get fixture
	var fixture models.Fixture
	err = db.FixturesCollection.FindOne(ctx, bson.M{"_id": fixtureObjID}).Decode(&fixture)
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
			} else if teamObjID == fixture.Team2ID {
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
	} else if team2Score > team1Score {
		winnerID = &fixture.Team2ID
	} else {
		isDraw = true
	}

	// Update fixture
	_, err = db.FixturesCollection.UpdateOne(ctx, bson.M{"_id": fixtureObjID}, bson.M{
		"$set": bson.M{
			"status":     models.FixtureStatusCompleted,
			"winnerId":   winnerID,
			"team1Score": team1Score,
			"team2Score": team2Score,
			"isDraw":     isDraw,
			"updatedAt":  time.Now(),
		},
	})
	if err != nil {
		return err
	}

	// Update points table for both teams
	if err := updatePointsTableForTeam(ctx, tournamentObjID, fixture.Team1ID, team1Score, team2Score, winnerID, isDraw); err != nil {
		return err
	}
	if err := updatePointsTableForTeam(ctx, tournamentObjID, fixture.Team2ID, team2Score, team1Score, winnerID, isDraw); err != nil {
		return err
	}

	// Check if league phase is complete and generate playoffs if needed
	if fixture.MatchType == models.FixtureTypeLeague {
		if err := checkAndGeneratePlayoffs(ctx, tournamentObjID); err != nil {
			logrus.Error("Error:", "updateTournamentAfterMatch:", " Failed to generate playoffs: %v", err)
		}
	}

	// Check if tournament is complete
	if fixture.MatchType == models.FixtureTypeFinal {
		_, err = db.TournamentsCollection.UpdateOne(ctx, bson.M{"_id": tournamentObjID}, bson.M{
			"$set": bson.M{
				"status":    models.TournamentStatusCompleted,
				"winnerId":  winnerID,
				"updatedAt": time.Now(),
			},
		})
		// Mark event as completed only after final
		var tournament models.Tournament
		if err := db.TournamentsCollection.FindOne(ctx, bson.M{"_id": tournamentObjID}).Decode(&tournament); err == nil {
			_, _ = db.EventsCollection.UpdateOne(ctx, bson.M{"_id": tournament.EventID}, bson.M{
				"$set": bson.M{
					"status":     models.EventStatusCompleted,
					"updated_at": time.Now(),
				},
			})
		}
	}

	// If semifinal just completed, generate final
	if fixture.MatchType == models.FixtureTypeSemifinal && winnerID != nil {
		if err := generateFinalFixture(ctx, tournamentObjID, *winnerID); err != nil {
			logrus.Error("Error:", "updateTournamentAfterMatch:", " Failed to generate final: %v", err)
		}
	}

	return nil
}

// updatePointsTableForTeam updates a single team's points table entry
func updatePointsTableForTeam(ctx context.Context, tournamentID, teamID primitive.ObjectID, scored, conceded int, winnerID *primitive.ObjectID, isDraw bool) error {
	// Determine match result
	wins := 0
	losses := 0
	draws := 0
	points := 0

	if isDraw {
		draws = 1
		points = 1
	} else if winnerID != nil && *winnerID == teamID {
		wins = 1
		points = 2
	} else {
		losses = 1
		points = 0
	}

	// Get current entry
	var entry models.PointsTableEntry
	err := db.PointsTableCollection.FindOne(ctx, bson.M{
		"tournamentId": tournamentID,
		"teamId":       teamID,
	}).Decode(&entry)
	if err != nil {
		return err
	}

	// Calculate new values
	newMatchesPlayed := entry.MatchesPlayed + 1
	newWins := entry.Wins + wins
	newLosses := entry.Losses + losses
	newDraws := entry.Draws + draws
	newPoints := entry.Points + points
	newPointsScored := entry.PointsScored + scored
	newPointsConceded := entry.PointsConceded + conceded

	// Calculate NRR
	nrr := 0.0
	if newMatchesPlayed > 0 {
		nrr = (float64(newPointsScored) / float64(newMatchesPlayed)) - (float64(newPointsConceded) / float64(newMatchesPlayed))
	}

	// Update entry
	_, err = db.PointsTableCollection.UpdateOne(ctx, bson.M{
		"tournamentId": tournamentID,
		"teamId":       teamID,
	}, bson.M{
		"$set": bson.M{
			"matchesPlayed":  newMatchesPlayed,
			"wins":           newWins,
			"losses":         newLosses,
			"draws":          newDraws,
			"points":         newPoints,
			"pointsScored":   newPointsScored,
			"pointsConceded": newPointsConceded,
			"nrr":            nrr,
			"updatedAt":      time.Now(),
		},
	})

	return err
}

// checkAndGeneratePlayoffs checks if all league matches are done and generates playoffs
func checkAndGeneratePlayoffs(ctx context.Context, tournamentID primitive.ObjectID) error {
	// Check if all league fixtures are completed
	pendingCount, err := db.FixturesCollection.CountDocuments(ctx, bson.M{
		"tournamentId": tournamentID,
		"matchType":    models.FixtureTypeLeague,
		"status":       bson.M{"$ne": models.FixtureStatusCompleted},
	})
	if err != nil {
		return err
	}

	// If there are still pending league matches, don't generate playoffs yet
	if pendingCount > 0 {
		return nil
	}

	// Check if playoffs already generated
	existingPlayoffs, err := db.FixturesCollection.CountDocuments(ctx, bson.M{
		"tournamentId": tournamentID,
		"matchType":    bson.M{"$in": []string{models.FixtureTypeSemifinal, models.FixtureTypeFinal}},
	})
	if err != nil {
		return err
	}

	if existingPlayoffs > 0 {
		return nil // Playoffs already exist
	}

	// Get top 3 teams from points table
	opts := options.Find().SetSort(bson.D{
		{Key: "points", Value: -1},
		{Key: "nrr", Value: -1},
	}).SetLimit(3)

	cursor, err := db.PointsTableCollection.Find(ctx, bson.M{"tournamentId": tournamentID}, opts)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var standings []models.PointsTableEntry
	if err = cursor.All(ctx, &standings); err != nil {
		return err
	}

	if len(standings) < 3 {
		return fmt.Errorf("not enough teams for playoffs")
	}

	// Team 1 (1st place) goes directly to final
	// Team 2 vs Team 3 in semifinal
	semifinal := models.Fixture{
		ID:           primitive.NewObjectID(),
		TournamentID: tournamentID,
		Team1ID:      standings[1].TeamID, // 2nd place
		Team2ID:      standings[2].TeamID, // 3rd place
		MatchType:    models.FixtureTypeSemifinal,
		Status:       models.FixtureStatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err = db.FixturesCollection.InsertOne(ctx, semifinal)
	if err != nil {
		return err
	}

	// Update tournament phase
	_, err = db.TournamentsCollection.UpdateOne(ctx, bson.M{"_id": tournamentID}, bson.M{
		"$set": bson.M{
			"phase":     models.TournamentPhaseSemifinal,
			"updatedAt": time.Now(),
		},
	})

	logrus.Info("Info:", "checkAndGeneratePlayoffs:", " Generated semifinal for tournament:", tournamentID.Hex())

	return err
}

// generateFinalFixture is called after semifinal completion
func generateFinalFixture(ctx context.Context, tournamentID primitive.ObjectID, semifinalWinner primitive.ObjectID) error {
	// Get 1st place team from points table
	opts := options.Find().SetSort(bson.D{
		{Key: "points", Value: -1},
		{Key: "nrr", Value: -1},
	}).SetLimit(1)

	cursor, err := db.PointsTableCollection.Find(ctx, bson.M{"tournamentId": tournamentID}, opts)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var standings []models.PointsTableEntry
	if err = cursor.All(ctx, &standings); err != nil {
		return err
	}

	if len(standings) == 0 {
		return fmt.Errorf("no teams found in points table")
	}

	firstPlaceTeam := standings[0].TeamID

	// Create final fixture
	final := models.Fixture{
		ID:           primitive.NewObjectID(),
		TournamentID: tournamentID,
		Team1ID:      firstPlaceTeam,
		Team2ID:      semifinalWinner,
		MatchType:    models.FixtureTypeFinal,
		Status:       models.FixtureStatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err = db.FixturesCollection.InsertOne(ctx, final)
	if err != nil {
		return err
	}

	// Update tournament phase
	_, err = db.TournamentsCollection.UpdateOne(ctx, bson.M{"_id": tournamentID}, bson.M{
		"$set": bson.M{
			"phase":     models.TournamentPhaseFinal,
			"updatedAt": time.Now(),
		},
	})

	logrus.Info("Info:", "generateFinalFixture:", " Generated final for tournament:", tournamentID.Hex())

	return err
}
