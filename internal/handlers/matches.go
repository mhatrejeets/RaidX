package handlers

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/models"
	"github.com/mhatrejeets/RaidX/internal/redisImpl"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetAllMatches(c *fiber.Ctx) error {
	matchesCol := db.MongoClient.Database("raidx").Collection("matches")

	cursor, err := matchesCol.Find(context.TODO(), bson.M{})
	if err != nil {
		logrus.Error("Error:", "GetAllMatches:", " DB find error: %v", err)
		return c.Status(500).SendString("DB find error: " + err.Error())
	}
	defer cursor.Close(context.TODO())

	var matches []models.Match
	if err := cursor.All(context.TODO(), &matches); err != nil {
		logrus.Error("Error:", "GetAllMatches:", " Cursor decode error: %v", err)
		return c.Status(500).SendString("Cursor decode error: " + err.Error())
	}

	return c.Render("matches", fiber.Map{
		"Matches": matches,
	})
}

type PlayerWithID struct {
	ID   string
	Stat models.PlayerStat
}

func GetMatchByID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		logrus.Warn("Warning:", "GetMatchByID:", " Invalid match ID: %v", err)
		return c.Status(400).SendString("Invalid match ID")
	}

	matchesCol := db.MongoClient.Database("raidx").Collection("matches")

	var match models.Match
	err = matchesCol.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&match)
	if err != nil {
		logrus.Warn("Warning:", "GetMatchByID:", " Match not found: %v", err)
		return c.Status(404).SendString("Match not found")
	}

	// Convert PlayerStats map to slice
	playerList := []PlayerWithID{}
	for id, stat := range match.Data.PlayerStats {
		playerList = append(playerList, PlayerWithID{
			ID:   id,
			Stat: stat,
		})
	}

	// Now pass both match data and player list
	return c.Render("allmatches", fiber.Map{
		"Match":       match,
		"PlayerStats": playerList,
	})
}

// RaidPayload represents the payload expected from frontend when submitting a raid
type RaidPayload struct {
	RaidType        string   `json:"raidType"`
	RaiderID        string   `json:"raiderId"`
	DefenderIDs     []string `json:"defenderIds"`
	RaidingTeam     string   `json:"raidingTeam"`
	BonusTaken      bool     `json:"bonusTaken"`
	EmptyRaidCounts struct {
		TeamA int `json:"teamA"`
		TeamB int `json:"teamB"`
	} `json:"emptyRaidCounts"`
}

// ProcessRaidResult handles raid outcomes and updates scores/state in Redis and broadcasts updates
func ProcessRaidResult(c *fiber.Ctx) error {
	var raidData RaidPayload
	if err := c.BodyParser(&raidData); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request data"})
	}

	// Read current match state from Redis
	var currentMatch models.EnhancedStatsMessage
	if err := redisImpl.GetRedisKey("gameStats", &currentMatch); err != nil {
		logrus.Error("Error: ProcessRaidResult: Failed to read gameStats from Redis:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to read game state"})
	}

	// Process based on raid type
	switch raidData.RaidType {
	case "successful":
		processSuccessfulRaid(&currentMatch, raidData)
	case "defense":
		processDefenseSuccess(&currentMatch, raidData)
	case "empty":
		processEmptyRaid(&currentMatch, raidData)
	default:
		return c.Status(400).JSON(fiber.Map{"error": "unknown raid type"})
	}

	// Increment raid number after processing
	currentMatch.Data.RaidNumber++

	// Save back to Redis
	if err := redisImpl.SetRedisKey("gameStats", currentMatch); err != nil {
		logrus.Error("Error: ProcessRaidResult: Failed to save gameStats to Redis:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to persist game state"})
	}

	// Broadcast to viewers
	BroadcastToViewers(currentMatch)

	// respond with updated state
	// Use a simple wrapper with Data field to match frontend expectation
	resp := map[string]interface{}{"data": currentMatch.Data}
	return c.JSON(resp)
}

// validateRaidPayload performs server-side validation of incoming raid data
func validateRaidPayload(raid RaidPayload, match *models.EnhancedStatsMessage) error {
	// raid type
	if raid.RaidType != "successful" && raid.RaidType != "defense" && raid.RaidType != "empty" {
		return fmt.Errorf("invalid raidType: %s", raid.RaidType)
	}

	// raider exists
	if raid.RaiderID == "" {
		return fmt.Errorf("missing raiderId")
	}
	raider, ok := match.Data.PlayerStats[raid.RaiderID]
	if !ok {
		return fmt.Errorf("raider not found: %s", raid.RaiderID)
	}
	if raider.Status != "in" {
		return fmt.Errorf("raider is not active: %s", raid.RaiderID)
	}

	// raidingTeam
	if raid.RaidingTeam != "A" && raid.RaidingTeam != "B" {
		return fmt.Errorf("invalid raidingTeam: %s", raid.RaidingTeam)
	}

	// Verify alternating raid rule
	expectedRaidingTeam := "A"
	if match.Data.RaidNumber%2 == 0 {
		expectedRaidingTeam = "B"
	}
	if raid.RaidingTeam != expectedRaidingTeam {
		return fmt.Errorf("incorrect raiding team. Expected team %s to raid", expectedRaidingTeam)
	}

	// defenders validation for types that require defenders
	if raid.RaidType == "successful" || raid.RaidType == "defense" {
		if len(raid.DefenderIDs) == 0 {
			return fmt.Errorf("defenderIds required for raidType %s", raid.RaidType)
		}
		for _, defID := range raid.DefenderIDs {
			if defID == raid.RaiderID {
				return fmt.Errorf("defenderId equals raiderId: %s", defID)
			}
			d, ok := match.Data.PlayerStats[defID]
			if !ok {
				return fmt.Errorf("defender not found: %s", defID)
			}
			if d.Status != "in" {
				return fmt.Errorf("defender is not active: %s", defID)
			}
		}
	}

	// empty raid counts non-negative
	if raid.EmptyRaidCounts.TeamA < 0 || raid.EmptyRaidCounts.TeamB < 0 {
		return fmt.Errorf("emptyRaidCounts must be non-negative")
	}

	return nil
}

func processSuccessfulRaid(match *models.EnhancedStatsMessage, raid RaidPayload) {
	raidPoints := len(raid.DefenderIDs)

	// update team score
	if raid.RaidingTeam == "A" {
		match.Data.TeamA.Score += raidPoints
		if raid.BonusTaken {
			match.Data.TeamA.Score++
		}
	} else {
		match.Data.TeamB.Score += raidPoints
		if raid.BonusTaken {
			match.Data.TeamB.Score++
		}
	}

	// update raider stats
	raiderStat := match.Data.PlayerStats[raid.RaiderID]
	raiderStat.RaidPoints += raidPoints
	raiderStat.TotalPoints += raidPoints
	if raid.BonusTaken {
		raiderStat.RaidPoints++
		raiderStat.TotalPoints++
	}
	match.Data.PlayerStats[raid.RaiderID] = raiderStat

	// mark defenders out and keep their stats
	for _, defID := range raid.DefenderIDs {
		d := match.Data.PlayerStats[defID]
		d.Status = "out"
		match.Data.PlayerStats[defID] = d
	}

	// revival: for each point gained by raiding team, revive one out player from raiding team (if any)
	pointsGained := raidPoints + boolToInt(raid.BonusTaken)
	if raid.RaidingTeam == "A" {
		revivePlayersByIDs(match, match.Data.TeamAPlayerIDs, pointsGained)
	} else {
		revivePlayersByIDs(match, match.Data.TeamBPlayerIDs, pointsGained)
	}

	// record last raid and increment raid number
	match.Data.RaidDetails = models.RaidDetails{
		Type:         "raidSuccess",
		Raider:       raiderStat.Name,
		PointsGained: pointsGained,
		BonusTaken:   raid.BonusTaken,
	}
	match.Data.RaidNumber++
}

func processDefenseSuccess(match *models.EnhancedStatsMessage, raid RaidPayload) {
	// Determine defending team
	defendingTeam := "A"
	if raid.RaidingTeam == "A" {
		defendingTeam = "B"
	}

	// Simple super tackle check based on provided defenderIDs length
	points := 1
	if len(raid.DefenderIDs) <= 3 {
		points = 2
	}

	// award points to defending team
	if defendingTeam == "A" {
		match.Data.TeamA.Score += points
	} else {
		match.Data.TeamB.Score += points
	}

	// mark raider out
	r := match.Data.PlayerStats[raid.RaiderID]
	r.Status = "out"
	match.Data.PlayerStats[raid.RaiderID] = r

	// update defender stats
	for _, defID := range raid.DefenderIDs {
		d := match.Data.PlayerStats[defID]
		d.DefencePoints++
		d.TotalPoints++
		match.Data.PlayerStats[defID] = d
	}

	match.Data.RaidDetails = models.RaidDetails{
		Type:         "defenseSuccess",
		Raider:       r.Name,
		Defenders:    getDefenderNames(match, raid.DefenderIDs),
		PointsGained: points,
		SuperTackle:  points > 1,
	}
	// revival: defenders' team gets revived players equal to points
	if defendingTeam == "A" {
		revivePlayersByIDs(match, match.Data.TeamAPlayerIDs, points)
	} else {
		revivePlayersByIDs(match, match.Data.TeamBPlayerIDs, points)
	}
	match.Data.RaidNumber++
}

func processEmptyRaid(match *models.EnhancedStatsMessage, raid RaidPayload) {
	r := match.Data.PlayerStats[raid.RaiderID]

	if raid.BonusTaken {
		if raid.RaidingTeam == "A" {
			match.Data.TeamA.Score++
		} else {
			match.Data.TeamB.Score++
		}
		r.RaidPoints++
		r.TotalPoints++
	}

	// do-or-die handling
	emptyCount := 0
	if raid.RaidingTeam == "A" {
		emptyCount = raid.EmptyRaidCounts.TeamA
	} else {
		emptyCount = raid.EmptyRaidCounts.TeamB
	}

	if emptyCount >= 3 {
		// failure: raider out, opponent gets 1 point
		r.Status = "out"
		if raid.RaidingTeam == "A" {
			match.Data.TeamB.Score++
		} else {
			match.Data.TeamA.Score++
		}
	}

	match.Data.PlayerStats[raid.RaiderID] = r
	match.Data.RaidDetails = models.RaidDetails{
		Type:       ternaryString(emptyCount >= 3, "doOrDieRaid", "emptyRaid"),
		Raider:     r.Name,
		BonusTaken: raid.BonusTaken,
	}
	// If bonusTaken gave a point, revive for the raiding team
	if raid.BonusTaken {
		if raid.RaidingTeam == "A" {
			revivePlayersByIDs(match, match.Data.TeamAPlayerIDs, 1)
		} else {
			revivePlayersByIDs(match, match.Data.TeamBPlayerIDs, 1)
		}
	}
	// If do-or-die resulted in raider out, defending team gets 1 point and revival
	if emptyCount >= 3 {
		defending := "A"
		if raid.RaidingTeam == "A" {
			defending = "B"
		}
		if defending == "A" {
			revivePlayersByIDs(match, match.Data.TeamAPlayerIDs, 1)
		} else {
			revivePlayersByIDs(match, match.Data.TeamBPlayerIDs, 1)
		}
	}
	match.Data.RaidNumber++
}

// revivePlayersByIDs revives up to count players from the given playerID list by
// setting their status to "in" in match.Data.PlayerStats. It revives the earliest
// out players found in the provided order.
func revivePlayersByIDs(match *models.EnhancedStatsMessage, playerIDs []string, count int) {
	if count <= 0 || playerIDs == nil {
		return
	}
	revived := 0
	for _, pid := range playerIDs {
		if revived >= count {
			break
		}
		p, ok := match.Data.PlayerStats[pid]
		if !ok {
			continue
		}
		if p.Status == "out" {
			p.Status = "in"
			match.Data.PlayerStats[pid] = p
			revived++
		}
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func getDefenderNames(match *models.EnhancedStatsMessage, defenderIDs []string) []string {
	names := make([]string, len(defenderIDs))
	for i, id := range defenderIDs {
		names[i] = match.Data.PlayerStats[id].Name
	}
	return names
}

func ternaryString(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
