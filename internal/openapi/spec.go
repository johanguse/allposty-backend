package openapi

// Build returns the complete OpenAPI 3.0.3 spec for the allposty API.
// Served at GET /api/v1/openapi.json — consumed by the frontend's gen:api script.
func Build(serverURL string) Spec {
	return Spec{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:       "Allposty API",
			Version:     "1.0.0",
			Description: "Social media scheduling SaaS. All endpoints return `{\"data\": ...}` on success and `{\"error\": \"...\"}` on failure.",
			License:     License{Name: "AGPL-3.0", URL: "https://www.gnu.org/licenses/agpl-3.0.html"},
		},
		Servers: []Server{
			{URL: serverURL, Description: "Current environment"},
		},
		Tags: []Tag{
			{Name: "auth", Description: "Registration, login, token refresh"},
			{Name: "orgs", Description: "Organizations and workspaces"},
			{Name: "social", Description: "Social account OAuth connections"},
			{Name: "posts", Description: "Post composer, scheduling, calendar"},
			{Name: "media", Description: "Media library"},
			{Name: "ai", Description: "AI-assisted caption generation"},
			{Name: "api-keys", Description: "Long-lived API keys for programmatic access"},
			{Name: "billing", Description: "Stripe subscription management"},
		},
		Components: Components{
			Schemas: buildSchemas(),
			SecuritySchemes: map[string]SecurityScheme{
				"bearerAuth": {
					Type:         "http",
					Scheme:       "bearer",
					BearerFormat: "JWT or allposty_*",
					Description:  "JWT access token from /auth/login, or an API key (allposty_...) from /api-keys",
				},
			},
		},
		Paths: buildPaths(),
	}
}

func buildPaths() map[string]PathItem {
	return map[string]PathItem{

		// ── Auth ───────────────────────────────────────────────────────────

		"/auth/register": {
			Post: &Operation{
				OperationID: "authRegister",
				Summary:     "Register a new user",
				Tags:        []string{"auth"},
				RequestBody: jsonBody(Schema{
					Type:     "object",
					Required: []string{"name", "email", "password"},
					Properties: map[string]Schema{
						"name":     str("Display name"),
						"email":    str(""),
						"password": {Type: "string", Format: "password", Description: "Min 8 characters"},
					},
				}),
				Responses: map[string]Response{
					"201": jsonResponse("Created", Schema{
						Type: "object",
						Properties: map[string]Schema{
							"user":   ref("User"),
							"tokens": ref("TokenPair"),
						},
					}),
					"409": errResponse("Email already registered"),
				},
			},
		},

		"/auth/login": {
			Post: &Operation{
				OperationID: "authLogin",
				Summary:     "Login with email and password",
				Tags:        []string{"auth"},
				RequestBody: jsonBody(Schema{
					Type:     "object",
					Required: []string{"email", "password"},
					Properties: map[string]Schema{
						"email":    str(""),
						"password": {Type: "string", Format: "password"},
					},
				}),
				Responses: map[string]Response{
					"200": jsonResponse("OK", Schema{
						Type: "object",
						Properties: map[string]Schema{
							"user":   ref("User"),
							"tokens": ref("TokenPair"),
						},
					}),
					"401": errResponse("Invalid credentials"),
				},
			},
		},

		"/auth/refresh": {
			Post: &Operation{
				OperationID: "authRefresh",
				Summary:     "Rotate refresh token",
				Tags:        []string{"auth"},
				RequestBody: jsonBody(Schema{
					Type:     "object",
					Required: []string{"refresh_token"},
					Properties: map[string]Schema{
						"refresh_token": str(""),
					},
				}),
				Responses: map[string]Response{
					"200": jsonResponse("New token pair", ref("TokenPair")),
					"401": errResponse("Token expired or revoked"),
				},
			},
		},

		"/auth/logout": {
			Post: &Operation{
				OperationID: "authLogout",
				Summary:     "Revoke refresh token",
				Tags:        []string{"auth"},
				RequestBody: jsonBody(Schema{
					Type: "object",
					Properties: map[string]Schema{
						"refresh_token": str(""),
					},
				}),
				Responses: map[string]Response{
					"204": {Description: "Logged out"},
				},
			},
		},

		"/auth/me": {
			Get: &Operation{
				OperationID: "authMe",
				Summary:     "Get current user",
				Tags:        []string{"auth"},
				Security:    bearer(),
				Responses: map[string]Response{
					"200": jsonResponse("Current user", ref("User")),
					"401": errResponse("Unauthorized"),
				},
			},
		},

		// ── Orgs ───────────────────────────────────────────────────────────

		"/orgs": {
			Post: &Operation{
				OperationID: "createOrg",
				Summary:     "Create organization",
				Tags:        []string{"orgs"},
				Security:    bearer(),
				RequestBody: jsonBody(Schema{
					Type:     "object",
					Required: []string{"name"},
					Properties: map[string]Schema{"name": str("")},
				}),
				Responses: map[string]Response{
					"201": jsonResponse("Created", ref("Organization")),
				},
			},
			Get: &Operation{
				OperationID: "listOrgs",
				Summary:     "List organizations I belong to",
				Tags:        []string{"orgs"},
				Security:    bearer(),
				Responses: map[string]Response{
					"200": jsonResponse("OK", arr(ref("Organization"))),
				},
			},
		},

		"/orgs/{org_id}": {
			Get: &Operation{
				OperationID: "getOrg",
				Summary:     "Get organization",
				Tags:        []string{"orgs"},
				Security:    bearer(),
				Parameters:  []Parameter{pathParam("org_id", "Organization ID")},
				Responses: map[string]Response{
					"200": jsonResponse("OK", ref("Organization")),
					"403": errResponse("Forbidden"),
					"404": errResponse("Not found"),
				},
			},
		},

		"/orgs/{org_id}/workspaces": {
			Post: &Operation{
				OperationID: "createWorkspace",
				Summary:     "Create workspace",
				Tags:        []string{"orgs"},
				Security:    bearer(),
				Parameters:  []Parameter{pathParam("org_id", "")},
				RequestBody: jsonBody(Schema{
					Type:     "object",
					Required: []string{"name"},
					Properties: map[string]Schema{"name": str("")},
				}),
				Responses: map[string]Response{
					"201": jsonResponse("Created", ref("Workspace")),
					"403": errResponse("Only org owner can create workspaces"),
				},
			},
			Get: &Operation{
				OperationID: "listWorkspaces",
				Summary:     "List workspaces in org",
				Tags:        []string{"orgs"},
				Security:    bearer(),
				Parameters:  []Parameter{pathParam("org_id", "")},
				Responses: map[string]Response{
					"200": jsonResponse("OK", arr(ref("Workspace"))),
				},
			},
		},

		// ── Social ─────────────────────────────────────────────────────────

		"/social/connect/{platform}": {
			Get: &Operation{
				OperationID: "socialConnect",
				Summary:     "Start OAuth flow for a platform",
				Tags:        []string{"social"},
				Security:    bearer(),
				Parameters: []Parameter{
					{Name: "platform", In: "path", Required: true,
						Schema: Schema{Type: "string", Enum: []string{"instagram", "facebook", "linkedin", "twitter", "tiktok", "youtube"}}},
					queryParam("workspace_id", "Target workspace ID", true),
				},
				Responses: map[string]Response{
					"302": {Description: "Redirect to platform OAuth consent page"},
					"400": errResponse("workspace_id required"),
				},
			},
		},

		"/social/callback/{platform}": {
			Get: &Operation{
				OperationID: "socialCallback",
				Summary:     "OAuth callback (called by the platform, not your frontend)",
				Tags:        []string{"social"},
				Parameters: []Parameter{
					{Name: "platform", In: "path", Required: true, Schema: str("")},
					queryParam("code", "OAuth authorization code", true),
					queryParam("state", "CSRF state token", true),
				},
				Responses: map[string]Response{
					"302": {Description: "Redirect to frontend success page"},
					"400": errResponse("Missing code or state"),
				},
			},
		},

		"/social/accounts": {
			Get: &Operation{
				OperationID: "listSocialAccounts",
				Summary:     "List connected social accounts",
				Tags:        []string{"social"},
				Security:    bearer(),
				Parameters:  []Parameter{queryParam("workspace_id", "", true)},
				Responses: map[string]Response{
					"200": jsonResponse("OK", arr(ref("SocialAccount"))),
				},
			},
		},

		"/social/accounts/{id}": {
			Delete: &Operation{
				OperationID: "disconnectSocialAccount",
				Summary:     "Disconnect a social account",
				Tags:        []string{"social"},
				Security:    bearer(),
				Parameters:  []Parameter{pathParam("id", "Social account ID")},
				Responses: map[string]Response{
					"204": {Description: "Disconnected"},
					"404": errResponse("Not found"),
				},
			},
		},

		// ── Posts ──────────────────────────────────────────────────────────

		"/posts": {
			Post: &Operation{
				OperationID: "createPost",
				Summary:     "Create a post (optionally schedule it)",
				Tags:        []string{"posts"},
				Security:    bearer(),
				RequestBody: jsonBody(Schema{
					Type:     "object",
					Required: []string{"workspace_id", "social_account_ids"},
					Properties: map[string]Schema{
						"workspace_id":       uuid_(""),
						"caption":            str("Main caption text"),
						"media_urls":         arr(str("Public URL to media file")),
						"social_account_ids": arr(uuid_("IDs of SocialAccount to publish to")),
						"scheduled_at":       ts("Omit to save as draft"),
					},
				}),
				Responses: map[string]Response{
					"201": jsonResponse("Created", ref("Post")),
					"403": errResponse("No access to workspace"),
				},
			},
			Get: &Operation{
				OperationID: "listPosts",
				Summary:     "List posts in a workspace",
				Tags:        []string{"posts"},
				Security:    bearer(),
				Parameters: []Parameter{
					queryParam("workspace_id", "", true),
					queryParam("status", "draft | scheduled | published | failed", false),
				},
				Responses: map[string]Response{
					"200": jsonResponse("OK", arr(ref("Post"))),
				},
			},
		},

		"/posts/calendar": {
			Get: &Operation{
				OperationID: "getCalendar",
				Summary:     "Posts in a date range (calendar view)",
				Tags:        []string{"posts"},
				Security:    bearer(),
				Parameters: []Parameter{
					queryParam("workspace_id", "", true),
					queryParam("start", "RFC3339 start date e.g. 2025-01-01T00:00:00Z", true),
					queryParam("end", "RFC3339 end date", true),
				},
				Responses: map[string]Response{
					"200": jsonResponse("OK", arr(ref("Post"))),
				},
			},
		},

		"/posts/{id}/schedule": {
			Post: &Operation{
				OperationID: "schedulePost",
				Summary:     "Schedule a draft post",
				Tags:        []string{"posts"},
				Security:    bearer(),
				Parameters:  []Parameter{pathParam("id", "Post ID")},
				RequestBody: jsonBody(Schema{
					Type:     "object",
					Required: []string{"scheduled_at"},
					Properties: map[string]Schema{
						"scheduled_at": ts("Must be in the future"),
					},
				}),
				Responses: map[string]Response{
					"200": jsonResponse("Scheduled", ref("Post")),
					"400": errResponse("scheduled_at must be in the future"),
					"404": errResponse("Post not found"),
				},
			},
		},

		"/posts/{id}": {
			Delete: &Operation{
				OperationID: "deletePost",
				Summary:     "Delete a post",
				Tags:        []string{"posts"},
				Security:    bearer(),
				Parameters:  []Parameter{pathParam("id", "Post ID")},
				Responses: map[string]Response{
					"204": {Description: "Deleted"},
					"404": errResponse("Not found"),
				},
			},
		},

		// ── Media ──────────────────────────────────────────────────────────

		"/media": {
			Post: &Operation{
				OperationID: "uploadMedia",
				Summary:     "Upload a file to the media library",
				Tags:        []string{"media"},
				Security:    bearer(),
				Parameters:  []Parameter{queryParam("workspace_id", "", true)},
				RequestBody: &RequestBody{
					Required: true,
					Content: map[string]MediaType{
						"multipart/form-data": {
							Schema: Schema{
								Type: "object",
								Properties: map[string]Schema{
									"file":   {Type: "string", Format: "binary"},
									"folder": str("Optional folder name"),
								},
								Required: []string{"file"},
							},
						},
					},
				},
				Responses: map[string]Response{
					"201": jsonResponse("Uploaded", ref("MediaFile")),
					"400": errResponse("File too large (max 100 MB)"),
				},
			},
			Get: &Operation{
				OperationID: "listMedia",
				Summary:     "List media files in a workspace",
				Tags:        []string{"media"},
				Security:    bearer(),
				Parameters: []Parameter{
					queryParam("workspace_id", "", true),
					queryParam("folder", "Filter by folder", false),
				},
				Responses: map[string]Response{
					"200": jsonResponse("OK", arr(ref("MediaFile"))),
				},
			},
		},

		"/media/{id}": {
			Delete: &Operation{
				OperationID: "deleteMedia",
				Summary:     "Delete a media file",
				Tags:        []string{"media"},
				Security:    bearer(),
				Parameters:  []Parameter{pathParam("id", "Media file ID")},
				Responses: map[string]Response{
					"204": {Description: "Deleted"},
					"404": errResponse("Not found"),
				},
			},
		},

		// ── AI ─────────────────────────────────────────────────────────────

		"/ai/caption": {
			Post: &Operation{
				OperationID: "generateCaption",
				Summary:     "Generate an AI caption (GPT-4o)",
				Tags:        []string{"ai"},
				Security:    bearer(),
				RequestBody: jsonBody(Schema{
					Type:     "object",
					Required: []string{"topic", "platform"},
					Properties: map[string]Schema{
						"topic":    str("What the post is about"),
						"platform": {Type: "string", Enum: []string{"instagram", "facebook", "linkedin", "twitter", "tiktok", "youtube"}},
						"tone":     {Type: "string", Enum: []string{"casual", "professional", "funny", "inspirational"}, Description: "Defaults to casual"},
						"keywords": arr(str("Keyword or theme to incorporate")),
						"language": {Type: "string", Description: "ISO 639-1 code. Defaults to en", Example: "en"},
					},
				}),
				Responses: map[string]Response{
					"200": jsonResponse("Generated caption", ref("CaptionResult")),
					"500": errResponse("OpenAI error"),
				},
			},
		},

		// ── API Keys ───────────────────────────────────────────────────────

		"/api-keys": {
			Post: &Operation{
				OperationID: "createAPIKey",
				Summary:     "Create an API key — plaintext returned once, save it immediately",
				Tags:        []string{"api-keys"},
				Security:    bearer(),
				RequestBody: jsonBody(Schema{
					Type:     "object",
					Required: []string{"name"},
					Properties: map[string]Schema{
						"name":       str("Human-readable label, e.g. 'CI pipeline'"),
						"scopes":     arr(str("Scope string. Omit for full access (*)")),
						"expires_at": ts("Omit for non-expiring key"),
					},
				}),
				Responses: map[string]Response{
					"201": jsonResponse("Created — save the key now", ref("APIKeyCreated")),
					"400": errResponse("Invalid scope"),
				},
			},
			Get: &Operation{
				OperationID: "listAPIKeys",
				Summary:     "List API keys for the current user",
				Tags:        []string{"api-keys"},
				Security:    bearer(),
				Responses: map[string]Response{
					"200": jsonResponse("OK", arr(ref("APIKey"))),
				},
			},
		},

		"/api-keys/scopes": {
			Get: &Operation{
				OperationID: "listAPIKeyScopes",
				Summary:     "List all valid scope strings",
				Tags:        []string{"api-keys"},
				Security:    bearer(),
				Responses: map[string]Response{
					"200": jsonResponse("OK", arr(str("Scope string"))),
				},
			},
		},

		"/api-keys/{id}": {
			Delete: &Operation{
				OperationID: "revokeAPIKey",
				Summary:     "Revoke an API key",
				Tags:        []string{"api-keys"},
				Security:    bearer(),
				Parameters:  []Parameter{pathParam("id", "API key ID")},
				Responses: map[string]Response{
					"204": {Description: "Revoked"},
					"403": errResponse("Not your key"),
					"404": errResponse("Not found"),
				},
			},
		},

		// ── Billing ────────────────────────────────────────────────────────

		"/billing/checkout": {
			Post: &Operation{
				OperationID: "createCheckout",
				Summary:     "Create Stripe Checkout session for upgrading",
				Tags:        []string{"billing"},
				Security:    bearer(),
				RequestBody: jsonBody(Schema{
					Type:     "object",
					Required: []string{"org_id", "tier"},
					Properties: map[string]Schema{
						"org_id": uuid_(""),
						"tier":   {Type: "string", Enum: []string{"pro", "agency"}},
					},
				}),
				Responses: map[string]Response{
					"200": jsonResponse("Checkout URL", Schema{
						Type:       "object",
						Properties: map[string]Schema{"url": str("Stripe Checkout URL")},
					}),
				},
			},
		},

		"/billing/portal": {
			Post: &Operation{
				OperationID: "createPortal",
				Summary:     "Create Stripe Billing Portal session",
				Tags:        []string{"billing"},
				Security:    bearer(),
				RequestBody: jsonBody(Schema{
					Type:     "object",
					Required: []string{"org_id"},
					Properties: map[string]Schema{
						"org_id": uuid_(""),
					},
				}),
				Responses: map[string]Response{
					"200": jsonResponse("Portal URL", Schema{
						Type:       "object",
						Properties: map[string]Schema{"url": str("Stripe Billing Portal URL")},
					}),
					"404": errResponse("No active subscription"),
				},
			},
		},

		"/billing/webhook": {
			Post: &Operation{
				OperationID: "stripeWebhook",
				Summary:     "Stripe webhook receiver (internal — do not call directly)",
				Tags:        []string{"billing"},
				Parameters: []Parameter{
					{Name: "Stripe-Signature", In: "header", Required: true, Schema: str("Stripe webhook signature")},
				},
				Responses: map[string]Response{
					"200": {Description: "Acknowledged"},
				},
			},
		},
	}
}
