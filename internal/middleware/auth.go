package middleware

import (
	"strings"

	"github.com/allposty/allposty-backend/internal/auth"
	"github.com/allposty/allposty-backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	LocalUserID = "user_id"
	LocalEmail  = "user_email"
)

// JWT returns a Fiber middleware that validates the Authorization: Bearer token.
func JWT(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return response.Unauthorized(c)
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := auth.ParseToken(tokenStr, secret)
		if err != nil {
			return response.Unauthorized(c)
		}

		c.Locals(LocalUserID, claims.UserID)
		c.Locals(LocalEmail, claims.Email)
		return c.Next()
	}
}

// UserIDFromCtx extracts the authenticated user's UUID from Fiber locals.
func UserIDFromCtx(c *fiber.Ctx) (uuid.UUID, bool) {
	id, ok := c.Locals(LocalUserID).(uuid.UUID)
	return id, ok
}
