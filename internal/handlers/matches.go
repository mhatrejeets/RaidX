package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/models"
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
	Stat models.PlayerStatt
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
