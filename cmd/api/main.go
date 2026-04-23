package main

import (
	"log"

	"github.com/allposty/allposty-backend/internal/config"
	"github.com/allposty/allposty-backend/internal/database"
	aihandler "github.com/allposty/allposty-backend/internal/handlers/ai"
	apikeyhandler "github.com/allposty/allposty-backend/internal/handlers/apikeys"
	authhandler "github.com/allposty/allposty-backend/internal/handlers/auth"
	billinghandler "github.com/allposty/allposty-backend/internal/handlers/billing"
	mediahandler "github.com/allposty/allposty-backend/internal/handlers/media"
	orghandler "github.com/allposty/allposty-backend/internal/handlers/orgs"
	posthandler "github.com/allposty/allposty-backend/internal/handlers/posts"
	socialhandler "github.com/allposty/allposty-backend/internal/handlers/social"
	"github.com/allposty/allposty-backend/internal/middleware"
	"github.com/allposty/allposty-backend/internal/openapi"
	"github.com/allposty/allposty-backend/internal/providers"
	"github.com/allposty/allposty-backend/internal/repository"
	"github.com/allposty/allposty-backend/internal/services"
	"github.com/allposty/allposty-backend/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	zapLog, _ := zap.NewProduction()
	if cfg.App.Env == "development" {
		zapLog, _ = zap.NewDevelopment()
	}
	defer zapLog.Sync()

	// Database
	db, err := database.Connect(cfg.Database.URL, zapLog)
	if err != nil {
		zapLog.Fatal("database connect", zap.Error(err))
	}
	if err := database.Migrate(db); err != nil {
		zapLog.Fatal("database migrate", zap.Error(err))
	}

	// Asynq client
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Redis.URL})
	defer asynqClient.Close()

	// R2 storage
	r2, err := storage.NewR2Client(cfg)
	if err != nil {
		zapLog.Fatal("r2 client", zap.Error(err))
	}

	// Repositories
	userRepo := repository.NewUserRepository(db)
	orgRepo := repository.NewOrgRepository(db)
	socialRepo := repository.NewSocialRepository(db)
	postRepo := repository.NewPostRepository(db)
	mediaRepo := repository.NewMediaRepository(db)
	subRepo := repository.NewSubscriptionRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)

	// Shared deps
	providerRegistry := providers.NewRegistry(cfg)
	credStore := storage.NewCredentialStore(cfg.App.Secret)
	stateStore, err := storage.NewStateStore(cfg.Redis.URL)
	if err != nil {
		zapLog.Fatal("state store", zap.Error(err))
	}
	rateLimiter, err := storage.NewRateLimiter(cfg.Redis.URL)
	if err != nil {
		zapLog.Fatal("rate limiter", zap.Error(err))
	}

	// Services
	authSvc := services.NewAuthService(userRepo, cfg)
	orgSvc := services.NewOrgService(orgRepo)
	postSvc := services.NewPostService(postRepo, socialRepo, orgSvc, providerRegistry, credStore, asynqClient)
	mediaSvc := services.NewMediaService(mediaRepo, orgSvc, r2)
	aiSvc := services.NewAIService(cfg.OpenAI.APIKey)
	billingSvc := services.NewBillingService(orgRepo, userRepo, subRepo, cfg)
	apiKeySvc := services.NewAPIKeyService(apiKeyRepo)

	// Handlers
	authH := authhandler.NewHandler(authSvc, userRepo)
	orgH := orghandler.NewHandler(orgSvc)
	socialH := socialhandler.NewHandler(providerRegistry, socialRepo, orgSvc, credStore, stateStore, cfg.Frontend)
	postH := posthandler.NewHandler(postSvc)
	mediaH := mediahandler.NewHandler(mediaSvc)
	aiH := aihandler.NewHandler(aiSvc)
	billingH := billinghandler.NewHandler(billingSvc, cfg.Frontend)
	apiKeyH := apikeyhandler.NewHandler(apiKeySvc)

	// Fiber
	app := fiber.New(fiber.Config{
		AppName:      "allposty-api",
		BodyLimit:    110 * 1024 * 1024, // 110 MB — media uploads
		ErrorHandler: errorHandler,
	})

	app.Use(recover.New())
	app.Use(fiberlogger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.Frontend,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		AllowCredentials: true,
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "version": "1.0.0"})
	})

	port := cfg.App.Port
	if port == "" {
		port = "8080"
	}

	// OpenAPI spec — consumed by frontend's `bun run gen:api`
	app.Get("/api/v1/openapi.json", func(c *fiber.Ctx) error {
		serverURL := "https://" + c.Hostname() + "/api/v1"
		if cfg.App.Env == "development" {
			serverURL = "http://localhost:" + port + "/api/v1"
		}
		return c.JSON(openapi.Build(serverURL))
	})

	v1 := app.Group("/api/v1")

	// ── Public ──────────────────────────────────────────────────────────────
	authGroup := v1.Group("/auth")
	authGroup.Post("/register", authH.Register)
	authGroup.Post("/login", authH.Login)
	authGroup.Post("/refresh", authH.Refresh)
	authGroup.Post("/logout", authH.Logout)

	// OAuth callbacks come back from social platforms — must be public
	v1.Get("/social/callback/:platform", socialH.Callback)

	// Stripe webhook — public but signature-verified inside handler
	v1.Post("/billing/webhook", billingH.Webhook)

	// ── Protected ───────────────────────────────────────────────────────────
	authMiddleware := middleware.JWT(cfg.JWT.Secret, apiKeySvc, userRepo, rateLimiter)
	p := v1.Group("", authMiddleware)

	// Auth
	p.Get("/auth/me", authH.Me)

	// Orgs & workspaces
	p.Post("/orgs", orgH.CreateOrg)
	p.Get("/orgs", orgH.ListOrgs)
	p.Get("/orgs/:org_id", orgH.GetOrg)
	p.Post("/orgs/:org_id/workspaces", middleware.RequireWorkspaceSlot(userRepo, orgRepo), orgH.CreateWorkspace)
	p.Get("/orgs/:org_id/workspaces", orgH.ListWorkspaces)

	// Social accounts
	p.Get("/social/connect/:platform", middleware.RequireSocialSlot(userRepo, socialRepo), socialH.Connect)
	p.Get("/social/accounts", socialH.ListAccounts)
	p.Delete("/social/accounts/:id", socialH.Disconnect)

	// Posts & calendar
	p.Post("/posts", postH.CreatePost)
	p.Get("/posts", postH.ListPosts)
	p.Get("/posts/calendar", postH.Calendar)
	p.Post("/posts/:id/schedule", postH.SchedulePost)
	p.Delete("/posts/:id", postH.DeletePost)

	// Media library
	p.Post("/media", mediaH.Upload)
	p.Get("/media", mediaH.List)
	p.Delete("/media/:id", mediaH.Delete)

	// AI
	p.Post("/ai/caption", middleware.RequireAI(userRepo), aiH.GenerateCaption)

	// API keys
	p.Post("/api-keys", apiKeyH.Create)
	p.Get("/api-keys", apiKeyH.List)
	p.Get("/api-keys/scopes", apiKeyH.Scopes)
	p.Delete("/api-keys/:id", apiKeyH.Revoke)

	// Billing
	p.Post("/billing/checkout", billingH.CreateCheckout)
	p.Post("/billing/portal", billingH.CreatePortal)

	// ── Boot ────────────────────────────────────────────────────────────────
	zapLog.Info("allposty api starting",
		zap.String("port", port),
		zap.String("env", cfg.App.Env),
	)
	if err := app.Listen(":" + port); err != nil {
		zapLog.Fatal("server error", zap.Error(err))
	}
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	return c.Status(code).JSON(fiber.Map{"error": err.Error()})
}
