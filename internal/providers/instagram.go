package providers

// InstagramProvider implements SocialProvider for Instagram Graph API.
// Reference: brightbean-studio/providers/instagram.py
// API: https://developers.facebook.com/docs/instagram-api
//
// Key flow notes (from brightbean study):
//   - Auth goes through Facebook OAuth (same app), with Instagram-specific scopes
//   - Publishing images: create container → poll until FINISHED → publish container
//   - Publishing video: similar but CONTAINER_POLL_MAX_ATTEMPTS is much higher
//   - Access tokens can be exchanged for long-lived tokens (60 days)

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/allposty/allposty-backend/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
)

const (
	instagramBaseURL     = "https://graph.facebook.com/v21.0"
	containerPollMax     = 60
	containerPollSleep   = 2 * time.Second
)

var instagramScopes = []string{
	"instagram_basic",
	"instagram_content_publish",
	"instagram_manage_comments",
	"instagram_manage_insights",
	"pages_show_list",
	"pages_read_engagement",
}

type InstagramProvider struct {
	clientID     string
	clientSecret string
}

func NewInstagramProvider(cfg *config.Config) *InstagramProvider {
	return &InstagramProvider{
		clientID:     cfg.OAuth.Facebook.ClientID,
		clientSecret: cfg.OAuth.Facebook.ClientSecret,
	}
}

func (p *InstagramProvider) Platform() Platform { return Instagram }

func (p *InstagramProvider) OAuthConfig(redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       instagramScopes,
		Endpoint:     facebook.Endpoint,
	}
}

func (p *InstagramProvider) AuthURL(redirectURL, state string) string {
	return p.OAuthConfig(redirectURL).AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (p *InstagramProvider) ExchangeCode(ctx context.Context, redirectURL, code string) (*OAuthCredentials, error) {
	cfg := p.OAuthConfig(redirectURL)
	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("instagram: exchange code: %w", err)
	}

	// Exchange for long-lived token (60 days)
	longLived, err := p.exchangeLongLived(ctx, token.AccessToken)
	if err != nil {
		// Fall back to short-lived if exchange fails
		longLived = &OAuthCredentials{
			AccessToken: token.AccessToken,
			ExpiresAt:   token.Expiry,
		}
	}

	return longLived, nil
}

func (p *InstagramProvider) RefreshTokens(ctx context.Context, creds *OAuthCredentials) (*OAuthCredentials, error) {
	// Instagram long-lived tokens can be refreshed within 60 days of expiry
	refreshed, err := p.exchangeLongLived(ctx, creds.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("instagram: refresh: %w", err)
	}
	return refreshed, nil
}

func (p *InstagramProvider) GetProfile(ctx context.Context, creds *OAuthCredentials) (*AccountProfile, error) {
	igAccountID, err := p.getInstagramAccountID(ctx, creds.AccessToken)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("%s/%s?fields=id,name,username,profile_picture_url&access_token=%s",
		instagramBaseURL, igAccountID, creds.AccessToken)

	resp, err := httpGet(ctx, apiURL)
	if err != nil {
		return nil, err
	}

	var result struct {
		ID                 string `json:"id"`
		Name               string `json:"name"`
		Username           string `json:"username"`
		ProfilePictureURL  string `json:"profile_picture_url"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	return &AccountProfile{
		PlatformUserID: result.ID,
		Name:           result.Name,
		Username:       result.Username,
		AvatarURL:      result.ProfilePictureURL,
	}, nil
}

func (p *InstagramProvider) Publish(ctx context.Context, creds *OAuthCredentials, content *PublishContent) (*PublishResult, error) {
	igAccountID := creds.Extra["ig_account_id"]
	if igAccountID == "" {
		var err error
		igAccountID, err = p.getInstagramAccountID(ctx, creds.AccessToken)
		if err != nil {
			return nil, err
		}
	}

	// Step 1: Create media container
	containerID, err := p.createContainer(ctx, creds.AccessToken, igAccountID, content)
	if err != nil {
		return nil, fmt.Errorf("instagram: create container: %w", err)
	}

	// Step 2: Poll until container status is FINISHED
	if err := p.waitForContainer(ctx, creds.AccessToken, igAccountID, containerID); err != nil {
		return nil, fmt.Errorf("instagram: container not ready: %w", err)
	}

	// Step 3: Publish the container
	postID, err := p.publishContainer(ctx, creds.AccessToken, igAccountID, containerID)
	if err != nil {
		return nil, fmt.Errorf("instagram: publish container: %w", err)
	}

	return &PublishResult{
		PlatformPostID: postID,
		PostURL:        fmt.Sprintf("https://www.instagram.com/p/%s", postID),
		PublishedAt:    time.Now().UTC(),
	}, nil
}

// --- internal helpers ---

func (p *InstagramProvider) exchangeLongLived(ctx context.Context, shortToken string) (*OAuthCredentials, error) {
	apiURL := fmt.Sprintf("%s/oauth/access_token?grant_type=fb_exchange_token&client_id=%s&client_secret=%s&fb_exchange_token=%s",
		instagramBaseURL, p.clientID, p.clientSecret, shortToken)

	body, err := httpGet(ctx, apiURL)
	if err != nil {
		return nil, err
	}

	var result struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &OAuthCredentials{
		AccessToken: result.AccessToken,
		TokenType:   result.TokenType,
		ExpiresAt:   time.Now().Add(time.Duration(result.ExpiresIn) * time.Second),
	}, nil
}

func (p *InstagramProvider) getInstagramAccountID(ctx context.Context, accessToken string) (string, error) {
	apiURL := fmt.Sprintf("%s/me/accounts?access_token=%s", instagramBaseURL, accessToken)
	body, err := httpGet(ctx, apiURL)
	if err != nil {
		return "", err
	}

	var pages struct {
		Data []struct {
			ID          string `json:"id"`
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &pages); err != nil || len(pages.Data) == 0 {
		return "", fmt.Errorf("instagram: no linked Facebook page found")
	}

	pageID := pages.Data[0].ID
	igURL := fmt.Sprintf("%s/%s?fields=instagram_business_account&access_token=%s",
		instagramBaseURL, pageID, accessToken)

	body, err = httpGet(ctx, igURL)
	if err != nil {
		return "", err
	}

	var igResult struct {
		InstagramBusinessAccount struct {
			ID string `json:"id"`
		} `json:"instagram_business_account"`
	}
	if err := json.Unmarshal(body, &igResult); err != nil || igResult.InstagramBusinessAccount.ID == "" {
		return "", fmt.Errorf("instagram: no Instagram Business account linked to page")
	}

	return igResult.InstagramBusinessAccount.ID, nil
}

func (p *InstagramProvider) createContainer(ctx context.Context, token, igAccountID string, content *PublishContent) (string, error) {
	params := url.Values{}
	params.Set("access_token", token)
	params.Set("caption", content.Caption)

	if len(content.MediaURLs) > 0 {
		if content.MediaType == MediaTypeVideo {
			params.Set("media_type", "REELS")
			params.Set("video_url", content.MediaURLs[0])
		} else {
			params.Set("image_url", content.MediaURLs[0])
		}
	}

	apiURL := fmt.Sprintf("%s/%s/media", instagramBaseURL, igAccountID)
	body, err := httpPost(ctx, apiURL, params)
	if err != nil {
		return "", err
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return result.ID, nil
}

func (p *InstagramProvider) waitForContainer(ctx context.Context, token, igAccountID, containerID string) error {
	for i := 0; i < containerPollMax; i++ {
		apiURL := fmt.Sprintf("%s/%s?fields=status_code&access_token=%s",
			instagramBaseURL, containerID, token)
		body, err := httpGet(ctx, apiURL)
		if err != nil {
			return err
		}

		var status struct {
			StatusCode string `json:"status_code"`
		}
		if err := json.Unmarshal(body, &status); err != nil {
			return err
		}

		switch status.StatusCode {
		case "FINISHED":
			return nil
		case "ERROR", "EXPIRED":
			return fmt.Errorf("instagram: container %s status: %s", containerID, status.StatusCode)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(containerPollSleep):
		}
	}
	return fmt.Errorf("instagram: container %s timed out after %d polls", containerID, containerPollMax)
}

func (p *InstagramProvider) publishContainer(ctx context.Context, token, igAccountID, containerID string) (string, error) {
	params := url.Values{}
	params.Set("access_token", token)
	params.Set("creation_id", containerID)

	apiURL := fmt.Sprintf("%s/%s/media_publish", instagramBaseURL, igAccountID)
	body, err := httpPost(ctx, apiURL, params)
	if err != nil {
		return "", err
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return result.ID, nil
}

// --- shared HTTP helpers ---

func httpGet(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func httpPost(ctx context.Context, rawURL string, params url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
