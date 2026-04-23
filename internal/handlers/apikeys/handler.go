package apikeys

import (
	"time"

	"github.com/allposty/allposty-backend/internal/middleware"
	"github.com/allposty/allposty-backend/internal/services"
	"github.com/allposty/allposty-backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Handler struct {
	svc *services.APIKeyService
}

func NewHandler(svc *services.APIKeyService) *Handler {
	return &Handler{svc: svc}
}

// Create POST /api/v1/api-keys
func (h *Handler) Create(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}

	var body struct {
		Name      string    `json:"name"`
		Scopes    []string  `json:"scopes"`
		ExpiresAt *time.Time `json:"expires_at"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	if body.Name == "" {
		return response.BadRequest(c, "name required")
	}
	for _, s := range body.Scopes {
		if !services.ValidScopes[s] {
			return response.BadRequest(c, "unknown scope: "+s)
		}
	}

	result, err := h.svc.Create(services.CreateAPIKeyInput{
		UserID:    userID,
		Name:      body.Name,
		Scopes:    body.Scopes,
		ExpiresAt: body.ExpiresAt,
	})
	if err != nil {
		return response.InternalError(c)
	}

	// Return the plaintext key in this response only.
	return response.Created(c, fiber.Map{
		"key":        result.Plain,
		"id":         result.Key.ID,
		"name":       result.Key.Name,
		"prefix":     result.Key.Prefix,
		"scopes":     result.Key.Scopes,
		"expires_at": result.Key.ExpiresAt,
		"created_at": result.Key.CreatedAt,
	})
}

// List GET /api/v1/api-keys
func (h *Handler) List(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}

	keys, err := h.svc.List(userID)
	if err != nil {
		return response.InternalError(c)
	}
	return response.OK(c, keys)
}

// Revoke DELETE /api/v1/api-keys/:id
func (h *Handler) Revoke(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}

	keyID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "invalid id")
	}

	if err := h.svc.Revoke(keyID, userID); err != nil {
		switch err {
		case services.ErrAPIKeyNotFound:
			return response.NotFound(c, "api key")
		case services.ErrForbidden:
			return response.Forbidden(c)
		default:
			return response.InternalError(c)
		}
	}
	return response.NoContent(c)
}

// ValidScopes GET /api/v1/api-keys/scopes
func (h *Handler) Scopes(c *fiber.Ctx) error {
	scopes := make([]string, 0, len(services.ValidScopes))
	for s := range services.ValidScopes {
		scopes = append(scopes, s)
	}
	return response.OK(c, pq.StringArray(scopes))
}
