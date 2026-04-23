package middleware

import (
	"context"
	"strconv"
	"strings"

	"github.com/allposty/allposty-backend/internal/auth"
	"github.com/allposty/allposty-backend/internal/repository"
	"github.com/allposty/allposty-backend/internal/services"
	"github.com/allposty/allposty-backend/internal/storage"
	"github.com/allposty/allposty-backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	LocalUserID    = "user_id"
	LocalEmail     = "user_email"
	LocalAPIKeyID  = "api_key_id"
	LocalAPIScopes = "api_scopes"
)

// JWT returns a Fiber middleware that validates the Authorization: Bearer token.
// It accepts both JWT access tokens and allposty_* API keys.
func JWT(
	secret string,
	apiKeySvc *services.APIKeyService,
	userRepo *repository.UserRepository,
	rl *storage.RateLimiter,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return response.Unauthorized(c)
		}
		token := strings.TrimPrefix(header, "Bearer ")

		if strings.HasPrefix(token, "allposty_") {
			return authenticateAPIKey(c, token, apiKeySvc, userRepo, rl)
		}
		return authenticateJWT(c, token, secret)
	}
}

func authenticateJWT(c *fiber.Ctx, token, secret string) error {
	claims, err := auth.ParseToken(token, secret)
	if err != nil {
		return response.Unauthorized(c)
	}
	c.Locals(LocalUserID, claims.UserID)
	c.Locals(LocalEmail, claims.Email)
	return c.Next()
}

func authenticateAPIKey(
	c *fiber.Ctx,
	plain string,
	apiKeySvc *services.APIKeyService,
	userRepo *repository.UserRepository,
	rl *storage.RateLimiter,
) error {
	key, err := apiKeySvc.Authenticate(plain)
	if err != nil {
		return response.Unauthorized(c)
	}

	// Fetch user to determine plan tier for rate limit.
	user, err := userRepo.FindByID(key.UserID)
	if err != nil {
		return response.Unauthorized(c)
	}

	limit := services.RateLimitPerPlan(user.PlanTier)
	allowed, count, err := rl.Allow(context.Background(), key.ID.String(), limit)
	if err != nil {
		return response.InternalError(c)
	}
	remaining := int64(limit) - count
	if remaining < 0 {
		remaining = 0
	}
	c.Set("X-RateLimit-Limit", itoa(limit))
	c.Set("X-RateLimit-Remaining", itoa(int(remaining)))
	if !allowed {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": "rate limit exceeded",
		})
	}

	c.Locals(LocalUserID, key.UserID)
	c.Locals(LocalAPIKeyID, key.ID)
	c.Locals(LocalAPIScopes, []string(key.Scopes))
	return c.Next()
}

// UserIDFromCtx extracts the authenticated user's UUID from Fiber locals.
func UserIDFromCtx(c *fiber.Ctx) (uuid.UUID, bool) {
	id, ok := c.Locals(LocalUserID).(uuid.UUID)
	return id, ok
}

// APIKeyIDFromCtx returns the API key ID if the request was authenticated via API key.
func APIKeyIDFromCtx(c *fiber.Ctx) (uuid.UUID, bool) {
	id, ok := c.Locals(LocalAPIKeyID).(uuid.UUID)
	return id, ok
}

// ScopesFromCtx returns scopes when authenticated via API key (nil for JWT auth).
func ScopesFromCtx(c *fiber.Ctx) []string {
	s, _ := c.Locals(LocalAPIScopes).([]string)
	return s
}

func itoa(n int) string { return strconv.Itoa(n) }
