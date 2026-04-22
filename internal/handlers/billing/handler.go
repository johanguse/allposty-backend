package billing

import (
	"errors"
	"io"

	"github.com/allposty/allposty-backend/internal/middleware"
	"github.com/allposty/allposty-backend/internal/services"
	"github.com/allposty/allposty-backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	billing     *services.BillingService
	frontendURL string
}

func NewHandler(billing *services.BillingService, frontendURL string) *Handler {
	return &Handler{billing: billing, frontendURL: frontendURL}
}

// CreateCheckout POST /api/v1/billing/checkout
func (h *Handler) CreateCheckout(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}

	var body struct {
		OrgID string `json:"org_id"`
		Tier  string `json:"tier"` // pro | agency
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "invalid body")
	}

	orgID, err := uuid.Parse(body.OrgID)
	if err != nil {
		return response.BadRequest(c, "invalid org_id")
	}
	if body.Tier != "pro" && body.Tier != "agency" {
		return response.BadRequest(c, "tier must be pro or agency")
	}

	successURL := h.frontendURL + "/settings/billing?success=true"
	cancelURL := h.frontendURL + "/settings/billing?canceled=true"

	checkoutURL, err := h.billing.CreateCheckoutSession(userID, orgID, body.Tier, successURL, cancelURL)
	if err != nil {
		if errors.Is(err, services.ErrForbidden) {
			return response.Forbidden(c)
		}
		if errors.Is(err, services.ErrOrgNotFound) {
			return response.NotFound(c, "organization")
		}
		return response.InternalError(c)
	}
	return response.OK(c, fiber.Map{"url": checkoutURL})
}

// CreatePortal POST /api/v1/billing/portal
func (h *Handler) CreatePortal(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}

	var body struct {
		OrgID string `json:"org_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "invalid body")
	}

	orgID, err := uuid.Parse(body.OrgID)
	if err != nil {
		return response.BadRequest(c, "invalid org_id")
	}

	returnURL := h.frontendURL + "/settings/billing"
	portalURL, err := h.billing.CreatePortalSession(userID, orgID, returnURL)
	if err != nil {
		if errors.Is(err, services.ErrForbidden) {
			return response.Forbidden(c)
		}
		if errors.Is(err, services.ErrSubscriptionNotFound) {
			return response.NotFound(c, "subscription")
		}
		return response.InternalError(c)
	}
	return response.OK(c, fiber.Map{"url": portalURL})
}

// Webhook POST /api/v1/billing/webhook  (public — signed by Stripe)
func (h *Handler) Webhook(c *fiber.Ctx) error {
	payload, err := io.ReadAll(c.Request().BodyStream())
	if err != nil {
		return response.BadRequest(c, "failed to read body")
	}

	signature := c.Get("Stripe-Signature")
	if signature == "" {
		return response.BadRequest(c, "missing stripe signature")
	}

	if err := h.billing.HandleWebhook(payload, signature); err != nil {
		// Log but return 200 — Stripe retries on non-200
		return c.SendStatus(fiber.StatusOK)
	}
	return c.SendStatus(fiber.StatusOK)
}
