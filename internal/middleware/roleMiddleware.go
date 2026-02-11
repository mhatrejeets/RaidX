package middleware

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// RoleRequired enforces that the authenticated user has one of the allowed roles.
// Use it after AuthRequired middleware.
func RoleRequired(allowedRoles ...string) fiber.Handler {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, role := range allowedRoles {
		allowed[strings.TrimSpace(strings.ToLower(role))] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		roleVal := c.Locals("role")
		roleStr := ""
		if roleVal != nil {
			roleStr = strings.TrimSpace(strings.ToLower(fmt.Sprint(roleVal)))
		}
		if strings.Contains(c.Path(), "playerselection") {
			for k := range allowed {
				println(" -", k)
			}
		}

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
