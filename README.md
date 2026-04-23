# allposty-backend

Go + Fiber REST API for [Allposty](https://allposty.com) — a SaaS social media scheduling and management platform.

The frontend lives in a separate repo (`allposty-frontend`). The contract between them is the OpenAPI spec served at `GET /api/v1/openapi.json`.

---

## Tech stack

| Layer | Technology |
|---|---|
| Language | Go 1.22+ |
| HTTP framework | Fiber v2 |
| Database | PostgreSQL (GORM + pgx) |
| Job queue | Redis + Asynq |
| Media storage | Cloudflare R2 (S3-compatible) |
| Auth | JWT (HS256) + opaque refresh tokens |
| Billing | Stripe Checkout + webhooks |
| AI | OpenAI GPT-4o (caption generation) |
| Hosting | Railway / Render / Fly.io |

---

## Project structure

```
cmd/
  api/        # HTTP server entry point
  worker/     # Asynq background worker entry point
  seed/       # Development seed data

internal/
  auth/       # JWT signing + parsing
  config/     # Viper-based env config
  database/   # GORM connect + auto-migrate
  handlers/   # HTTP handlers, one package per resource
    ai/
    auth/
    billing/
    media/
    orgs/
    posts/
    social/
  jobs/       # Asynq task definitions + handlers
  middleware/ # JWT auth, plan limit enforcement
  models/     # GORM models
  openapi/    # Programmatic OpenAPI 3.0.3 spec builder
  providers/  # Social platform OAuth + publish interface
  repository/ # Database access layer
  services/   # Business logic
  storage/    # Credential encryption, R2 client, Redis state store

pkg/
  response/   # Uniform JSON response helpers
```

---

## Prerequisites

You need **PostgreSQL** and **Redis**. No Docker required for local development.

**Option A — cloud (recommended, zero install):**
- Postgres: [neon.tech](https://neon.tech) free tier → copy the connection string
- Redis: [upstash.com](https://upstash.com) free tier → copy the `REDIS_URL`

**Option B — local install:**

```bash
# macOS
brew install postgresql redis
brew services start postgresql redis

# Ubuntu / Debian
sudo apt install postgresql redis-server
sudo service postgresql start
sudo service redis start
createdb allposty
```

**Option C — Docker (optional):**

```bash
make docker-up
```

---

## Getting started

```bash
# 1. Clone
git clone https://github.com/allposty/allposty-backend.git
cd allposty-backend

# 2. Copy env and fill in values
cp .env.example .env

# 3. Install tools
go install github.com/air-verse/air@latest       # live reload
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 4. Download dependencies
go mod tidy

# 5. Seed development data
make seed

# 6. Start API (with live reload)
make dev

# 7. In a second terminal, start the background worker
make worker
```

The API is now running at `http://localhost:8080`.

---

## Environment variables

Copy `.env.example` to `.env` and fill in values. Required fields are marked below.

| Variable | Required | Description |
|---|---|---|
| `APP_ENV` | | `development` or `production` |
| `APP_PORT` | | HTTP port (default `8080`) |
| `APP_SECRET` | **yes** | 32+ char random string — used to derive AES-256 key for credential encryption |
| `DATABASE_URL` | **yes** | PostgreSQL connection string |
| `REDIS_URL` | **yes** | Redis connection URL |
| `JWT_SECRET` | **yes** | Random string for JWT signing |
| `JWT_ACCESS_TTL` | | Access token TTL (default `15m`) |
| `JWT_REFRESH_TTL` | | Refresh token TTL (default `168h`) |
| `R2_ACCOUNT_ID` | | Cloudflare account ID |
| `R2_ACCESS_KEY_ID` | | R2 API token key ID |
| `R2_SECRET_ACCESS_KEY` | | R2 API token secret |
| `R2_BUCKET` | | R2 bucket name |
| `R2_PUBLIC_URL` | | Public CDN URL for the bucket |
| `OPENAI_API_KEY` | | Required for AI caption generation |
| `STRIPE_SECRET_KEY` | | Stripe secret key (`sk_test_...` for dev) |
| `STRIPE_WEBHOOK_SECRET` | | Stripe webhook signing secret (`whsec_...`) |
| `STRIPE_PRICE_PRO` | | Stripe price ID for the Pro tier |
| `STRIPE_PRICE_AGENCY` | | Stripe price ID for the Agency tier |
| `FACEBOOK_APP_ID` | | Meta app ID (Instagram + Facebook) |
| `FACEBOOK_APP_SECRET` | | Meta app secret |
| `LINKEDIN_CLIENT_ID` | | LinkedIn OAuth client ID |
| `LINKEDIN_CLIENT_SECRET` | | LinkedIn OAuth client secret |
| `TWITTER_CLIENT_ID` | | Twitter/X OAuth 2.0 client ID |
| `TWITTER_CLIENT_SECRET` | | Twitter/X OAuth 2.0 client secret |
| `TIKTOK_CLIENT_KEY` | | TikTok client key |
| `TIKTOK_CLIENT_SECRET` | | TikTok client secret |
| `GOOGLE_CLIENT_ID` | | Google OAuth client ID (YouTube) |
| `GOOGLE_CLIENT_SECRET` | | Google OAuth client secret |
| `FRONTEND_URL` | **yes** | Frontend origin for CORS (e.g. `http://localhost:5173`) |

---

## Make targets

```bash
make dev            # Run API with live reload (air)
make run            # Run API directly
make worker         # Run background worker
make seed           # Seed development database

make build          # Build ./bin/api
make build-worker   # Build ./bin/worker

make test           # Run tests with race detector
make test-coverage  # Run tests + open HTML coverage report
make lint           # Run golangci-lint
make tidy           # go mod tidy

make docker-up      # Start Postgres + Redis via Docker Compose
make docker-down    # Stop Docker Compose services
```

---

## Seed accounts

Running `make seed` creates three users, one per plan tier. The database is seeded idempotently — re-running is safe.

| Email | Password | Plan | Org | Workspace |
|---|---|---|---|---|
| `free@allposty.dev` | `Seed1234!` | Free | Freddy's Brand | Main Workspace |
| `pro@allposty.dev` | `Seed1234!` | Pro | Paula Studio | Client Work |
| `agency@allposty.dev` | `Seed1234!` | Agency | Andy Agency Co | Operations |

Each workspace is seeded with three posts (one draft, one scheduled 24 h from now, one published 48 h ago).

---

## API overview

All protected endpoints require `Authorization: Bearer <access_token>`.

The full machine-readable spec is at `GET /api/v1/openapi.json` — use this to generate the TypeScript client:

```bash
# from allposty-frontend/
bun run gen:api
```

### Auth

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/auth/register` | Register |
| `POST` | `/api/v1/auth/login` | Login → returns `access_token` + `refresh_token` |
| `POST` | `/api/v1/auth/refresh` | Rotate tokens |
| `POST` | `/api/v1/auth/logout` | Revoke refresh token |
| `GET` | `/api/v1/auth/me` | Current user |

### Organizations & workspaces

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/orgs` | Create organization |
| `GET` | `/api/v1/orgs` | List user's organizations |
| `GET` | `/api/v1/orgs/:org_id` | Get organization |
| `POST` | `/api/v1/orgs/:org_id/workspaces` | Create workspace (plan limit enforced) |
| `GET` | `/api/v1/orgs/:org_id/workspaces` | List workspaces |

### Social accounts

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/social/connect/:platform` | Start OAuth flow → redirects to platform |
| `GET` | `/api/v1/social/callback/:platform` | OAuth callback (public, platform redirects here) |
| `GET` | `/api/v1/social/accounts` | List connected accounts for a workspace |
| `DELETE` | `/api/v1/social/accounts/:id` | Disconnect account |

Supported platforms: `instagram`, `facebook`, `linkedin`, `twitter`, `tiktok`, `youtube`

### Posts

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/posts` | Create post |
| `GET` | `/api/v1/posts` | List posts for a workspace |
| `GET` | `/api/v1/posts/calendar` | Posts in a date range (calendar view) |
| `POST` | `/api/v1/posts/:id/schedule` | Schedule a post |
| `DELETE` | `/api/v1/posts/:id` | Delete post |

### Media library

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/media` | Upload file → stored in R2 |
| `GET` | `/api/v1/media` | List media files for a workspace |
| `DELETE` | `/api/v1/media/:id` | Delete media file |

### AI

| Method | Path | Description | Plan |
|---|---|---|---|
| `POST` | `/api/v1/ai/caption` | Generate caption from topic + platform + tone | Pro / Agency |

### API keys

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/api-keys` | Create key — plaintext returned once, never stored again |
| `GET` | `/api/v1/api-keys` | List your keys (prefix + metadata, no plaintext) |
| `GET` | `/api/v1/api-keys/scopes` | List valid scope strings |
| `DELETE` | `/api/v1/api-keys/:id` | Revoke a key |

**Authentication:** send the key as `Authorization: Bearer allposty_<key>` — same header as JWT, detected by prefix.

**Scopes:** `*` (full access), `posts:read`, `posts:write`, `social:read`, `media:read`, `media:write`, `ai:write`

**Rate limits (requests/minute):**

| Plan | Limit |
|---|---|
| Free | 60 |
| Pro | 300 |
| Agency | 1000 |

Rate limit headers are returned on every API key request: `X-RateLimit-Limit`, `X-RateLimit-Remaining`. Exceeding the limit returns HTTP `429`.

### Billing

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/billing/checkout` | Create Stripe Checkout session |
| `POST` | `/api/v1/billing/portal` | Create Stripe Billing Portal session |
| `POST` | `/api/v1/billing/webhook` | Stripe webhook receiver (public, signature-verified) |

---

## Plan limits (server-enforced)

| Tier | Price | Workspaces | Social accounts | Posts/month | AI |
|---|---|---|---|---|---|
| Free | $0 | 1 | 3 | 30 | No |
| Pro | ~$19/mo | 3 | 15 | Unlimited | Yes |
| Agency | ~$79/mo | Unlimited | Unlimited | Unlimited | Yes |

Limits are enforced via route-level middleware (`internal/middleware/plan.go`). Exceeding a limit returns HTTP `402 Payment Required`.

---

## OAuth flows

Each platform uses standard OAuth 2.0. Twitter/X uses OAuth 2.0 + PKCE (S256). The PKCE verifier is stored in Redis (10-minute TTL, one-time pop) keyed by the OAuth state token.

```
Browser                     API                         Platform
  │                          │                              │
  ├─ GET /social/connect ───►│                              │
  │                          ├─ generate state + PKCE ─────┤
  │                          ├─ store in Redis              │
  │◄─ 302 redirect ──────────┤                              │
  │                          │                              │
  ├─────────────────────────────────────────────────────────►│ (user approves)
  │◄─────────────────────────────────────────────────────────┤
  │                          │                              │
  ├─ GET /social/callback ──►│                              │
  │                          ├─ pop state from Redis        │
  │                          ├─ exchange code + verifier ──►│
  │                          ├─ fetch profile               │
  │                          ├─ encrypt + upsert account    │
  │◄─ 302 /apps?connected= ──┤                              │
```

OAuth credentials are encrypted at rest using AES-256-GCM. The key is derived from `APP_SECRET` via SHA-256.

---

## Background worker

The worker (`cmd/worker`) runs as a separate process and handles scheduled post publishing via Asynq queues.

```
Queues:  critical (weight 6) | default (weight 3) | low (weight 1)
Concurrency: 10 goroutines
```

When a post is scheduled via `POST /api/v1/posts/:id/schedule`, an Asynq task is enqueued to fire at the scheduled time. The worker picks it up, decrypts the platform credentials, and calls the appropriate provider's `Publish()` method.

---

## CI

GitHub Actions runs on every push and pull request:

| Job | What it does |
|---|---|
| `lint` | golangci-lint |
| `build` | `go build ./...` for both binaries |
| `test` | `go test -race -cover ./...` with Postgres + Redis service containers |

---

## License

[AGPL-3.0](LICENSE)
