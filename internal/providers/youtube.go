package providers

// YouTubeProvider implements SocialProvider for YouTube Data API v3.
// API: https://developers.google.com/youtube/v3/docs/videos/insert
//
// Key flow notes:
//   - Auth via Google OAuth 2.0 with YouTube-specific scopes
//   - Video upload is a resumable upload (multipart or resumable URI)
//   - For URL-based publishing: not supported natively — must download then re-upload
//   - Metadata: title, description, tags, category, privacy status

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
	"golang.org/x/oauth2/google"
)

const youtubeBaseURL = "https://www.googleapis.com/youtube/v3"

var youtubeScopes = []string{
	"https://www.googleapis.com/auth/youtube.upload",
	"https://www.googleapis.com/auth/youtube.readonly",
	"openid",
	"profile",
	"email",
}

type YouTubeProvider struct {
	clientID     string
	clientSecret string
}

func NewYouTubeProvider(cfg *config.Config) *YouTubeProvider {
	return &YouTubeProvider{
		clientID:     cfg.OAuth.Google.ClientID,
		clientSecret: cfg.OAuth.Google.ClientSecret,
	}
}

func (p *YouTubeProvider) Platform() Platform { return YouTube }

func (p *YouTubeProvider) OAuthConfig(redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       youtubeScopes,
		Endpoint:     google.Endpoint,
	}
}

func (p *YouTubeProvider) AuthURL(redirectURL, state string) string {
	return p.OAuthConfig(redirectURL).AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
	)
}

func (p *YouTubeProvider) ExchangeCode(ctx context.Context, redirectURL, code string) (*OAuthCredentials, error) {
	token, err := p.OAuthConfig(redirectURL).Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("youtube: exchange code: %w", err)
	}
	return &OAuthCredentials{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		ExpiresAt:    token.Expiry,
	}, nil
}

func (p *YouTubeProvider) RefreshTokens(ctx context.Context, creds *OAuthCredentials) (*OAuthCredentials, error) {
	src := p.OAuthConfig("").TokenSource(ctx, &oauth2.Token{RefreshToken: creds.RefreshToken})
	token, err := src.Token()
	if err != nil {
		return nil, fmt.Errorf("youtube: refresh: %w", err)
	}
	return &OAuthCredentials{
		AccessToken:  token.AccessToken,
		RefreshToken: creds.RefreshToken,
		ExpiresAt:    token.Expiry,
	}, nil
}

func (p *YouTubeProvider) GetProfile(ctx context.Context, creds *OAuthCredentials) (*AccountProfile, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		youtubeBaseURL+"/channels?part=snippet&mine=true", nil)
	req.Header.Set("Authorization", "Bearer "+creds.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title     string `json:"title"`
				Thumbnails struct {
					Default struct {
						URL string `json:"url"`
					} `json:"default"`
				} `json:"thumbnails"`
			} `json:"snippet"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &result); err != nil || len(result.Items) == 0 {
		return nil, fmt.Errorf("youtube: no channel found")
	}
	ch := result.Items[0]
	return &AccountProfile{
		PlatformUserID: ch.ID,
		Name:           ch.Snippet.Title,
		Username:       ch.ID,
		AvatarURL:      ch.Snippet.Thumbnails.Default.URL,
	}, nil
}

func (p *YouTubeProvider) Publish(ctx context.Context, creds *OAuthCredentials, content *PublishContent) (*PublishResult, error) {
	if len(content.MediaURLs) == 0 {
		return nil, fmt.Errorf("youtube: video URL is required")
	}

	title := content.Options["title"]
	if title == nil || title == "" {
		// Truncate caption to 100 chars as title
		title = content.Caption
		if len(content.Caption) > 100 {
			title = content.Caption[:97] + "..."
		}
	}

	metadata := map[string]any{
		"snippet": map[string]any{
			"title":       title,
			"description": content.Caption,
			"tags":        []string{},
			"categoryId":  "22", // People & Blogs
		},
		"status": map[string]any{
			"privacyStatus": "public",
		},
	}

	metaBytes, _ := json.Marshal(metadata)
	uploadURL := "https://www.googleapis.com/upload/youtube/v3/videos?uploadType=resumable&part=snippet,status"

	initReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(metaBytes))
	initReq.Header.Set("Authorization", "Bearer "+creds.AccessToken)
	initReq.Header.Set("Content-Type", "application/json")
	initReq.Header.Set("X-Upload-Content-Type", "video/*")

	initResp, err := http.DefaultClient.Do(initReq)
	if err != nil {
		return nil, fmt.Errorf("youtube: init resumable upload: %w", err)
	}
	defer initResp.Body.Close()

	resumableURI := initResp.Header.Get("Location")
	if resumableURI == "" {
		return nil, fmt.Errorf("youtube: no resumable URI returned")
	}

	// For URL-based media: fetch the video then stream it
	// In production this should be done asynchronously (the video may be large)
	videoResp, err := http.Get(content.MediaURLs[0])
	if err != nil {
		return nil, fmt.Errorf("youtube: fetch video: %w", err)
	}
	defer videoResp.Body.Close()

	uploadReq, _ := http.NewRequestWithContext(ctx, http.MethodPut, resumableURI, videoResp.Body)
	uploadReq.Header.Set("Content-Type", "video/*")
	uploadReq.ContentLength = videoResp.ContentLength

	uploadResp, err := http.DefaultClient.Do(uploadReq)
	if err != nil {
		return nil, fmt.Errorf("youtube: upload video: %w", err)
	}
	defer uploadResp.Body.Close()
	respBody, _ := io.ReadAll(uploadResp.Body)

	var videoResult struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &videoResult); err != nil {
		return nil, err
	}

	return &PublishResult{
		PlatformPostID: videoResult.ID,
		PostURL:        fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoResult.ID),
		PublishedAt:    time.Now().UTC(),
	}, nil
}
