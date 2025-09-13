package handlers

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// RequestsHandler handles the /requests/:id route
func RequestsHandler(c *fiber.Ctx) error {
	// Get the player ID from the URL
	playerID := c.Params("id")

	// Convert the string ID to ObjectId
	objID, err := primitive.ObjectIDFromHex(playerID)
	if err != nil {
		logrus.Warn("Warning:", "RequestsHandler:", " Invalid player ID: %v", err)
		return c.Status(fiber.StatusBadRequest).SendString("Invalid player ID format")
	}

	// Define MongoDB collection
	playerCollection := db.MongoClient.Database("raidx").Collection("players")

	// Query to find the player by ID
	var player struct {
		Requests struct {
			TeamName string `bson:"team-name"`
			TeamID   string `bson:"team-id"`
			Status   string `bson:"status"`
		} `bson:"requests"`
		FullName string `bson:"fullName"`
	}
	err = playerCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&player)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			logrus.Info("Info:", "RequestsHandler:", " Player not found: %v", err)
			return c.Status(fiber.StatusNotFound).SendString("Player not found")
		}
		logrus.Error("Error:", "RequestsHandler:", " Error fetching player data: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error fetching player data")
	}

	// Check if the player has a request with status "Select"
	if player.Requests.Status != "Select" {
		return c.Render("requests", fiber.Map{
			"Message": "No pending requests.",
		})
	}

	// Render the requests.html view
	return c.Render("requests", fiber.Map{
		"TeamName": player.Requests.TeamName,
		"YesURL":   fmt.Sprintf("/requests/%s/accept", playerID),
		"NoURL":    fmt.Sprintf("/requests/%s/reject", playerID),
	})
}

// AcceptRequestHandler handles the logic for accepting a request
func AcceptRequestHandler(c *fiber.Ctx) error {
	// Get the player ID from the URL
	playerID := c.Params("id")

	// Convert the string ID to ObjectId
	objID, err := primitive.ObjectIDFromHex(playerID)
	if err != nil {
		logrus.Warn("Warning:", "AcceptRequestHandler:", " Invalid player ID: %v", err)
		return c.Status(fiber.StatusBadRequest).SendString("Invalid player ID format")
	}

	// Define MongoDB collections
	playerCollection := db.MongoClient.Database("raidx").Collection("players")
	teamCollection := db.MongoClient.Database("raidx").Collection("teams")

	// Fetch the player document
	var player struct {
		Requests struct {
			TeamName string `bson:"team-name"`
			TeamID   string `bson:"team-id"`
		} `bson:"requests"`
		FullName string `bson:"fullName"`
		UserID   string `bson:"userId"`
	}
	err = playerCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&player)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			logrus.Info("Info:", "AcceptRequestHandler:", " Player not found: %v", err)
			return c.Status(fiber.StatusNotFound).SendString("Player not found")
		}
		logrus.Error("Error:", "AcceptRequestHandler:", " Error fetching player data: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error fetching player data")
	}

	// Update the player's "team enrolled" field
	teamEnrolled := bson.M{
		"team_name": player.Requests.TeamName,
		"team_id":   player.Requests.TeamID,
	}
	_, err = playerCollection.UpdateOne(
		context.TODO(),
		bson.M{"_id": objID},
		bson.M{"$set": bson.M{"teams_enrolled": teamEnrolled}},
	)
	if err != nil {
		logrus.Error("Error:", "AcceptRequestHandler:", " Error updating player data: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error updating player data")
	}

	// Convert TeamID to ObjectId
	teamObjID, err := primitive.ObjectIDFromHex(player.Requests.TeamID)
	if err != nil {
		logrus.Warn("Warning:", "AcceptRequestHandler:", " Invalid team ID: %v", err)
		return c.Status(fiber.StatusBadRequest).SendString("Invalid team ID format")
	}

	// Add the player to the team's players array
	playerInfo := bson.M{
		"name": player.FullName,
		"id":   player.UserID,
	}
	_, err = teamCollection.UpdateOne(
		context.TODO(),
		bson.M{"_id": teamObjID},
		bson.M{"$push": bson.M{"players": playerInfo}},
	)
	if err != nil {
		logrus.Error("Error:", "AcceptRequestHandler:", " Error updating team data: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error updating team data")
	}

	// Send a success response
	return c.SendString("Request accepted successfully")
}

// RejectRequestHandler handles the logic for rejecting a request
func RejectRequestHandler(c *fiber.Ctx) error {
	// Get the player ID from the URL
	playerID := c.Params("id")

	// Convert the string ID to ObjectId
	objID, err := primitive.ObjectIDFromHex(playerID)
	if err != nil {
		logrus.Warn("Warning:", "RejectRequestHandler:", " Invalid player ID: %v", err)
		return c.Status(fiber.StatusBadRequest).SendString("Invalid player ID format")
	}

	// Define MongoDB collection
	playerCollection := db.MongoClient.Database("raidx").Collection("players")

	// Update the player's request status to "Rejected"
	_, err = playerCollection.UpdateOne(
		context.TODO(),
		bson.M{"_id": objID},
		bson.M{"$set": bson.M{"requests.status": "Rejected"}},
	)
	if err != nil {
		logrus.Error("Error:", "RejectRequestHandler:", " Error updating request status: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error updating request status")
	}

	// Send a success response
	return c.SendString("Request rejected successfully")
}
