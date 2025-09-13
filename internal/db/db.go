package db

import (
    "context"
    "log"
    "time"

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
        log.Fatalf("Failed to connect to MongoDB: %v", err)
    }

    // Ping to ensure connection
    err = MongoClient.Ping(ctx, nil)
    if err != nil {
        log.Fatalf("Failed to ping MongoDB: %v", err)
    }

    log.Println("✅ Connected to MongoDB")
}

func CloseDB() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := MongoClient.Disconnect(ctx); err != nil {
        log.Fatalf("Error disconnecting MongoDB: %v", err)
    }
    log.Println("🛑 MongoDB connection closed")
}
