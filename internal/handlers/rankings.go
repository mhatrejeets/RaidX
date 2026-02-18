package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// GetEventRankingsHandler returns stored rankings for a tournament/championship
func GetEventRankingsHandler(c *fiber.Ctx) error {
	eventType := strings.ToLower(c.Params("type"))
	if eventType != "tournament" && eventType != "championship" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid event type"})
	}
	idParam := c.Params("id")
	if _, err := primitive.ObjectIDFromHex(idParam); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid event id"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rankingsColl := db.MongoClient.Database("raidx").Collection("rankings")
	var doc bson.M
	candidateEventIDs := []string{idParam}
	if resolvedEventID, ok := resolveEventIDForRanking(ctx, eventType, idParam); ok && resolvedEventID != "" && resolvedEventID != idParam {
		candidateEventIDs = append(candidateEventIDs, resolvedEventID)
	}

	if err := rankingsColl.FindOne(ctx, bson.M{"eventId": bson.M{"$in": candidateEventIDs}, "eventType": eventType}).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Rankings not found"})
		}
		logrus.Error("Error:", "GetEventRankingsHandler:", " Failed to fetch rankings: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch rankings"})
	}

	return c.JSON(doc)
}

func resolveEventIDForRanking(ctx context.Context, eventType, idParam string) (string, bool) {
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return "", false
	}

	if eventType == "tournament" {
		var tournament struct {
			EventID primitive.ObjectID `bson:"eventId"`
		}
		if err := db.TournamentsCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&tournament); err == nil {
			return tournament.EventID.Hex(), true
		}
	}

	if eventType == "championship" {
		var championship struct {
			EventID primitive.ObjectID `bson:"eventId"`
		}
		if err := db.ChampionshipsCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&championship); err == nil {
			return championship.EventID.Hex(), true
		}
	}

	return "", false
}
