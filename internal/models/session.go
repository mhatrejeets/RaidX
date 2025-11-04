package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Session struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	SessionID  string             `bson:"session_id"`
	UserID     string             `bson:"user_id"`
	JWTToken   string             `bson:"jwt_token"`
	LoginTime  time.Time          `bson:"login_time"`
	ExpiryTime time.Time          `bson:"expiry_time"`
	Active     bool               `bson:"active"`
}
