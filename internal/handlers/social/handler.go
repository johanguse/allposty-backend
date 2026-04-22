package social

import (
	"context"
	"errors"
	"fmt"

	"github.com/allposty/allposty-backend/internal/middleware"
	"github.com/allposty/allposty-backend/internal/models"
	"github.com/allposty/allposty-backend/internal/providers"
	"github.com/allposty/allposty-backend/internal/repository"
	"github.com/allposty/allposty-backend/internal/services"
	"github.com/allposty/allposty-backend/internal/storage"
	"github.com/allposty/allposty-backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	registry    *providers.Registry
	social      *repository.SocialRepository
	orgs        *services.OrgService
	creds       *storage.CredentialStore
	state       *storage.StateStore
	frontendURL string
}

func NewHandler(
	registry *providers.Registry,
	social *repository.SocialRepository,
	orgs *services.OrgService,
	creds *storage.CredentialStore,
	state *storage.StateStore,
	frontendURL string,
) *Handler {
	return &Handler{
		registry:    registry,
		social:      social,
		orgs:        orgs,
		creds:       creds,
		state:       state,
		frontendURL: frontendURL,
	}
}

// Connect GET /api/v1/social/connect/:platform?workspace_id=...
func (h *Handler) Connect(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}

	platformStr := c.Params("platform")
	workspaceID, err := uuid.Parse(c.Query("workspace_id"))
	if err != nil {
		return response.BadRequest(c, "workspace_id query param required")
	}

	if err := h.orgs.RequireWorkspaceAccess(workspaceID, userID); err != nil {
		return response.Forbidden(c)
	}

	provider, err := h.registry.Get(providers.Platform(platformStr))
	if err != nil {
		return response.NotFound(c, "platform")
	}

	redirectURL := h.callbackURL(c, platformStr)
	state := uuid.New().String()

	statePayload := storage.OAuthStatePayload{
		WorkspaceID: workspaceID.String(),
		Platform:    platformStr,
	}

	// Twitter requires PKCE — generate verifier and store it in the state payload
	var authURL string
	if tp, ok := provider.(*providers.TwitterProvider); ok {
		pkce, err := tp.GeneratePKCE()
		if err != nil {
			return response.InternalError(c)
		}
		statePayload.CodeVerifier = pkce.Verifier
		authURL = tp.AuthURLWithPKCE(redirectURL, state, pkce.Challenge)
	} else {
		authURL = provider.AuthURL(redirectURL, state)
	}

	if err := h.state.Save(c.Context(), state, statePayload); err != nil {
		return response.InternalError(c)
	}

	return c.Redirect(authURL)
}

// Callback GET /api/v1/social/callback/:platform
func (h *Handler) Callback(c *fiber.Ctx) error {
	platformStr := c.Params("platform")
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		return response.BadRequest(c, "missing code or state")
	}

	// Pop is one-time use — prevents state replay attacks
	statePayload, err := h.state.Pop(c.Context(), state)
	if err != nil {
		return response.BadRequest(c, "invalid or expired state")
	}

	workspaceID, err := uuid.Parse(statePayload.WorkspaceID)
	if err != nil {
		return response.BadRequest(c, "invalid workspace in state")
	}

	provider, err := h.registry.Get(providers.Platform(platformStr))
	if err != nil {
		return response.NotFound(c, "platform")
	}

	redirectURL := h.callbackURL(c, platformStr)
	ctx := context.Background()

	// Twitter uses PKCE — exchange with verifier
	var oauthCreds *providers.OAuthCredentials
	if tp, ok := provider.(*providers.TwitterProvider); ok {
		oauthCreds, err = tp.ExchangeCodeWithVerifier(ctx, redirectURL, code, statePayload.CodeVerifier)
	} else {
		oauthCreds, err = provider.ExchangeCode(ctx, redirectURL, code)
	}
	if err != nil {
		return response.BadRequest(c, "OAuth exchange failed: "+err.Error())
	}

	profile, err := provider.GetProfile(ctx, oauthCreds)
	if err != nil {
		return response.InternalError(c)
	}

	encrypted, err := h.creds.Encrypt(oauthCreds)
	if err != nil {
		return response.InternalError(c)
	}

	// Upsert — reconnecting an existing account updates its tokens
	existing, _ := h.social.FindByWorkspaceAndPlatformUser(workspaceID, models.Platform(platformStr), profile.PlatformUserID)
	if existing != nil {
		existing.CredentialsEnc = encrypted
		existing.Name = profile.Name
		existing.Username = profile.Username
		if profile.AvatarURL != "" {
			existing.AvatarURL = &profile.AvatarURL
		}
		_ = h.social.Update(existing)
	} else {
		avatarURL := profile.AvatarURL
		_ = h.social.Create(&models.SocialAccount{
			WorkspaceID:    workspaceID,
			Platform:       models.Platform(platformStr),
			PlatformUserID: profile.PlatformUserID,
			Name:           profile.Name,
			Username:       profile.Username,
			AvatarURL:      &avatarURL,
			CredentialsEnc: encrypted,
		})
	}

	return c.Redirect(fmt.Sprintf("%s/apps?connected=%s", h.frontendURL, platformStr))
}

// ListAccounts GET /api/v1/social/accounts?workspace_id=...
func (h *Handler) ListAccounts(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}

	workspaceID, err := uuid.Parse(c.Query("workspace_id"))
	if err != nil {
		return response.BadRequest(c, "workspace_id required")
	}

	if err := h.orgs.RequireWorkspaceAccess(workspaceID, userID); err != nil {
		return response.Forbidden(c)
	}

	accounts, err := h.social.FindByWorkspace(workspaceID)
	if err != nil {
		return response.InternalError(c)
	}
	return response.OK(c, accounts)
}

// Disconnect DELETE /api/v1/social/accounts/:id
func (h *Handler) Disconnect(c *fiber.Ctx) error {
	userID, ok := middleware.UserIDFromCtx(c)
	if !ok {
		return response.Unauthorized(c)
	}

	accountID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "invalid id")
	}

	account, err := h.social.FindByID(accountID)
	if err != nil {
		return response.NotFound(c, "social account")
	}

	if err := h.orgs.RequireWorkspaceAccess(account.WorkspaceID, userID); err != nil {
		return response.Forbidden(c)
	}

	if err := h.social.Delete(accountID); err != nil {
		return response.InternalError(c)
	}
	return response.NoContent(c)
}

func (h *Handler) callbackURL(c *fiber.Ctx, platform string) string {
	return fmt.Sprintf("%s://%s/api/v1/social/callback/%s", c.Protocol(), c.Hostname(), platform)
}

var _ = errors.New // keep import
