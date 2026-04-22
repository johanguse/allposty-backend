package posts

import (
	"errors"
	"time"

	"github.com/allposty/allposty-backend/internal/middleware"
	"github.com/allposty/allposty-backend/internal/models"
	"github.com/allposty/allposty-backend/internal/services"
	"github.com/allposty/allposty-backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	posts *services.PostService
}

func NewHandler(posts *services.PostService) *Handler {
	return &Handler{posts: posts}
}

// CreatePost POST /api/v1/posts
func (h *Handler) CreatePost(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}

	var body struct {
		WorkspaceID      string    `json:"workspace_id"`
		Caption          string    `json:"caption"`
		MediaURLs        []string  `json:"media_urls"`
		SocialAccountIDs []string  `json:"social_account_ids"`
		ScheduledAt      *time.Time `json:"scheduled_at"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "invalid body")
	}

	workspaceID, err := uuid.Parse(body.WorkspaceID)
	if err != nil {
		return response.BadRequest(c, "invalid workspace_id")
	}

	var accountIDs []uuid.UUID
	for _, idStr := range body.SocialAccountIDs {
		id, err := uuid.Parse(idStr)
		if err == nil {
			accountIDs = append(accountIDs, id)
		}
	}

	post, err := h.posts.CreatePost(userID, services.CreatePostInput{
		WorkspaceID:      workspaceID,
		Caption:          body.Caption,
		MediaURLs:        body.MediaURLs,
		SocialAccountIDs: accountIDs,
		ScheduledAt:      body.ScheduledAt,
	})
	if err != nil {
		if errors.Is(err, services.ErrForbidden) {
			return response.Forbidden(c)
		}
		return response.InternalError(c)
	}
	return response.Created(c, post)
}

// ListPosts GET /api/v1/posts?workspace_id=...&status=...
func (h *Handler) ListPosts(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}

	workspaceID, err := uuid.Parse(c.Query("workspace_id"))
	if err != nil {
		return response.BadRequest(c, "workspace_id required")
	}

	var status *models.PostStatus
	if s := c.Query("status"); s != "" {
		ps := models.PostStatus(s)
		status = &ps
	}

	postList, err := h.posts.ListPosts(userID, workspaceID, status)
	if err != nil {
		if errors.Is(err, services.ErrForbidden) {
			return response.Forbidden(c)
		}
		return response.InternalError(c)
	}
	return response.OK(c, postList)
}

// SchedulePost POST /api/v1/posts/:id/schedule
func (h *Handler) SchedulePost(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}

	postID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "invalid post id")
	}

	var body struct {
		ScheduledAt time.Time `json:"scheduled_at"`
	}
	if err := c.BodyParser(&body); err != nil || body.ScheduledAt.IsZero() {
		return response.BadRequest(c, "scheduled_at is required")
	}
	if body.ScheduledAt.Before(time.Now()) {
		return response.BadRequest(c, "scheduled_at must be in the future")
	}

	post, err := h.posts.SchedulePost(userID, postID, body.ScheduledAt)
	if err != nil {
		if errors.Is(err, services.ErrPostNotFound) {
			return response.NotFound(c, "post")
		}
		if errors.Is(err, services.ErrForbidden) {
			return response.Forbidden(c)
		}
		return response.InternalError(c)
	}
	return response.OK(c, post)
}

// Calendar GET /api/v1/posts/calendar?workspace_id=...&start=...&end=...
func (h *Handler) Calendar(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}

	workspaceID, err := uuid.Parse(c.Query("workspace_id"))
	if err != nil {
		return response.BadRequest(c, "workspace_id required")
	}

	from, err := time.Parse(time.RFC3339, c.Query("start"))
	if err != nil {
		return response.BadRequest(c, "start must be RFC3339 (e.g. 2025-01-01T00:00:00Z)")
	}
	to, err := time.Parse(time.RFC3339, c.Query("end"))
	if err != nil {
		return response.BadRequest(c, "end must be RFC3339")
	}

	posts, err := h.posts.GetCalendar(userID, workspaceID, from, to)
	if err != nil {
		if errors.Is(err, services.ErrForbidden) {
			return response.Forbidden(c)
		}
		return response.InternalError(c)
	}
	return response.OK(c, posts)
}

// DeletePost DELETE /api/v1/posts/:id
func (h *Handler) DeletePost(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}

	postID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "invalid post id")
	}

	if err := h.posts.DeletePost(userID, postID); err != nil {
		if errors.Is(err, services.ErrPostNotFound) {
			return response.NotFound(c, "post")
		}
		if errors.Is(err, services.ErrForbidden) {
			return response.Forbidden(c)
		}
		return response.InternalError(c)
	}
	return response.NoContent(c)
}
