package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Session struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	SessionID         string             `bson:"session_id"`
	UserID            string             `bson:"user_id"`
	JWTToken          string             `bson:"jwt_token"`
	LoginTime         time.Time          `bson:"login_time"`
	ExpiryTime        time.Time          `bson:"expiry_time"`
	RefreshToken      string             `bson:"refresh_token"`
	RefreshExpiryTime time.Time          `bson:"refresh_expiry_time"`
	Active            bool               `bson:"active"`
	// Device tracking for multi-device sessions
	DeviceID          string             `bson:"device_id"`           // Unique device fingerprint
	UserAgent         string             `bson:"user_agent"`          // Browser/client info
	IPAddress         string             `bson:"ip_address"`          // Client IP
	CreatedAt         time.Time          `bson:"created_at"`          // When session was created
	LastUsedAt        time.Time          `bson:"last_used_at"`        // When token was last refreshed
}
