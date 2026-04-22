# Allposty Roadmap

**Current Milestone:** M2 — AI + Monetization
**Status:** In Progress

---

## M1 — Foundation
**Goal:** Backend boots, auth works, one social account connects and a post publishes. Internal only, no billing yet.
**Target:** Working end-to-end with Instagram before moving to M2.

### Features

**Project bootstrap** — COMPLETE
- Go/Fiber project scaffold with folder structure
- PostgreSQL schema + migrations (GORM)
- Redis + Asynq job queue wired up
- Docker Compose for local dev (Postgres, Redis)
- OpenAPI spec generation middleware (swaggo or fiber-swagger)
- CI: GitHub Actions (lint, test, build)
- Cloudflare R2 media upload integration

**Auth** — COMPLETE
- JWT-based auth (register, login, refresh, logout)
- better-auth frontend integration
- Email/password + Google OAuth
- Middleware: route protection, org/workspace context injection

**Multi-tenancy core** — COMPLETE
- Organizations → Workspaces → Members model
- RBAC: Owner / Admin / Member roles
- Invite by email flow
- Data isolation: all queries scoped to org_id

**Social account connections** — COMPLETE
- Generic OAuth provider interface in Go (designed for easy extension)
- Instagram (Graph API via Facebook OAuth)
- Facebook (Pages)
- LinkedIn (Personal + Company)
- Twitter/X (OAuth 1.0a + v2)
- TikTok (Content Posting API)
- YouTube (Google OAuth + YouTube Data API v3)
- Encrypted credential storage (AES-256)
- Token refresh logic per provider

**Post composer** — COMPLETE
- Create / edit / delete posts
- Per-platform text + media overrides
- Draft → Scheduled → Published → Failed state machine
- Media upload to R2, attach to post
- Schedule datetime picker

**Scheduling engine** — COMPLETE
- Asynq worker that polls and dispatches publishing jobs
- Per-platform publish functions (one per provider)
- Retry logic with exponential backoff
- Publish audit log (last 90 days)

**Content calendar** — COMPLETE
- API: posts by date range, by workspace
- Frontend: visual monthly/weekly calendar view

**Media library** — COMPLETE
- Org + workspace scoped folders
- Upload, rename, delete
- Reuse media across posts

---

## M2 — AI + Monetization
**Goal:** AI captions live, Stripe billing enforced, ready for public beta launch.

### Features

**AI caption generation** — COMPLETE
- OpenAI GPT-4o integration
- Generate caption from: topic, platform, tone, keywords
- Regenerate / edit inline in composer
- Usage metered per plan tier

**Stripe billing** — COMPLETE
- Subscription tiers: Free / Pro / Agency
- Checkout flow (Stripe Checkout or embedded)
- Webhook handler: subscription created / updated / cancelled
- Plan limits enforced server-side (accounts, workspaces, posts/month)
- Billing portal (manage subscription, invoices)
- Usage dashboard in frontend

**Public API + SDK** — PLANNED
- `/api/v1/` versioned REST API (already internal in M1 — expose + document)
- API key management (generate, revoke, scoped permissions)
- OpenAPI spec published at `/api/v1/openapi.json`
- TypeScript SDK generated from spec (hey-api)
- Rate limiting per API key

**Onboarding flow** — PLANNED
- Step-by-step: create org → connect first social account → create first post
- Frontend onboarding route (already scaffolded)

---

## M3 — Power Features
**Goal:** Retention features that justify Agency tier pricing.

### Features

**Social inbox** — PLANNED
- Comments, mentions, DMs from connected platforms
- Unified inbox view
- Reply inline
- Sentiment tagging

**Approval workflows** — PLANNED
- Configurable: none / internal / internal + client
- Threaded comments on posts
- Client portal (magic link, no login required)
- Approval audit trail

**Analytics** — PLANNED
- Post performance per platform (impressions, reach, engagement)
- Workspace-level aggregate view
- Charts (recharts frontend, already available)

**Additional platforms** — PLANNED
- Bluesky
- Threads
- Pinterest
- Mastodon
- (Google Business Profile)

---

## M4 — AI Expansion
**Goal:** AI as a differentiator beyond caption writing.

### Features

**AI image generation** — PLANNED
- Generate post images from prompt (DALL-E 3 or Stable Diffusion)
- Save to media library
- Edit + regenerate

**AI content calendar** — PLANNED
- Suggest posting schedule based on platform best practices
- Auto-generate content ideas for a topic/niche

**AI analytics insights** — PLANNED
- Natural language summaries of performance
- Recommendations for best posting times

---

## Future Considerations

- Mobile app (React Native or PWA)
- N8N / Make.com integration nodes
- White-label (custom domain, logo, colors per workspace)
- Zapier integration
- RSS → auto-post
- Multi-language UI (i18n already scaffolded in frontend)
