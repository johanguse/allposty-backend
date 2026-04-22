package openapi

// schemas.go — reusable component schemas referenced via $ref.

func buildSchemas() map[string]Schema {
	return map[string]Schema{

		// ── Auth ───────────────────────────────────────────────────────────

		"TokenPair": {
			Type: "object",
			Properties: map[string]Schema{
				"access_token":  str("Short-lived JWT access token"),
				"refresh_token": str("Long-lived opaque refresh token"),
				"expires_in":    {Type: "integer", Description: "Access token TTL in seconds"},
			},
			Required: []string{"access_token", "refresh_token", "expires_in"},
		},

		"User": {
			Type: "object",
			Properties: map[string]Schema{
				"id":             uuid_("User ID"),
				"email":          str("Email address"),
				"name":           str("Display name"),
				"avatar_url":     str("Profile picture URL"),
				"plan_tier":      {Type: "string", Enum: []string{"free", "pro", "agency"}},
				"email_verified": {Type: "boolean"},
				"created_at":     ts(""),
				"updated_at":     ts(""),
			},
		},

		// ── Orgs ───────────────────────────────────────────────────────────

		"Organization": {
			Type: "object",
			Properties: map[string]Schema{
				"id":         uuid_(""),
				"name":       str(""),
				"slug":       str("URL-safe identifier"),
				"logo_url":   str(""),
				"owner_id":   uuid_("Owner user ID"),
				"created_at": ts(""),
			},
		},

		"Workspace": {
			Type: "object",
			Properties: map[string]Schema{
				"id":              uuid_(""),
				"organization_id": uuid_(""),
				"name":            str(""),
				"slug":            str(""),
				"description":     str(""),
				"created_at":      ts(""),
			},
		},

		"WorkspaceMember": {
			Type: "object",
			Properties: map[string]Schema{
				"id":           uuid_(""),
				"workspace_id": uuid_(""),
				"user_id":      uuid_(""),
				"role":         {Type: "string", Enum: []string{"owner", "admin", "member"}},
				"user":         ref("User"),
			},
		},

		// ── Social ─────────────────────────────────────────────────────────

		"SocialAccount": {
			Type: "object",
			Properties: map[string]Schema{
				"id":               uuid_(""),
				"workspace_id":     uuid_(""),
				"platform":         {Type: "string", Enum: []string{"instagram", "facebook", "linkedin", "twitter", "tiktok", "youtube"}},
				"platform_user_id": str(""),
				"name":             str("Display name on the platform"),
				"username":         str(""),
				"avatar_url":       str(""),
				"created_at":       ts(""),
			},
		},

		// ── Posts ──────────────────────────────────────────────────────────

		"Post": {
			Type: "object",
			Properties: map[string]Schema{
				"id":           uuid_(""),
				"workspace_id": uuid_(""),
				"caption":      str("Main post text"),
				"media_urls":   arr(str("Public media URL")),
				"status":       {Type: "string", Enum: []string{"draft", "scheduled", "published", "failed"}},
				"scheduled_at": ts("When to publish"),
				"published_at": ts("When it was published"),
				"platforms":    arr(ref("PostPlatform")),
				"created_at":   ts(""),
			},
		},

		"PostPlatform": {
			Type: "object",
			Properties: map[string]Schema{
				"id":                uuid_(""),
				"post_id":           uuid_(""),
				"social_account_id": uuid_(""),
				"platform":          str(""),
				"caption_override":  str("Platform-specific caption override"),
				"status":            str(""),
				"platform_post_id":  str("ID returned by the platform after publishing"),
				"published_at":      ts(""),
				"error_message":     str(""),
			},
		},

		// ── Media ──────────────────────────────────────────────────────────

		"MediaFile": {
			Type: "object",
			Properties: map[string]Schema{
				"id":           uuid_(""),
				"workspace_id": uuid_(""),
				"uploaded_by":  uuid_("User who uploaded"),
				"filename":     str("Original filename"),
				"mime_type":    str("e.g. image/jpeg"),
				"size_bytes":   {Type: "integer"},
				"url":          str("Public CDN URL"),
				"folder":       str("Optional folder path"),
				"created_at":   ts(""),
			},
		},

		// ── AI ─────────────────────────────────────────────────────────────

		"CaptionResult": {
			Type: "object",
			Properties: map[string]Schema{
				"caption":  str("Generated caption text"),
				"hashtags": str("Space-separated hashtag string"),
			},
			Required: []string{"caption"},
		},

		// ── Billing ────────────────────────────────────────────────────────

		"Subscription": {
			Type: "object",
			Properties: map[string]Schema{
				"organization_id":     uuid_(""),
				"tier":                {Type: "string", Enum: []string{"free", "pro", "agency"}},
				"status":              str("active | canceled | past_due"),
				"current_period_end":  {Type: "integer", Description: "Unix timestamp"},
			},
		},

		// ── Shared ─────────────────────────────────────────────────────────

		"Error": {
			Type:     "object",
			Required: []string{"error"},
			Properties: map[string]Schema{
				"error": str("Human-readable error message"),
			},
		},
	}
}
