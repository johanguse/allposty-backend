package media

import (
	"errors"

	"github.com/allposty/allposty-backend/internal/middleware"
	"github.com/allposty/allposty-backend/internal/services"
	"github.com/allposty/allposty-backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	media *services.MediaService
}

func NewHandler(media *services.MediaService) *Handler {
	return &Handler{media: media}
}

// Upload POST /api/v1/media?workspace_id=...
// Accepts multipart/form-data with a "file" field.
func (h *Handler) Upload(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}

	workspaceID, err := uuid.Parse(c.Query("workspace_id"))
	if err != nil {
		return response.BadRequest(c, "workspace_id required")
	}

	fh, err := c.FormFile("file")
	if err != nil {
		return response.BadRequest(c, "file field is required")
	}

	var folder *string
	if f := c.FormValue("folder"); f != "" {
		folder = &f
	}

	file, err := h.media.Upload(c.Context(), userID, workspaceID, fh, folder)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrForbidden):
			return response.Forbidden(c)
		case errors.Is(err, services.ErrFileTooLarge):
			return response.BadRequest(c, "file exceeds 100 MB limit")
		default:
			return response.InternalError(c)
		}
	}
	return response.Created(c, file)
}

// List GET /api/v1/media?workspace_id=...&folder=...
func (h *Handler) List(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}

	workspaceID, err := uuid.Parse(c.Query("workspace_id"))
	if err != nil {
		return response.BadRequest(c, "workspace_id required")
	}

	var folder *string
	if f := c.Query("folder"); f != "" {
		folder = &f
	}

	files, err := h.media.List(userID, workspaceID, folder)
	if err != nil {
		if errors.Is(err, services.ErrForbidden) {
			return response.Forbidden(c)
		}
		return response.InternalError(c)
	}
	return response.OK(c, files)
}

// Delete DELETE /api/v1/media/:id
func (h *Handler) Delete(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}

	fileID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "invalid id")
	}

	if err := h.media.Delete(c.Context(), userID, fileID); err != nil {
		switch {
		case errors.Is(err, services.ErrMediaNotFound):
			return response.NotFound(c, "media file")
		case errors.Is(err, services.ErrForbidden):
			return response.Forbidden(c)
		default:
			return response.InternalError(c)
		}
	}
	return response.NoContent(c)
}
