package ai

import (
	"github.com/allposty/allposty-backend/internal/middleware"
	"github.com/allposty/allposty-backend/internal/services"
	"github.com/allposty/allposty-backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	ai *services.AIService
}

func NewHandler(ai *services.AIService) *Handler {
	return &Handler{ai: ai}
}

// GenerateCaption POST /api/v1/ai/caption
func (h *Handler) GenerateCaption(c *fiber.Ctx) error {
	_, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}

	var body struct {
		Topic    string   `json:"topic"`
		Platform string   `json:"platform"`
		Tone     string   `json:"tone"`
		Keywords []string `json:"keywords"`
		Language string   `json:"language"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	if body.Topic == "" || body.Platform == "" {
		return response.BadRequest(c, "topic and platform are required")
	}

	result, err := h.ai.GenerateCaption(c.Context(), services.CaptionInput{
		Topic:    body.Topic,
		Platform: body.Platform,
		Tone:     body.Tone,
		Keywords: body.Keywords,
		Language: body.Language,
	})
	if err != nil {
		return response.InternalError(c)
	}
	return response.OK(c, result)
}
