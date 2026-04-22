# Allposty — Project State

**Last updated:** 2026-04-22
**Phase:** Planning → M1 Foundation

---

## Decisions

| Date | Decision | Reason |
|---|---|---|
| 2026-04-22 | Backend: Go + Fiber v2 | Informed choice — no brightbean provider reuse, accepted trade-off, better long-term foundation |
| 2026-04-22 | Separate repos (not monorepo) | User preference: likes to separate concerns. Contract = OpenAPI spec |
| 2026-04-22 | AGPL-3.0 license | Aligned with brightbean-studio + postiz-app reference projects |
| 2026-04-22 | Job queue: Asynq (Redis) | Simpler than Temporal, well-supported Go-native queue |
| 2026-04-22 | Media storage: Cloudflare R2 | Free egress, S3-compatible, natural fit with Cloudflare Pages hosting |
| 2026-04-22 | Auth: JWT (backend) + better-auth (frontend) | Frontend boilerplate already uses better-auth |
| 2026-04-22 | AI captions in v1, image gen in v2 | User priority: text AI first, image AI second |
| 2026-04-22 | Public REST API from day 1 | Frontend is just a client — clean API-first design |
| 2026-04-22 | Twitter/X in v1 | Explicitly requested; note: OAuth 1.0a + v2 hybrid is complex |
| 2026-04-22 | v1 platforms: Instagram, Facebook, TikTok, YouTube, Twitter/X, LinkedIn | Core 6. Bluesky/Threads/Pinterest/Mastodon deferred to v2 |

---

## Blockers

None yet.

---

## Lessons

- brightbean-studio providers can't be directly reused (Python → Go), but their OAuth flow logic, API endpoints, rate limit handling, and edge cases are excellent references. Study them when implementing Go providers.
- postiz-app's orchestrator (Temporal) is too heavy for MVP; Asynq is the right call at this scale.

---

## Todos

- [ ] Initialize `allposty-backend` Go repository
- [ ] Design database schema (ERD)
- [ ] Design provider interface in Go (`providers/base.go`)
- [ ] Set up Railway/Render project for backend deployment
- [ ] Set up Neon or Railway Postgres instance
- [ ] Set up Upstash Redis or Railway Redis

---

## Deferred Ideas

- White-label: per-workspace branding (Agency tier, M3)
- Client portal: magic-link access for clients to approve posts (M3)
- Zapier / N8N integrations (post-M2)
- RSS auto-post (future)
- Mobile app (future)
- Multi-language UI — i18n is already scaffolded in frontend, can be activated anytime

---

## Preferences

- Responses should be terse. No trailing summaries.
