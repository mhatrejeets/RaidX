package main

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)
// Team structure
type Teamm struct {
	Name  string `json:"name" bson:"name"`
	Score int    `json:"score" bson:"score"`
}

// PlayerStat represents a player’s stats (dynamic keys in MongoDB)
type PlayerStatt struct {
	Name          string `json:"name" bson:"name"`
	RaidPoints    int    `json:"raidPoints" bson:"raidPoints"`
	DefencePoints int    `json:"defencePoints" bson:"defencePoints"`
	TotalPoints   int    `json:"totalPoints" bson:"totalPoints"`
	Status        string `json:"status" bson:"status"`
}

// RaidDetails of the last raid
type RaidDetailss struct {
	Type         string `json:"type" bson:"type"`
	Raider       string `json:"raider" bson:"raider"`
	PointsGained int    `json:"pointsGained" bson:"pointsGained"`
}

// Match struct matching your MongoDB document
type Match struct {
	ID   primitive.ObjectID `json:"id" bson:"_id"`
	Type string             `json:"type" bson:"type"`
	Data struct {
		TeamA       Teamm                         `json:"teamA" bson:"teamA"`
		TeamB       Teamm                         `json:"teamB" bson:"teamB"`
		PlayerStats map[string]PlayerStatt        `json:"playerStats" bson:"playerStats"`
		RaidDetails RaidDetailss                  `json:"raidDetails" bson:"raidDetails"`
	} `json:"data" bson:"data"`
}



func GetAllMatches(c *fiber.Ctx) error {
	matchesCol := Client.Database("raidx").Collection("matches")

	cursor, err := matchesCol.Find(context.TODO(), bson.M{})
	if err != nil {
		return c.Status(500).SendString("DB find error: " + err.Error())
	}
	defer cursor.Close(context.TODO())

	var matches []Match
	if err := cursor.All(context.TODO(), &matches); err != nil {
		return c.Status(500).SendString("Cursor decode error: " + err.Error())
	}

	return c.Render("matches", fiber.Map{
		"Matches": matches,
	})
}

type PlayerWithID struct {
	ID   string
	Stat PlayerStatt
}

func GetMatchByID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(400).SendString("Invalid match ID")
	}

	matchesCol := Client.Database("raidx").Collection("matches")

	var match Match
	err = matchesCol.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&match)
	if err != nil {
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

