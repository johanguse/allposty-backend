package providers

// TikTokProvider implements SocialProvider for TikTok Content Posting API.
// API: https://developers.tiktok.com/doc/content-posting-api-get-started
//
// Key flow notes:
//   - OAuth 2.0 with PKCE
//   - Two upload approaches: PULL_FROM_URL (give TikTok a URL) or FILE_UPLOAD (chunked)
//   - Video is the primary content type; photos also supported (photo post)
//   - Must call /v2/post/publish/video/init/ → get upload_url → upload file → check status
//   - Privacy levels: PUBLIC_TO_EVERYONE, MUTUAL_FOLLOW_FRIENDS, FOLLOWER_OF_CREATOR, SELF_ONLY

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

const tiktokBaseURL = "https://open.tiktokapis.com/v2"

var tiktokEndpoint = oauth2.Endpoint{
	AuthURL:   "https://www.tiktok.com/v2/auth/authorize/",
	TokenURL:  "https://open.tiktokapis.com/v2/oauth/token/",
	AuthStyle: oauth2.AuthStyleInParams,
}

var tiktokScopes = []string{
	"user.info.basic",
	"video.publish",
	"video.upload",
}

type TikTokProvider struct {
	clientKey    string
	clientSecret string
}

func NewTikTokProvider(cfg *config.Config) *TikTokProvider {
	return &TikTokProvider{
		clientKey:    cfg.OAuth.TikTok.ClientID,
		clientSecret: cfg.OAuth.TikTok.ClientSecret,
	}
}

func (p *TikTokProvider) Platform() Platform { return TikTok }

func (p *TikTokProvider) OAuthConfig(redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.clientKey,
		ClientSecret: p.clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       tiktokScopes,
		Endpoint:     tiktokEndpoint,
	}
}

func (p *TikTokProvider) AuthURL(redirectURL, state string) string {
	return p.OAuthConfig(redirectURL).AuthCodeURL(state,
		oauth2.SetAuthURLParam("client_key", p.clientKey),
	)
}

func (p *TikTokProvider) ExchangeCode(ctx context.Context, redirectURL, code string) (*OAuthCredentials, error) {
	token, err := p.OAuthConfig(redirectURL).Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("tiktok: exchange code: %w", err)
	}
	return &OAuthCredentials{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		ExpiresAt:    token.Expiry,
	}, nil
}

func (p *TikTokProvider) RefreshTokens(ctx context.Context, creds *OAuthCredentials) (*OAuthCredentials, error) {
	if creds.RefreshToken == "" {
		return nil, ErrRefreshNotSupported
	}
	src := p.OAuthConfig("").TokenSource(ctx, &oauth2.Token{RefreshToken: creds.RefreshToken})
	token, err := src.Token()
	if err != nil {
		return nil, fmt.Errorf("tiktok: refresh: %w", err)
	}
	return &OAuthCredentials{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.Expiry,
	}, nil
}

func (p *TikTokProvider) GetProfile(ctx context.Context, creds *OAuthCredentials) (*AccountProfile, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		tiktokBaseURL+"/user/info/?fields=open_id,display_name,avatar_url,union_id", nil)
	req.Header.Set("Authorization", "Bearer "+creds.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Data struct {
			User struct {
				OpenID      string `json:"open_id"`
				DisplayName string `json:"display_name"`
				AvatarURL   string `json:"avatar_url"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	u := result.Data.User
	return &AccountProfile{
		PlatformUserID: u.OpenID,
		Name:           u.DisplayName,
		Username:       u.DisplayName,
		AvatarURL:      u.AvatarURL,
	}, nil
}

func (p *TikTokProvider) Publish(ctx context.Context, creds *OAuthCredentials, content *PublishContent) (*PublishResult, error) {
	if len(content.MediaURLs) == 0 {
		return nil, fmt.Errorf("tiktok: video URL is required")
	}

	// Use PULL_FROM_URL approach (simpler: give TikTok a public URL)
	payload := map[string]any{
		"post_info": map[string]any{
			"title":         content.Caption,
			"privacy_level": "PUBLIC_TO_EVERYONE",
			"disable_duet":  false,
			"disable_stitch": false,
			"disable_comment": false,
		},
		"source_info": map[string]any{
			"source":    "PULL_FROM_URL",
			"video_url": content.MediaURLs[0],
		},
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		tiktokBaseURL+"/post/publish/video/init/", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+creds.AccessToken)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tiktok: publish init: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Data struct {
			PublishID string `json:"publish_id"`
		} `json:"data"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	if result.Error.Code != "" && result.Error.Code != "ok" {
		return nil, fmt.Errorf("tiktok: %s: %s", result.Error.Code, result.Error.Message)
	}

	return &PublishResult{
		PlatformPostID: result.Data.PublishID,
		PublishedAt:    time.Now().UTC(),
	}, nil
}
