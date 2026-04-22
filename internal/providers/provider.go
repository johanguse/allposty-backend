package providers

import (
	"context"
	"time"

	"golang.org/x/oauth2"
)

// Platform identifies a social media platform.
type Platform string

const (
	Instagram Platform = "instagram"
	Facebook  Platform = "facebook"
	LinkedIn  Platform = "linkedin"
	Twitter   Platform = "twitter"
	TikTok    Platform = "tiktok"
	YouTube   Platform = "youtube"
)

// OAuthCredentials holds the tokens stored (encrypted) per social account.
type OAuthCredentials struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	// Platform-specific extras (page IDs, user IDs, etc.)
	Extra map[string]string `json:"extra,omitempty"`
}

// AccountProfile is the basic profile returned after connecting an account.
type AccountProfile struct {
	PlatformUserID string
	Name           string
	Username       string
	AvatarURL      string
}

// MediaType classifies the media attached to a post.
type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"
)

// PublishContent is everything the provider needs to publish a post.
type PublishContent struct {
	Caption   string
	MediaURLs []string
	MediaType MediaType
	// Platform-specific options (e.g. TikTok privacy level, YouTube title)
	Options map[string]any
}

// PublishResult is returned after a successful publish.
type PublishResult struct {
	PlatformPostID string
	PostURL        string
	PublishedAt    time.Time
}

// SocialProvider is the interface every platform must implement.
// Study brightbean-studio/providers/*.py for OAuth flow details per platform.
type SocialProvider interface {
	// Platform returns the platform identifier.
	Platform() Platform

	// OAuthConfig returns the oauth2.Config for this provider.
	// The redirectURL is injected at runtime so tests can override it.
	OAuthConfig(redirectURL string) *oauth2.Config

	// AuthURL returns the URL to redirect the user to for OAuth consent.
	AuthURL(redirectURL, state string) string

	// ExchangeCode exchanges the OAuth callback code for credentials.
	ExchangeCode(ctx context.Context, redirectURL, code string) (*OAuthCredentials, error)

	// RefreshTokens refreshes the access token using the stored refresh token.
	// Returns new credentials. If the platform doesn't support refresh, return ErrRefreshNotSupported.
	RefreshTokens(ctx context.Context, creds *OAuthCredentials) (*OAuthCredentials, error)

	// GetProfile fetches the connected account's profile using the stored credentials.
	GetProfile(ctx context.Context, creds *OAuthCredentials) (*AccountProfile, error)

	// Publish publishes content to the platform.
	Publish(ctx context.Context, creds *OAuthCredentials, content *PublishContent) (*PublishResult, error)
}

// ProviderErrors
var (
	ErrRefreshNotSupported = providerError("refresh not supported for this platform")
	ErrInvalidCredentials  = providerError("invalid or expired credentials")
	ErrRateLimited         = providerError("rate limited by platform")
	ErrPublishFailed       = providerError("publish failed")
)

type providerError string

func (e providerError) Error() string { return string(e) }
