package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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

	// Ensure matchId is stored in Mongo for shareable lookups
	gameStats["matchId"] = matchId
	if objID, err := primitive.ObjectIDFromHex(matchId); err == nil {
		gameStats["_id"] = objID
	}

	// Set event_type and event_id based on query params
	tournamentIDParam := c.Query("tournament_id")
	championshipIDParam := c.Query("championship_id")
	eventIDParam := c.Query("event_id")

	if tournamentIDParam != "" {
		gameStats["event_type"] = "tournament"
		if tournamentOID, err := primitive.ObjectIDFromHex(tournamentIDParam); err == nil {
			// Get tournament to find the event ID
			var tournament models.Tournament
			if err := db.TournamentsCollection.FindOne(ctx, bson.M{"_id": tournamentOID}).Decode(&tournament); err == nil {
				gameStats["event_id"] = tournament.EventID
			}
		}
	} else if championshipIDParam != "" {
		gameStats["event_type"] = "championship"
		if championshipOID, err := primitive.ObjectIDFromHex(championshipIDParam); err == nil {
			// Get championship to find the event ID
			var championship models.Championship
			if err := db.ChampionshipsCollection.FindOne(ctx, bson.M{"_id": championshipOID}).Decode(&championship); err == nil {
				gameStats["event_id"] = championship.EventID
			}
		}
	} else if eventIDParam != "" {
		// Standalone match event
		gameStats["event_type"] = "match"
		if eventOID, err := primitive.ObjectIDFromHex(eventIDParam); err == nil {
			gameStats["event_id"] = eventOID

			// Update event status to completed for standalone match events
			eventsColl := db.MongoClient.Database("raidx").Collection("events")
			var evt models.Event
			if err := eventsColl.FindOne(ctx, bson.M{"_id": eventOID}).Decode(&evt); err == nil {
				if evt.EventType != models.EventTypeTournament && evt.EventType != models.EventTypeChampionship {
					_, _ = eventsColl.UpdateOne(ctx, bson.M{"_id": eventOID}, bson.M{
						"$set": bson.M{
							"status":     models.EventStatusCompleted,
							"updated_at": time.Now(),
						},
					})
				}
			}
		}
	} else {
		// No event association (legacy/standalone)
		gameStats["event_type"] = "match"
	}

	// 3. Handle tournament match completion BEFORE saving match (to set fixture's matchId)
	fixtureIDParam := c.Query("fixture_id")

	if tournamentIDParam != "" && fixtureIDParam != "" {
		if err := updateTournamentAfterMatch(ctx, tournamentIDParam, fixtureIDParam, gameStats); err != nil {
			logrus.Error("Error:", "EndGameHandler:", " Failed to update tournament: %v", err)
			// Don't fail the whole endgame, just log the error
		}
	}

	// 4. Handle championship match completion
	championshipFixtureIDParam := c.Query("championship_fixture_id")

	if championshipIDParam != "" && championshipFixtureIDParam != "" {
		if err := updateChampionshipAfterMatch(ctx, championshipIDParam, championshipFixtureIDParam, gameStats); err != nil {
			logrus.Error("Error:", "EndGameHandler:", " Failed to update championship: %v", err)
			// Don't fail the whole endgame, just log the error
		}
	}

	// 5. Insert full gameStats into matches collection (after event_type and event_id are set)
	matchesColl := db.MongoClient.Database("raidx").Collection("matches")
	logrus.Debug("EndGame Handler will insert ", gameStats)
	_, err = matchesColl.InsertOne(ctx, gameStats)
	if err != nil {
		logrus.Error("Error:", "EndGameHandler:", " Failed to insert into matches: %v", err)
		return c.Status(500).SendString("Failed to insert into matches: " + err.Error())
	}

	// Update rankings for tournament/championship events
	if eventType, ok := gameStats["event_type"].(string); ok {
		if eventType == "tournament" || eventType == "championship" {
			if eventOID, ok := getEventIDFromGameStats(gameStats); ok {
				if err := updateEventRankings(ctx, eventType, eventOID); err != nil {
					logrus.Error("Error:", "EndGameHandler:", " Failed to update rankings: %v", err)
				}
			}
		}
	}

	// 6. Update each player in players collection
	data := gameStats["data"].(map[string]interface{})
	playerStats := data["playerStats"].(map[string]interface{})
	data["awards"] = computeMatchAwards(playerStats)
	playersColl := db.MongoClient.Database("raidx").Collection("players")

	awardsMap, _ := data["awards"].(map[string]interface{})
	mvpID := getAwardPlayerID(awardsMap, "mvp")
	bestRaiderID := getAwardPlayerID(awardsMap, "bestRaider")
	bestDefenderID := getAwardPlayerID(awardsMap, "bestDefender")

	for id, raw := range playerStats {
		player := raw.(map[string]interface{})

		mvpInc := 0
		bestRaiderInc := 0
		bestDefenderInc := 0
		if id == mvpID {
			mvpInc = 1
		}
		if id == bestRaiderID {
			bestRaiderInc = 1
		}
		if id == bestDefenderID {
			bestDefenderInc = 1
		}

		update := bson.M{
			"$inc": bson.M{
				"totalPoints":       int(player["totalPoints"].(float64)),
				"raidPoints":        int(player["raidPoints"].(float64)),
				"defencePoints":     int(player["defencePoints"].(float64)),
				"superRaids":        int(getFloatOrZero(player, "superRaids")),
				"superTackles":      int(getFloatOrZero(player, "superTackles")),
				"totalRaids":        int(getFloatOrZero(player, "totalRaids")),
				"successfulRaids":   int(getFloatOrZero(player, "successfulRaids")),
				"totalTackles":      int(getFloatOrZero(player, "totalTackles")),
				"successfulTackles": int(getFloatOrZero(player, "successfulTackles")),
				"matchesPlayed":     1,
				"mvpCount":          mvpInc,
				"bestRaiderCount":   bestRaiderInc,
				"bestDefenderCount": bestDefenderInc,
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

	// 7. Clean up Redis key for this match
	if err := redisImpl.RedisClient.Del(ctx, redisKey).Err(); err != nil {
		logrus.Error("Error:", "EndGameHandler:", " Failed to delete Redis key for match %s: %v", matchId, err)
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
	// Fetch semifinal fixture to identify the two teams that played semifinal.
	// The bye finalist must be the top-ranked team excluding these two.
	var semifinal models.Fixture
	if err := db.FixturesCollection.FindOne(ctx, bson.M{
		"tournamentId": tournamentID,
		"matchType":    models.FixtureTypeSemifinal,
	}).Decode(&semifinal); err != nil {
		return fmt.Errorf("semifinal fixture not found: %w", err)
	}

	if semifinal.Team1ID == semifinalWinner {
		// valid, keep going
	} else if semifinal.Team2ID == semifinalWinner {
		// valid, keep going
	} else {
		return fmt.Errorf("semifinal winner does not belong to semifinal fixture")
	}

	// Select top-ranked team from points table excluding semifinal teams.
	// This locks the league topper (bye finalist) and prevents same-team finals
	// even if semifinal points re-order the top 3.
	opts := options.FindOne().SetSort(bson.D{{Key: "points", Value: -1}, {Key: "nrr", Value: -1}})
	var byeEntry models.PointsTableEntry
	if err := db.PointsTableCollection.FindOne(ctx, bson.M{
		"tournamentId": tournamentID,
		"teamId": bson.M{"$nin": []primitive.ObjectID{
			semifinal.Team1ID,
			semifinal.Team2ID,
		}},
	}, opts).Decode(&byeEntry); err != nil {
		return fmt.Errorf("failed to resolve bye finalist: %w", err)
	}

	firstPlaceTeam := byeEntry.TeamID
	if firstPlaceTeam == semifinalWinner {
		return fmt.Errorf("invalid final pairing: bye finalist equals semifinal winner")
	}

	// Avoid duplicate final fixture creation.
	existingFinalCount, err := db.FixturesCollection.CountDocuments(ctx, bson.M{
		"tournamentId": tournamentID,
		"matchType":    models.FixtureTypeFinal,
	})
	if err != nil {
		return err
	}
	if existingFinalCount > 0 {
		return nil
	}

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

func getFloatOrZero(m map[string]interface{}, key string) float64 {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	if f, ok := v.(float64); ok {
		return f
	}
	if i, ok := v.(int); ok {
		return float64(i)
	}
	return 0
}

func computeMatchAwards(playerStats map[string]interface{}) map[string]interface{} {
	type stat struct {
		id    string
		name  string
		total int
		raid  int
		def   int
	}
	list := []stat{}
	for id, raw := range playerStats {
		p, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := p["name"].(string)
		if name == "" {
			if n, ok := p["Name"].(string); ok {
				name = n
			}
		}
		list = append(list, stat{
			id:    id,
			name:  name,
			total: int(getFloatOrZero(p, "totalPoints")),
			raid:  int(getFloatOrZero(p, "raidPoints")),
			def:   int(getFloatOrZero(p, "defencePoints")),
		})
	}
	if len(list) == 0 {
		return map[string]interface{}{}
	}
	bestMVP := list[0]
	bestRaider := list[0]
	bestDefender := list[0]
	for _, s := range list[1:] {
		if s.total > bestMVP.total || (s.total == bestMVP.total && s.raid > bestMVP.raid) || (s.total == bestMVP.total && s.raid == bestMVP.raid && s.def > bestMVP.def) {
			bestMVP = s
		}
		if s.raid > bestRaider.raid || (s.raid == bestRaider.raid && s.total > bestRaider.total) {
			bestRaider = s
		}
		if s.def > bestDefender.def || (s.def == bestDefender.def && s.total > bestDefender.total) {
			bestDefender = s
		}
	}
	return map[string]interface{}{
		"mvp": map[string]interface{}{
			"playerId": bestMVP.id,
			"name":     bestMVP.name,
			"points":   bestMVP.total,
		},
		"bestRaider": map[string]interface{}{
			"playerId": bestRaider.id,
			"name":     bestRaider.name,
			"points":   bestRaider.raid,
		},
		"bestDefender": map[string]interface{}{
			"playerId": bestDefender.id,
			"name":     bestDefender.name,
			"points":   bestDefender.def,
		},
	}
}

func getEventIDFromGameStats(gameStats map[string]interface{}) (primitive.ObjectID, bool) {
	if gameStats == nil {
		return primitive.NilObjectID, false
	}
	if v, ok := gameStats["event_id"]; ok {
		switch t := v.(type) {
		case primitive.ObjectID:
			return t, true
		case string:
			if oid, err := primitive.ObjectIDFromHex(t); err == nil {
				return oid, true
			}
		}
	}
	return primitive.NilObjectID, false
}

func getAwardPlayerID(awards map[string]interface{}, key string) string {
	if awards == nil {
		return ""
	}
	awardRaw, ok := awards[key]
	if !ok {
		return ""
	}
	award, ok := awardRaw.(map[string]interface{})
	if !ok {
		return ""
	}
	if playerID, ok := award["playerId"].(string); ok {
		return playerID
	}
	return ""
}

func updateEventRankings(ctx context.Context, eventType string, eventID primitive.ObjectID) error {
	matchesColl := db.MongoClient.Database("raidx").Collection("matches")
	filter := bson.M{"$or": []bson.M{
		{"event_id": eventID},
		{"event_id": eventID.Hex()},
		{"eventId": eventID.Hex()},
	}}
	cursor, err := matchesColl.Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	type agg struct {
		id    string
		name  string
		total int
		raid  int
		def   int
	}
	acc := map[string]*agg{}

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		dataVal, ok := doc["data"]
		if !ok {
			continue
		}
		data, ok := dataVal.(bson.M)
		if !ok {
			if m, ok2 := dataVal.(map[string]interface{}); ok2 {
				data = bson.M(m)
			} else {
				continue
			}
		}
		psVal, ok := data["playerStats"]
		if !ok {
			continue
		}
		psMap, ok := psVal.(bson.M)
		if !ok {
			if m, ok2 := psVal.(map[string]interface{}); ok2 {
				psMap = bson.M(m)
			} else {
				continue
			}
		}
		for id, raw := range psMap {
			p, ok := raw.(map[string]interface{})
			if !ok {
				if bm, ok2 := raw.(bson.M); ok2 {
					p = map[string]interface{}(bm)
				} else {
					continue
				}
			}
			name, _ := p["name"].(string)
			if name == "" {
				if n, ok := p["Name"].(string); ok {
					name = n
				}
			}
			entry, ok := acc[id]
			if !ok {
				entry = &agg{id: id, name: name}
				acc[id] = entry
			}
			entry.total += int(getFloatOrZero(p, "totalPoints"))
			entry.raid += int(getFloatOrZero(p, "raidPoints"))
			entry.def += int(getFloatOrZero(p, "defencePoints"))
		}
	}

	list := []agg{}
	for _, v := range acc {
		list = append(list, *v)
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].total != list[j].total {
			return list[i].total > list[j].total
		}
		if list[i].raid != list[j].raid {
			return list[i].raid > list[j].raid
		}
		return list[i].def > list[j].def
	})
	toTop := func(items []agg, limit int, pick func(a agg) int) []bson.M {
		if len(items) == 0 {
			return []bson.M{}
		}
		cpy := append([]agg(nil), items...)
		sort.Slice(cpy, func(i, j int) bool {
			ai := pick(cpy[i])
			aj := pick(cpy[j])
			if ai != aj {
				return ai > aj
			}
			return cpy[i].total > cpy[j].total
		})
		if len(cpy) > limit {
			cpy = cpy[:limit]
		}
		out := make([]bson.M, 0, len(cpy))
		for _, s := range cpy {
			out = append(out, bson.M{"playerId": s.id, "name": s.name, "points": pick(s)})
		}
		return out
	}

	topMvp := toTop(list, 10, func(a agg) int { return a.total })
	topRaiders := toTop(list, 10, func(a agg) int { return a.raid })
	topDefenders := toTop(list, 10, func(a agg) int { return a.def })

	rankingsColl := db.MongoClient.Database("raidx").Collection("rankings")
	_, err = rankingsColl.UpdateOne(ctx,
		bson.M{"eventId": eventID.Hex(), "eventType": eventType},
		bson.M{"$set": bson.M{
			"eventId":      eventID.Hex(),
			"eventType":    eventType,
			"updatedAt":    time.Now(),
			"topMvp":       topMvp,
			"topRaiders":   topRaiders,
			"topDefenders": topDefenders,
		}},
		options.Update().SetUpsert(true),
	)
	return err
}
