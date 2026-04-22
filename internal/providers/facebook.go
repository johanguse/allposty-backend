package providers

// FacebookProvider implements SocialProvider for Facebook Pages.
// Reference: brightbean-studio/providers/facebook.py
// API: https://developers.facebook.com/docs/pages/publishing
//
// Key flow notes:
//   - Auth via Facebook OAuth, needs pages_manage_posts + pages_read_engagement
//   - Publishing to a Page (not personal profile): POST /{page-id}/feed
//   - Page access token is separate from user token — obtained via /me/accounts
//   - Videos go to /{page-id}/videos with a video_url parameter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/allposty/allposty-backend/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
)

var facebookScopes = []string{
	"pages_manage_posts",
	"pages_read_engagement",
	"pages_show_list",
	"public_profile",
}

type FacebookProvider struct {
	clientID     string
	clientSecret string
}

func NewFacebookProvider(cfg *config.Config) *FacebookProvider {
	return &FacebookProvider{
		clientID:     cfg.OAuth.Facebook.ClientID,
		clientSecret: cfg.OAuth.Facebook.ClientSecret,
	}
}

func (p *FacebookProvider) Platform() Platform { return Facebook }

func (p *FacebookProvider) OAuthConfig(redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       facebookScopes,
		Endpoint:     facebook.Endpoint,
	}
}

func (p *FacebookProvider) AuthURL(redirectURL, state string) string {
	return p.OAuthConfig(redirectURL).AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (p *FacebookProvider) ExchangeCode(ctx context.Context, redirectURL, code string) (*OAuthCredentials, error) {
	cfg := p.OAuthConfig(redirectURL)
	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("facebook: exchange code: %w", err)
	}
	return &OAuthCredentials{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		ExpiresAt:   token.Expiry,
	}, nil
}

func (p *FacebookProvider) RefreshTokens(_ context.Context, _ *OAuthCredentials) (*OAuthCredentials, error) {
	return nil, ErrRefreshNotSupported
}

func (p *FacebookProvider) GetProfile(ctx context.Context, creds *OAuthCredentials) (*AccountProfile, error) {
	pages, err := p.getPages(ctx, creds.AccessToken)
	if err != nil || len(pages) == 0 {
		return nil, fmt.Errorf("facebook: no pages found")
	}
	page := pages[0]
	return &AccountProfile{
		PlatformUserID: page.ID,
		Name:           page.Name,
		Username:       page.ID,
	}, nil
}

func (p *FacebookProvider) Publish(ctx context.Context, creds *OAuthCredentials, content *PublishContent) (*PublishResult, error) {
	pageID := creds.Extra["page_id"]
	pageToken := creds.Extra["page_access_token"]

	if pageID == "" || pageToken == "" {
		pages, err := p.getPages(ctx, creds.AccessToken)
		if err != nil || len(pages) == 0 {
			return nil, fmt.Errorf("facebook: no page available to publish")
		}
		pageID = pages[0].ID
		pageToken = pages[0].AccessToken
	}

	params := url.Values{}
	params.Set("access_token", pageToken)
	params.Set("message", content.Caption)
	if len(content.MediaURLs) > 0 {
		params.Set("link", content.MediaURLs[0])
	}

	endpoint := fmt.Sprintf("%s/%s/feed", instagramBaseURL, pageID)
	if content.MediaType == MediaTypeVideo {
		endpoint = fmt.Sprintf("%s/%s/videos", instagramBaseURL, pageID)
		params.Set("file_url", content.MediaURLs[0])
		params.Set("description", content.Caption)
		delete(params, "message")
	}

	body, err := httpPost(ctx, endpoint, params)
	if err != nil {
		return nil, fmt.Errorf("facebook: publish: %w", err)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &PublishResult{
		PlatformPostID: result.ID,
		PostURL:        fmt.Sprintf("https://www.facebook.com/%s", result.ID),
		PublishedAt:    time.Now().UTC(),
	}, nil
}

type fbPage struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	AccessToken string `json:"access_token"`
}

func (p *FacebookProvider) getPages(ctx context.Context, userToken string) ([]fbPage, error) {
	apiURL := fmt.Sprintf("%s/me/accounts?access_token=%s", instagramBaseURL, userToken)
	body, err := httpGet(ctx, apiURL)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data []fbPage `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}
