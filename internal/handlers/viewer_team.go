package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetPublicTeamByIDHandler(c *fiber.Ctx) error {
	teamID := c.Params("id")
	teamOID, err := primitive.ObjectIDFromHex(teamID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid team ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	teamsCollection := db.MongoClient.Database("raidx").Collection("rbac_teams")
	playersCollection := db.MongoClient.Database("raidx").Collection("players")

	var team models.TeamProfile
	err = teamsCollection.FindOne(ctx, bson.M{"_id": teamOID}).Decode(&team)
	if err == mongo.ErrNoDocuments {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Team not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch team"})
	}

	ownerName := "Unknown"
	if team.OwnerID != primitive.NilObjectID {
		var owner models.User
		if err := playersCollection.FindOne(ctx, bson.M{"_id": team.OwnerID}).Decode(&owner); err == nil {
			if owner.FullName != "" {
				ownerName = owner.FullName
			}
		}
	}

	playerMap := map[string]bson.M{}
	if len(team.Players) > 0 {
		projection := bson.M{"fullName": 1, "userId": 1, "position": 1}
		cursor, err := playersCollection.Find(
			ctx,
			bson.M{"_id": bson.M{"$in": team.Players}},
			options.Find().SetProjection(projection),
		)
		if err == nil {
			defer cursor.Close(ctx)
			for cursor.Next(ctx) {
				var player bson.M
				if err := cursor.Decode(&player); err != nil {
					continue
				}
				if oid, ok := player["_id"].(primitive.ObjectID); ok {
					playerMap[oid.Hex()] = player
				}
			}
		}
	}

	players := make([]fiber.Map, 0, len(team.Players))
	for _, playerID := range team.Players {
		player, ok := playerMap[playerID.Hex()]
		if !ok {
			continue
		}
		players = append(players, fiber.Map{
			"id":       playerID.Hex(),
			"fullName": player["fullName"],
			"userId":   player["userId"],
			"position": player["position"],
		})
	}

	return c.JSON(fiber.Map{
		"id":          team.ID.Hex(),
		"teamName":    team.TeamName,
		"description": team.Description,
		"status":      team.Status,
		"ownerId":     team.OwnerID.Hex(),
		"ownerName":   ownerName,
		"playerCount": len(players),
		"players":     players,
	})
}
