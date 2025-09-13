package db

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MongoClient *mongo.Client

func InitDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Hardcoded MongoDB URI
	uri := "mongodb://localhost:27017" // Replace with your MongoDB URI

	clientOptions := options.Client().ApplyURI(uri)

	var err error
	MongoClient, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		logrus.Error("Error:","InitDB: "," Failed to connect to MongoDB: %v", err)
	}

	// Ping to ensure connection
	err = MongoClient.Ping(ctx, nil)
	if err != nil {
		logrus.Error("Error:","InitDB: "," Failed to ping MongoDB: %v", err)
	}

	logrus.Info("Info:","InitDB: "," ✅ Connected to MongoDB")
}

func CloseDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := MongoClient.Disconnect(ctx); err != nil {
		logrus.Error("Error:","CloseDB: "," Error disconnecting MongoDB: %v", err)
	}
	logrus.Info("Info:","CloseDB: "," MongoDB connection closed")
}
