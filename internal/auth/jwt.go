
package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"time"
)
// ParseJWT parses and validates a JWT token string
func ParseJWT(tokenStr string) (*jwt.Token, error) {
       return jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
	       return jwtSecret, nil
       })
}

var jwtSecret = []byte("SuperSecretKeyChangeMe")

func GenerateJWT(userID string, role string) (string, error) {
       claims := jwt.MapClaims{
	       "user_id": userID,
	       "role": role,
	       "exp": time.Now().Add(time.Hour * 24).Unix(),
       }
       token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
       return token.SignedString(jwtSecret)
}

func JWTMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenStr := c.Get("Authorization")
		if tokenStr == "" {
			tokenStr = c.Query("token")
		}
		if tokenStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing token"})
		}
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid claims"})
		}
		c.Locals("user_id", claims["user_id"])
		c.Locals("role", claims["role"])
		return c.Next()
	}
}
