package middleware

import (
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

func AuthRequired(c *fiber.Ctx) error {
	tokenStr := c.Get("Authorization")
	if tokenStr == "" {
		tokenStr = c.Query("token")
	}
	if tokenStr == "" {
		// Try HttpOnly cookie
		tokenStr = c.Cookies("token")
	}
	if tokenStr == "" {
		// For non-API routes, redirect to login; keep JSON for API
		if strings.HasPrefix(c.Path(), "/api") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing JWT token"})
		}
		returnUrl := c.OriginalURL()
		return c.Redirect("/login?returnUrl=" + url.QueryEscape(returnUrl))
	}
	if strings.HasPrefix(tokenStr, "Bearer ") {
		tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
	}
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		if strings.HasPrefix(c.Path(), "/api") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired JWT token"})
		}
		returnUrl := c.OriginalURL()
		return c.Redirect("/login?returnUrl=" + url.QueryEscape(returnUrl))
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		if strings.HasPrefix(c.Path(), "/api") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid JWT claims"})
		}
		returnUrl := c.OriginalURL()
		return c.Redirect("/login?returnUrl=" + url.QueryEscape(returnUrl))
	}
	// Check expiry
	exp, ok := claims["exp"].(float64)
	if !ok || int64(exp) < time.Now().Unix() {
		if strings.HasPrefix(c.Path(), "/api") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "JWT expired"})
		}
		returnUrl := c.OriginalURL()
		return c.Redirect("/login?returnUrl=" + url.QueryEscape(returnUrl))
	}
	// Attach user info to context
	c.Locals("user_id", claims["user_id"])
	c.Locals("role", claims["role"])
	c.Locals("session_id", claims["session_id"])
	return c.Next()
}

// For WebSocket handshake
func AuthWebSocket(tokenStr string) (map[string]interface{}, error) {
	if tokenStr == "" {
		return nil, fiber.ErrUnauthorized
	}
	if strings.HasPrefix(tokenStr, "Bearer ") {
		tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
	}
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, fiber.ErrUnauthorized
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fiber.ErrUnauthorized
	}
	exp, ok := claims["exp"].(float64)
	if !ok || int64(exp) < time.Now().Unix() {
		return nil, fiber.ErrUnauthorized
	}
	return claims, nil
}
