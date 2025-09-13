package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Player struct {
	ID          primitive.ObjectID `bson:"_id"`
	FullName    string             `bson:"fullName"`
	TotalPoints int                `bson:"totalPoints"`
	Position    string             `bson:"position"`
}

// Handler for GET /createteam/:id
func CreateTeamPage(c *fiber.Ctx) error {
	userID := c.Params("id")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	playerColl := db.MongoClient.Database("raidx").Collection("players")
	cursor, err := playerColl.Find(ctx, bson.M{})
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Error fetching players")
	}

	var players []Player
	if err := cursor.All(ctx, &players); err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Error parsing players")
	}

	return c.Render("createteam", fiber.Map{
		"Players": players,
		"UserID":  userID,
	})
}

// Handler for POST /createteam/:id
func SubmitTeam(c *fiber.Ctx) error {
	type TeamRequest struct {
		TeamName string   `json:"team_name"`
		Players  []string `json:"players"`
	}

	var team TeamRequest
	if err := c.BodyParser(&team); err != nil {
		return c.Status(http.StatusBadRequest).SendString("Invalid payload")
	}

	// Fetch player names by IDs
	var playerDocs []bson.M
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objIDs := make([]primitive.ObjectID, len(team.Players))
	for i, idStr := range team.Players {
		oid, _ := primitive.ObjectIDFromHex(idStr)
		objIDs[i] = oid
	}

	cursor, err := db.MongoClient.Database("raidx").Collection("players").Find(ctx, bson.M{"_id": bson.M{"$in": objIDs}})
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Failed to fetch players")
	}
	if err := cursor.All(ctx, &playerDocs); err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Failed to parse players")
	}

	// Prepare team document
	var teamPlayers []bson.M
	for _, doc := range playerDocs {
		teamPlayers = append(teamPlayers, bson.M{
			"id":   doc["_id"],
			"name": doc["fullName"],
		})
	}

	_, err = db.MongoClient.Database("raidx").Collection("teams").InsertOne(ctx, bson.M{
		"team_name": team.TeamName,
		"players":   teamPlayers,
	})
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Failed to save team")
	}

	return c.Redirect("/matchestype/" + c.Params("id"))
}
