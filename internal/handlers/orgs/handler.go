package orgs

import (
	"errors"

	"github.com/allposty/allposty-backend/internal/middleware"
	"github.com/allposty/allposty-backend/internal/services"
	"github.com/allposty/allposty-backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	orgs *services.OrgService
}

func NewHandler(orgs *services.OrgService) *Handler {
	return &Handler{orgs: orgs}
}

// CreateOrg POST /api/v1/orgs
func (h *Handler) CreateOrg(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil || body.Name == "" {
		return response.BadRequest(c, "name is required")
	}

	org, err := h.orgs.CreateOrg(userID, body.Name)
	if err != nil {
		return response.InternalError(c)
	}
	return response.Created(c, org)
}

// ListOrgs GET /api/v1/orgs
func (h *Handler) ListOrgs(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}
	orgs, err := h.orgs.ListOrgs(userID)
	if err != nil {
		return response.InternalError(c)
	}
	return response.OK(c, orgs)
}

// GetOrg GET /api/v1/orgs/:org_id
func (h *Handler) GetOrg(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}
	orgID, err := uuid.Parse(c.Params("org_id"))
	if err != nil {
		return response.BadRequest(c, "invalid org_id")
	}

	org, err := h.orgs.GetOrg(orgID, userID)
	if err != nil {
		if errors.Is(err, services.ErrOrgNotFound) {
			return response.NotFound(c, "organization")
		}
		if errors.Is(err, services.ErrForbidden) {
			return response.Forbidden(c)
		}
		return response.InternalError(c)
	}
	return response.OK(c, org)
}

// CreateWorkspace POST /api/v1/orgs/:org_id/workspaces
func (h *Handler) CreateWorkspace(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}
	orgID, err := uuid.Parse(c.Params("org_id"))
	if err != nil {
		return response.BadRequest(c, "invalid org_id")
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil || body.Name == "" {
		return response.BadRequest(c, "name is required")
	}

	ws, err := h.orgs.CreateWorkspace(orgID, userID, body.Name)
	if err != nil {
		if errors.Is(err, services.ErrForbidden) {
			return response.Forbidden(c)
		}
		return response.InternalError(c)
	}
	return response.Created(c, ws)
}

// ListWorkspaces GET /api/v1/orgs/:org_id/workspaces
func (h *Handler) ListWorkspaces(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}
	orgID, err := uuid.Parse(c.Params("org_id"))
	if err != nil {
		return response.BadRequest(c, "invalid org_id")
	}

	workspaces, err := h.orgs.ListWorkspaces(orgID, userID)
	if err != nil {
		if errors.Is(err, services.ErrForbidden) {
			return response.Forbidden(c)
		}
		return response.InternalError(c)
	}
	return response.OK(c, workspaces)
}
