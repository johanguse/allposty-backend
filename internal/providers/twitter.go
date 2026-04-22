package providers

// TwitterProvider implements SocialProvider for Twitter/X v2 API.
// API: https://developer.twitter.com/en/docs/twitter-api/tweets/manage-tweets/api-reference/post-tweets
//
// OAuth 2.0 + PKCE flow:
//   1. Connect()  → call GeneratePKCE(), store verifier in Redis state, redirect with challenge
//   2. Callback() → retrieve verifier from Redis state, call ExchangeCode(redirectURL, code, verifier)
//
// Token lifetimes: access token 2h, refresh token 6 months (offline.access scope).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/allposty/allposty-backend/internal/config"
	"golang.org/x/oauth2"
)

const twitterBaseURL = "https://api.twitter.com/2"

var twitterEndpoint = oauth2.Endpoint{
	AuthURL:   "https://twitter.com/i/oauth2/authorize",
	TokenURL:  "https://api.twitter.com/2/oauth2/token",
	AuthStyle: oauth2.AuthStyleInHeader,
}

var twitterScopes = []string{
	"tweet.read",
	"tweet.write",
	"users.read",
	"offline.access",
}

type TwitterProvider struct {
	clientID     string
	clientSecret string
}

func NewTwitterProvider(cfg *config.Config) *TwitterProvider {
	return &TwitterProvider{
		clientID:     cfg.OAuth.Twitter.ClientID,
		clientSecret: cfg.OAuth.Twitter.ClientSecret,
	}
}

func (p *TwitterProvider) Platform() Platform { return Twitter }

func (p *TwitterProvider) OAuthConfig(redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       twitterScopes,
		Endpoint:     twitterEndpoint,
	}
}

// GeneratePKCE returns a new PKCE pair. The caller stores the verifier in Redis
// (keyed by state) and passes the challenge to AuthURL.
func (p *TwitterProvider) GeneratePKCE() (*PKCEPair, error) {
	return NewPKCE()
}

// AuthURL builds the consent URL with the PKCE challenge embedded.
// challenge comes from PKCEPair.Challenge (S256).
func (p *TwitterProvider) AuthURL(redirectURL, state string) string {
	// Without a challenge — this signature satisfies the SocialProvider interface.
	// The social handler calls AuthURLWithPKCE instead when platform == twitter.
	return p.OAuthConfig(redirectURL).AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// AuthURLWithPKCE is the Twitter-specific variant that includes the S256 challenge.
func (p *TwitterProvider) AuthURLWithPKCE(redirectURL, state, challenge string) string {
	return p.OAuthConfig(redirectURL).AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

// ExchangeCode exchanges the authorization code using the stored PKCE verifier.
// verifier is retrieved from Redis by the callback handler.
func (p *TwitterProvider) ExchangeCode(ctx context.Context, redirectURL, code string) (*OAuthCredentials, error) {
	// This satisfies the SocialProvider interface but Twitter always needs a verifier.
	// The social handler calls ExchangeCodeWithVerifier for twitter.
	return p.ExchangeCodeWithVerifier(ctx, redirectURL, code, "")
}

// ExchangeCodeWithVerifier is the Twitter-specific token exchange with PKCE.
func (p *TwitterProvider) ExchangeCodeWithVerifier(ctx context.Context, redirectURL, code, verifier string) (*OAuthCredentials, error) {
	opts := []oauth2.AuthCodeOption{}
	if verifier != "" {
		opts = append(opts, oauth2.SetAuthURLParam("code_verifier", verifier))
	}
	token, err := p.OAuthConfig(redirectURL).Exchange(ctx, code, opts...)
	if err != nil {
		return nil, fmt.Errorf("twitter: exchange code: %w", err)
	}
	return &OAuthCredentials{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		ExpiresAt:    token.Expiry,
	}, nil
}

func (p *TwitterProvider) RefreshTokens(ctx context.Context, creds *OAuthCredentials) (*OAuthCredentials, error) {
	if creds.RefreshToken == "" {
		return nil, ErrRefreshNotSupported
	}
	src := p.OAuthConfig("").TokenSource(ctx, &oauth2.Token{RefreshToken: creds.RefreshToken})
	token, err := src.Token()
	if err != nil {
		return nil, fmt.Errorf("twitter: refresh: %w", err)
	}
	return &OAuthCredentials{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.Expiry,
	}, nil
}

func (p *TwitterProvider) GetProfile(ctx context.Context, creds *OAuthCredentials) (*AccountProfile, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		twitterBaseURL+"/users/me?user.fields=profile_image_url,username", nil)
	req.Header.Set("Authorization", "Bearer "+creds.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Data struct {
			ID              string `json:"id"`
			Name            string `json:"name"`
			Username        string `json:"username"`
			ProfileImageURL string `json:"profile_image_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &AccountProfile{
		PlatformUserID: result.Data.ID,
		Name:           result.Data.Name,
		Username:       result.Data.Username,
		AvatarURL:      result.Data.ProfileImageURL,
	}, nil
}

func (p *TwitterProvider) Publish(ctx context.Context, creds *OAuthCredentials, content *PublishContent) (*PublishResult, error) {
	payload := map[string]any{
		"text": content.Caption,
	}
	// TODO: media upload — attach media_ids after uploading via v1.1 /media/upload

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		twitterBaseURL+"/tweets", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+creds.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("twitter: publish: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("twitter: publish failed %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	return &PublishResult{
		PlatformPostID: result.Data.ID,
		PostURL:        fmt.Sprintf("https://twitter.com/i/web/status/%s", result.Data.ID),
		PublishedAt:    time.Now().UTC(),
	}, nil
}
