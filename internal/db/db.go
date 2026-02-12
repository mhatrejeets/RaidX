package db

import (
	"context"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MongoClient *mongo.Client

// Tournament collections
var TournamentsCollection *mongo.Collection
var FixturesCollection *mongo.Collection
var PointsTableCollection *mongo.Collection

// Other collections
var TeamsCollection *mongo.Collection
var EventsCollection *mongo.Collection
var InvitationsCollection *mongo.Collection

func InitDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use MongoDB URI from environment variable
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017/raidx" // fallback default
	}
	clientOptions := options.Client().ApplyURI(uri)

	var err error
	MongoClient, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		logrus.Error("Error:", "InitDB: ", " Failed to connect to MongoDB: %v", err)
	}

	// Ping to ensure connection
	err = MongoClient.Ping(ctx, nil)
	if err != nil {
		logrus.Error("Error:", "InitDB: ", " Failed to ping MongoDB: %v", err)
	}

	logrus.Info("Info:", "InitDB: ", " ✅ Connected to MongoDB")

	// Initialize collection references
	raidxDB := MongoClient.Database("raidx")
	TournamentsCollection = raidxDB.Collection("tournaments")
	FixturesCollection = raidxDB.Collection("fixtures")
	PointsTableCollection = raidxDB.Collection("points_table")
	TeamsCollection = raidxDB.Collection("rbac_teams")
	EventsCollection = raidxDB.Collection("events")
	InvitationsCollection = raidxDB.Collection("invitations")

	// TODO: Enable this when reaching production scale for automatic session cleanup
	// This creates a TTL index on the sessions collection to auto-delete expired refresh tokens
	/*
		createSessionTTLIndex(ctx)
	*/
}

func CloseDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := MongoClient.Disconnect(ctx); err != nil {
		logrus.Error("Error:", "CloseDB: ", " Error disconnecting MongoDB: %v", err)
	}
	logrus.Info("Info:", "CloseDB: ", " MongoDB connection closed")
}

// createSessionTTLIndex sets up automatic deletion of expired sessions
// Called during InitDB when production scale is reached
// Uncomment in InitDB() when needed
/*
func createSessionTTLIndex(ctx context.Context) {
	db := MongoClient.Database("raidx")
	sessions := db.Collection("sessions")

	indexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "refresh_expiry_time", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	}

	_, err := sessions.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		logrus.Error("Error:", "createSessionTTLIndex: ", " Failed to create TTL index: %v", err)
	} else {
		logrus.Info("Info:", "createSessionTTLIndex: ", " ✅ TTL index created on sessions collection")
	}
}
*/
