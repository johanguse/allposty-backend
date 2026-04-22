package providers

// LinkedInProvider implements SocialProvider for LinkedIn (personal + company).
// Reference: brightbean-studio/providers/linkedin.py + linkedin_company.py
// API: https://learn.microsoft.com/en-us/linkedin/marketing/community-management/shares/posts-api
//
// Key flow notes:
//   - OAuth 2.0 with PKCE recommended
//   - Use the UGC Posts API (v2) or newer Posts API (r_liteprofile, w_member_social)
//   - Images must be uploaded via the Assets API first, then referenced in the post
//   - Company posts go to /v2/ugcPosts with author = "urn:li:organization:{id}"

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/allposty/allposty-backend/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/linkedin"
)

const linkedInBaseURL = "https://api.linkedin.com/v2"

var linkedInScopes = []string{
	"openid",
	"profile",
	"email",
	"w_member_social",
}

type LinkedInProvider struct {
	clientID     string
	clientSecret string
}

func NewLinkedInProvider(cfg *config.Config) *LinkedInProvider {
	return &LinkedInProvider{
		clientID:     cfg.OAuth.LinkedIn.ClientID,
		clientSecret: cfg.OAuth.LinkedIn.ClientSecret,
	}
}

func (p *LinkedInProvider) Platform() Platform { return LinkedIn }

func (p *LinkedInProvider) OAuthConfig(redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       linkedInScopes,
		Endpoint:     linkedin.Endpoint,
	}
}

func (p *LinkedInProvider) AuthURL(redirectURL, state string) string {
	return p.OAuthConfig(redirectURL).AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (p *LinkedInProvider) ExchangeCode(ctx context.Context, redirectURL, code string) (*OAuthCredentials, error) {
	token, err := p.OAuthConfig(redirectURL).Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("linkedin: exchange code: %w", err)
	}
	return &OAuthCredentials{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		ExpiresAt:    token.Expiry,
	}, nil
}

func (p *LinkedInProvider) RefreshTokens(ctx context.Context, creds *OAuthCredentials) (*OAuthCredentials, error) {
	if creds.RefreshToken == "" {
		return nil, ErrRefreshNotSupported
	}
	token := &oauth2.Token{RefreshToken: creds.RefreshToken}
	newToken, err := p.OAuthConfig("").TokenSource(ctx, token).Token()
	if err != nil {
		return nil, fmt.Errorf("linkedin: refresh: %w", err)
	}
	return &OAuthCredentials{
		AccessToken:  newToken.AccessToken,
		RefreshToken: newToken.RefreshToken,
		TokenType:    newToken.TokenType,
		ExpiresAt:    newToken.Expiry,
	}, nil
}

func (p *LinkedInProvider) GetProfile(ctx context.Context, creds *OAuthCredentials) (*AccountProfile, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, linkedInBaseURL+"/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+creds.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var profile struct {
		Sub     string `json:"sub"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture string `json:"picture"`
	}
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, err
	}
	return &AccountProfile{
		PlatformUserID: profile.Sub,
		Name:           profile.Name,
		Username:       profile.Email,
		AvatarURL:      profile.Picture,
	}, nil
}

func (p *LinkedInProvider) Publish(ctx context.Context, creds *OAuthCredentials, content *PublishContent) (*PublishResult, error) {
	authorURN := creds.Extra["author_urn"]
	if authorURN == "" {
		profile, err := p.GetProfile(ctx, creds)
		if err != nil {
			return nil, err
		}
		authorURN = fmt.Sprintf("urn:li:person:%s", profile.PlatformUserID)
	}

	payload := map[string]any{
		"author":         authorURN,
		"lifecycleState": "PUBLISHED",
		"specificContent": map[string]any{
			"com.linkedin.ugc.ShareContent": map[string]any{
				"shareCommentary": map[string]any{
					"text": content.Caption,
				},
				"shareMediaCategory": "NONE",
			},
		},
		"visibility": map[string]any{
			"com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC",
		},
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		linkedInBaseURL+"/ugcPosts", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+creds.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("linkedin: publish: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("linkedin: publish failed %d: %s", resp.StatusCode, string(b))
	}

	postID := resp.Header.Get("X-RestLi-Id")
	return &PublishResult{
		PlatformPostID: postID,
		PostURL:        fmt.Sprintf("https://www.linkedin.com/feed/update/%s", postID),
		PublishedAt:    time.Now().UTC(),
	}, nil
}
