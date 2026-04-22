package auth

import (
	"errors"

	"github.com/allposty/allposty-backend/internal/middleware"
	"github.com/allposty/allposty-backend/internal/repository"
	"github.com/allposty/allposty-backend/internal/services"
	"github.com/allposty/allposty-backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	auth  *services.AuthService
	users *repository.UserRepository
}

func NewHandler(auth *services.AuthService, users *repository.UserRepository) *Handler {
	return &Handler{auth: auth, users: users}
}

// Register POST /api/v1/auth/register
func (h *Handler) Register(c *fiber.Ctx) error {
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if body.Name == "" || body.Email == "" || len(body.Password) < 8 {
		return response.BadRequest(c, "name, email, and password (min 8 chars) are required")
	}

	user, tokens, err := h.auth.Register(services.RegisterInput{
		Name:     body.Name,
		Email:    body.Email,
		Password: body.Password,
	})
	if err != nil {
		if errors.Is(err, services.ErrEmailTaken) {
			return response.Conflict(c, "email already registered")
		}
		return response.InternalError(c)
	}

	return response.Created(c, fiber.Map{
		"user":   user,
		"tokens": tokens,
	})
}

// Login POST /api/v1/auth/login
func (h *Handler) Login(c *fiber.Ctx) error {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	user, tokens, err := h.auth.Login(services.LoginInput{
		Email:    body.Email,
		Password: body.Password,
	})
	if err != nil {
		if errors.Is(err, services.ErrInvalidPassword) {
			return response.Unauthorized(c)
		}
		return response.InternalError(c)
	}

	return response.OK(c, fiber.Map{
		"user":   user,
		"tokens": tokens,
	})
}

// Refresh POST /api/v1/auth/refresh
func (h *Handler) Refresh(c *fiber.Ctx) error {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&body); err != nil || body.RefreshToken == "" {
		return response.BadRequest(c, "refresh_token is required")
	}

	tokens, err := h.auth.Refresh(body.RefreshToken)
	if err != nil {
		return response.Unauthorized(c)
	}

	return response.OK(c, tokens)
}

// Logout POST /api/v1/auth/logout
func (h *Handler) Logout(c *fiber.Ctx) error {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&body); err == nil && body.RefreshToken != "" {
		_ = h.auth.Logout(body.RefreshToken)
	}
	return response.NoContent(c)
}

// Me GET /api/v1/auth/me
func (h *Handler) Me(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}
	user, err := h.users.FindByID(userID)
	if err != nil {
		return response.NotFound(c, "user")
	}
	return response.OK(c, user)
}
