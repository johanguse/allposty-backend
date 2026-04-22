// cmd/seed/main.go — Development seed data.
//
// Creates three users covering all plan tiers, each with an org, a workspace,
// and a few draft posts so the frontend has something to render.
//
// Usage:
//
//	go run ./cmd/seed
//
// Safe to run multiple times — it skips rows that already exist (checked by email).
package main

import (
	"log"
	"time"

	"github.com/allposty/allposty-backend/internal/config"
	"github.com/allposty/allposty-backend/internal/database"
	"github.com/allposty/allposty-backend/internal/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ── Seed accounts ────────────────────────────────────────────────────────────
//
// | Email                    | Password      | Plan   |
// |--------------------------|---------------|--------|
// | free@allposty.dev        | Seed1234!     | free   |
// | pro@allposty.dev         | Seed1234!     | pro    |
// | agency@allposty.dev      | Seed1234!     | agency |

type seedUser struct {
	name      string
	email     string
	plan      string
	orgName   string
	orgSlug   string
	wsName    string
	wsSlug    string
}

var users = []seedUser{
	{
		name:    "Free Freddy",
		email:   "free@allposty.dev",
		plan:    "free",
		orgName: "Freddy's Brand",
		orgSlug: "freddys-brand",
		wsName:  "Main Workspace",
		wsSlug:  "main",
	},
	{
		name:    "Pro Paula",
		email:   "pro@allposty.dev",
		plan:    "pro",
		orgName: "Paula Studio",
		orgSlug: "paula-studio",
		wsName:  "Client Work",
		wsSlug:  "client-work",
	},
	{
		name:    "Agency Andy",
		email:   "agency@allposty.dev",
		plan:    "agency",
		orgName: "Andy Agency Co",
		orgSlug: "andy-agency-co",
		wsName:  "Operations",
		wsSlug:  "operations",
	},
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg.Database.URL, nopLogger())
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("Seed1234!"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("bcrypt: %v", err)
	}

	for _, su := range users {
		if err := seedOne(db, su, string(hash)); err != nil {
			log.Fatalf("seed %s: %v", su.email, err)
		}
		log.Printf("seeded  %-30s  plan=%-6s", su.email, su.plan)
	}

	log.Println("done")
}

func seedOne(db *gorm.DB, su seedUser, passwordHash string) error {
	// ── User ─────────────────────────────────────────────────────────────────
	var user models.User
	result := db.Where("email = ?", su.email).First(&user)
	if result.Error != nil {
		user = models.User{
			Name:         su.name,
			Email:        su.email,
			PasswordHash: passwordHash,
			PlanTier:     su.plan,
		}
		if err := db.Create(&user).Error; err != nil {
			return err
		}
	}

	// ── Organization ─────────────────────────────────────────────────────────
	var org models.Organization
	result = db.Where("slug = ?", su.orgSlug).First(&org)
	if result.Error != nil {
		org = models.Organization{
			Name:    su.orgName,
			Slug:    su.orgSlug,
			OwnerID: user.ID,
		}
		if err := db.Create(&org).Error; err != nil {
			return err
		}
	}

	// ── Workspace ─────────────────────────────────────────────────────────────
	var ws models.Workspace
	result = db.Where("organization_id = ? AND slug = ?", org.ID, su.wsSlug).First(&ws)
	if result.Error != nil {
		ws = models.Workspace{
			OrganizationID: org.ID,
			Name:           su.wsName,
			Slug:           su.wsSlug,
		}
		if err := db.Create(&ws).Error; err != nil {
			return err
		}
	}

	// ── WorkspaceMember (owner) ───────────────────────────────────────────────
	var member models.WorkspaceMember
	result = db.Where("workspace_id = ? AND user_id = ?", ws.ID, user.ID).First(&member)
	if result.Error != nil {
		member = models.WorkspaceMember{
			WorkspaceID: ws.ID,
			UserID:      user.ID,
			Role:        models.RoleOwner,
		}
		if err := db.Create(&member).Error; err != nil {
			return err
		}
	}

	// ── Draft posts ───────────────────────────────────────────────────────────
	seedPosts(db, ws.ID)

	return nil
}

func seedPosts(db *gorm.DB, workspaceID uuid.UUID) {
	now := time.Now().UTC()

	drafts := []models.Post{
		{
			WorkspaceID: workspaceID,
			Caption:     "Excited to share what we've been building — stay tuned! 🚀",
			MediaURLs:   pq.StringArray{},
			Status:      models.PostStatusDraft,
		},
		{
			WorkspaceID: workspaceID,
			Caption:     "5 tips for a consistent posting schedule that actually works.",
			MediaURLs:   pq.StringArray{},
			Status:      models.PostStatusScheduled,
			ScheduledAt: ptr(now.Add(24 * time.Hour)),
		},
		{
			WorkspaceID: workspaceID,
			Caption:     "Behind the scenes of our latest campaign — real results inside.",
			MediaURLs:   pq.StringArray{},
			Status:      models.PostStatusPublished,
			PublishedAt: ptr(now.Add(-48 * time.Hour)),
		},
	}

	for i := range drafts {
		// Skip if a post with identical caption already exists in this workspace.
		var count int64
		db.Model(&models.Post{}).
			Where("workspace_id = ? AND caption = ?", workspaceID, drafts[i].Caption).
			Count(&count)
		if count > 0 {
			continue
		}
		_ = db.Create(&drafts[i]).Error
	}
}

func ptr[T any](v T) *T { return &v }

// nopLogger returns a zap.Logger that discards all output — keeps seed output clean.
func nopLogger() *zap.Logger { return zap.NewNop() }
