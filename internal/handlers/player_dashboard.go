package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type playerTeamResponse struct {
	TeamID      string    `json:"teamId"`
	TeamName    string    `json:"teamName"`
	OwnerID     string    `json:"ownerId"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

type playerEventResponse struct {
	EventID    string `json:"eventId,omitempty"`
	EventType  string `json:"eventType"`
	EventName  string `json:"eventName"`
	Status     string `json:"status,omitempty"`
	MatchCount int    `json:"matchCount"`
}

func GetPlayerTeamsHandler(c *fiber.Ctx) error {
	playerID, err := getUserIDFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	teamColl := db.MongoClient.Database("raidx").Collection("rbac_teams")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := teamColl.Find(ctx, bson.M{"players": playerID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load teams"})
	}
	defer cursor.Close(ctx)

	teams := make([]playerTeamResponse, 0)
	for cursor.Next(ctx) {
		var team models.TeamProfile
		if err := cursor.Decode(&team); err != nil {
			continue
		}
		teams = append(teams, playerTeamResponse{
			TeamID:      team.ID.Hex(),
			TeamName:    team.TeamName,
			OwnerID:     team.OwnerID.Hex(),
			Description: team.Description,
			Status:      team.Status,
			CreatedAt:   team.CreatedAt,
		})
	}

	return c.JSON(teams)
}

func GetPlayerEventsHandler(c *fiber.Ctx) error {
	playerID, err := getUserIDFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}
	playerIDStr := playerID.Hex()

	matchesColl := db.MongoClient.Database("raidx").Collection("matches")
	eventsColl := db.MongoClient.Database("raidx").Collection("events")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	filter := bson.M{fmt.Sprintf("data.playerStats.%s", playerIDStr): bson.M{"$exists": true}}
	projection := bson.M{"event_type": 1, "eventType": 1, "event_id": 1, "eventId": 1, "matchId": 1}
	cursor, err := matchesColl.Find(ctx, filter, options.Find().SetProjection(projection))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load events"})
	}
	defer cursor.Close(ctx)

	type eventAgg struct {
		eventType  string
		eventID    string
		matchCount int
	}
	acc := map[string]*eventAgg{}
	cache := map[string]models.Event{}

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		eventType, _ := doc["event_type"].(string)
		if eventType == "" {
			eventType, _ = doc["eventType"].(string)
		}
		eventID := normalizeIDString(doc["event_id"])
		if eventID == "" {
			eventID = normalizeIDString(doc["eventId"])
		}
		if eventID == "" {
			if matchID, ok := doc["matchId"].(string); ok {
				eventID = matchID
			}
		}
		if eventType == "" {
			eventType = models.EventTypeMatch
		}
		key := fmt.Sprintf("%s:%s", eventType, eventID)
		entry, ok := acc[key]
		if !ok {
			entry = &eventAgg{eventType: eventType, eventID: eventID, matchCount: 0}
			acc[key] = entry
		}
		entry.matchCount += 1

		if eventID != "" {
			if _, ok := cache[eventID]; !ok {
				if oid, err := primitive.ObjectIDFromHex(eventID); err == nil {
					var evt models.Event
					if err := eventsColl.FindOne(ctx, bson.M{"_id": oid}).Decode(&evt); err == nil {
						cache[eventID] = evt
					}
				}
			}
		}
	}

	result := make([]playerEventResponse, 0, len(acc))
	for _, entry := range acc {
		eventName := "Standalone match"
		status := ""
		if entry.eventID != "" {
			if evt, ok := cache[entry.eventID]; ok {
				eventName = evt.EventName
				status = evt.Status
			} else if entry.eventType != models.EventTypeMatch {
				eventName = "Event"
			}
		}
		result = append(result, playerEventResponse{
			EventID:    entry.eventID,
			EventType:  entry.eventType,
			EventName:  eventName,
			Status:     status,
			MatchCount: entry.matchCount,
		})
	}

	return c.JSON(result)
}

func normalizeIDString(value interface{}) string {
	switch t := value.(type) {
	case primitive.ObjectID:
		return t.Hex()
	case string:
		return t
	case bson.M:
		if oid, ok := t["$oid"].(string); ok {
			return oid
		}
	case map[string]interface{}:
		if oid, ok := t["$oid"].(string); ok {
			return oid
		}
	}
	return ""
}
