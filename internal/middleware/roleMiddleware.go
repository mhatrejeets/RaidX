package middleware

import (
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// RoleRequired enforces that the authenticated user has one of the allowed roles.
// Use it after AuthRequired middleware.
func RoleRequired(allowedRoles ...string) fiber.Handler {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, role := range allowedRoles {
		allowed[strings.ToLower(role)] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		roleVal := c.Locals("role")
		roleStr, _ := roleVal.(string)
		roleStr = strings.ToLower(roleStr)

		if _, ok := allowed[roleStr]; !ok {
			if strings.HasPrefix(c.Path(), "/api") {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Insufficient role"})
			}
			returnUrl := c.OriginalURL()
			return c.Redirect("/login?returnUrl=" + url.QueryEscape(returnUrl))
		}

		return c.Next()
	}
}
