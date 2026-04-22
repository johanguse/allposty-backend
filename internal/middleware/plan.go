package middleware

// plan.go — Server-side plan limit enforcement.
//
// Applied as route-level middleware on specific endpoints:
//   - RequireAI          → POST /ai/caption
//   - RequireWorkspaceSlot → POST /orgs/:org_id/workspaces
//   - RequireSocialSlot  → GET  /social/connect/:platform?workspace_id=...
//
// All checks read the user's plan_tier from the DB (not JWT) so upgrades
// are reflected immediately without requiring a re-login.

import (
	"github.com/allposty/allposty-backend/internal/repository"
	"github.com/allposty/allposty-backend/internal/services"
	"github.com/allposty/allposty-backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// RequireAI rejects requests from users whose plan does not include AI features.
func RequireAI(users *repository.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := UserIDFromCtx(c)
		if !ok {
			return response.Unauthorized(c)
		}
		user, err := users.FindByID(userID)
		if err != nil {
			return response.Unauthorized(c)
		}
		limits, ok := services.PlanLimits[user.PlanTier]
		if !ok || !limits.AIEnabled {
			return response.PaymentRequired(c, "AI features require a Pro or Agency plan")
		}
		return c.Next()
	}
}

// RequireWorkspaceSlot rejects workspace creation if the user has reached their plan limit.
// It counts all workspaces across all orgs the user owns.
func RequireWorkspaceSlot(users *repository.UserRepository, orgs *repository.OrgRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := UserIDFromCtx(c)
		if !ok {
			return response.Unauthorized(c)
		}
		user, err := users.FindByID(userID)
		if err != nil {
			return response.Unauthorized(c)
		}
		limits, ok := services.PlanLimits[user.PlanTier]
		if !ok {
			return response.InternalError(c)
		}
		if limits.Workspaces == -1 {
			return c.Next() // unlimited
		}
		count, err := orgs.CountWorkspacesByOwner(userID)
		if err != nil {
			return response.InternalError(c)
		}
		if count >= int64(limits.Workspaces) {
			return response.PaymentRequired(c, "workspace limit reached for your plan")
		}
		return c.Next()
	}
}

// RequireSocialSlot rejects social account connections if the workspace has reached its plan limit.
// workspace_id is read from the query string (same as the Connect handler).
func RequireSocialSlot(users *repository.UserRepository, social *repository.SocialRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := UserIDFromCtx(c)
		if !ok {
			return response.Unauthorized(c)
		}
		user, err := users.FindByID(userID)
		if err != nil {
			return response.Unauthorized(c)
		}
		limits, ok := services.PlanLimits[user.PlanTier]
		if !ok {
			return response.InternalError(c)
		}
		if limits.SocialAccounts == -1 {
			return c.Next() // unlimited
		}
		workspaceID, err := uuid.Parse(c.Query("workspace_id"))
		if err != nil {
			return response.BadRequest(c, "workspace_id query param required")
		}
		count, err := social.CountByWorkspace(workspaceID)
		if err != nil {
			return response.InternalError(c)
		}
		if count >= int64(limits.SocialAccounts) {
			return response.PaymentRequired(c, "social account limit reached for your plan")
		}
		return c.Next()
	}
}
