package main

import (
	"context"
	"log"

	"github.com/allposty/allposty-backend/internal/config"
	"github.com/allposty/allposty-backend/internal/database"
	"github.com/allposty/allposty-backend/internal/jobs"
	"github.com/allposty/allposty-backend/internal/providers"
	"github.com/allposty/allposty-backend/internal/repository"
	"github.com/allposty/allposty-backend/internal/services"
	"github.com/allposty/allposty-backend/internal/storage"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	zapLog, _ := zap.NewProduction()
	defer zapLog.Sync()

	db, err := database.Connect(cfg.Database.URL, zapLog)
	if err != nil {
		zapLog.Fatal("database connect", zap.Error(err))
	}

	// Wire dependencies
	orgRepo := repository.NewOrgRepository(db)
	socialRepo := repository.NewSocialRepository(db)
	postRepo := repository.NewPostRepository(db)
	providerRegistry := providers.NewRegistry(cfg)
	credStore := storage.NewCredentialStore(cfg.App.Secret)
	orgSvc := services.NewOrgService(orgRepo)

	// Asynq client (worker also needs to enqueue retries/new tasks if needed)
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Redis.URL})
	defer asynqClient.Close()

	postSvc := services.NewPostService(postRepo, socialRepo, orgSvc, providerRegistry, credStore, asynqClient)

	// Asynq server
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.Redis.URL},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				zapLog.Error("task failed",
					zap.String("type", task.Type()),
					zap.Error(err),
				)
			}),
		},
	)

	mux := asynq.NewServeMux()

	publishHandler := jobs.NewPublishPostHandler(func(ctx context.Context, postID uuid.UUID) error {
		zapLog.Info("publishing post", zap.String("post_id", postID.String()))
		if err := postSvc.PublishNow(ctx, postID); err != nil {
			zapLog.Error("publish failed", zap.String("post_id", postID.String()), zap.Error(err))
			return err
		}
		zapLog.Info("post published", zap.String("post_id", postID.String()))
		return nil
	})

	mux.Handle(jobs.TypePublishPost, publishHandler)

	zapLog.Info("worker starting")
	if err := srv.Run(mux); err != nil {
		zapLog.Fatal("worker error", zap.Error(err))
	}
}
