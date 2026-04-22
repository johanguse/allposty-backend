package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/allposty/allposty-backend/internal/jobs"
	"github.com/allposty/allposty-backend/internal/models"
	"github.com/allposty/allposty-backend/internal/providers"
	"github.com/allposty/allposty-backend/internal/repository"
	"github.com/allposty/allposty-backend/internal/storage"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

var (
	ErrPostNotFound = errors.New("post not found")
)

type PostService struct {
	posts    *repository.PostRepository
	social   *repository.SocialRepository
	orgs     *OrgService
	registry *providers.Registry
	creds    *storage.CredentialStore
	asynq    *asynq.Client
}

func NewPostService(
	posts *repository.PostRepository,
	social *repository.SocialRepository,
	orgs *OrgService,
	registry *providers.Registry,
	creds *storage.CredentialStore,
	asynq *asynq.Client,
) *PostService {
	return &PostService{
		posts:    posts,
		social:   social,
		orgs:     orgs,
		registry: registry,
		creds:    creds,
		asynq:    asynq,
	}
}

type CreatePostInput struct {
	WorkspaceID     uuid.UUID
	Caption         string
	MediaURLs       []string
	SocialAccountIDs []uuid.UUID
	ScheduledAt     *time.Time
}

func (s *PostService) CreatePost(userID uuid.UUID, input CreatePostInput) (*models.Post, error) {
	if err := s.orgs.RequireWorkspaceAccess(input.WorkspaceID, userID); err != nil {
		return nil, ErrForbidden
	}

	post := &models.Post{
		WorkspaceID: input.WorkspaceID,
		Caption:     input.Caption,
		MediaURLs:   pq.StringArray(input.MediaURLs),
		Status:      models.PostStatusDraft,
		ScheduledAt: input.ScheduledAt,
	}

	for _, accountID := range input.SocialAccountIDs {
		account, err := s.social.FindByID(accountID)
		if err != nil {
			continue
		}
		post.Platforms = append(post.Platforms, models.PostPlatform{
			SocialAccountID: accountID,
			Platform:        account.Platform,
			Status:          models.PostStatusDraft,
		})
	}

	if err := s.posts.Create(post); err != nil {
		return nil, err
	}

	// If scheduled, enqueue the Asynq job
	if input.ScheduledAt != nil {
		if err := s.enqueuePublish(post.ID, *input.ScheduledAt); err != nil {
			return nil, err
		}
		_ = s.posts.UpdateStatus(post.ID, models.PostStatusScheduled, nil)
		post.Status = models.PostStatusScheduled
	}

	return post, nil
}

func (s *PostService) SchedulePost(userID, postID uuid.UUID, scheduledAt time.Time) (*models.Post, error) {
	post, err := s.posts.FindByID(postID)
	if err != nil {
		return nil, ErrPostNotFound
	}
	if err := s.orgs.RequireWorkspaceAccess(post.WorkspaceID, userID); err != nil {
		return nil, ErrForbidden
	}

	if err := s.enqueuePublish(postID, scheduledAt); err != nil {
		return nil, err
	}

	post.ScheduledAt = &scheduledAt
	post.Status = models.PostStatusScheduled
	_ = s.posts.Update(post)

	return post, nil
}

func (s *PostService) ListPosts(userID, workspaceID uuid.UUID, status *models.PostStatus) ([]models.Post, error) {
	if err := s.orgs.RequireWorkspaceAccess(workspaceID, userID); err != nil {
		return nil, ErrForbidden
	}
	return s.posts.FindByWorkspace(workspaceID, status)
}

func (s *PostService) GetCalendar(userID, workspaceID uuid.UUID, from, to time.Time) ([]models.Post, error) {
	if err := s.orgs.RequireWorkspaceAccess(workspaceID, userID); err != nil {
		return nil, ErrForbidden
	}
	return s.posts.FindByDateRange(workspaceID, from, to)
}

func (s *PostService) DeletePost(userID, postID uuid.UUID) error {
	post, err := s.posts.FindByID(postID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPostNotFound
		}
		return err
	}
	if err := s.orgs.RequireWorkspaceAccess(post.WorkspaceID, userID); err != nil {
		return ErrForbidden
	}
	return s.posts.Delete(postID)
}

// PublishNow is called by the Asynq worker — publishes to all platforms.
func (s *PostService) PublishNow(ctx context.Context, postID uuid.UUID) error {
	post, err := s.posts.FindByID(postID)
	if err != nil {
		return fmt.Errorf("post not found: %w", err)
	}

	allFailed := true
	for i := range post.Platforms {
		pp := &post.Platforms[i]
		account, err := s.social.FindByID(pp.SocialAccountID)
		if err != nil {
			errMsg := "social account not found"
			_ = s.posts.UpdatePlatformStatus(pp.ID, models.PostStatusFailed, nil, &errMsg)
			continue
		}

		provider, err := s.registry.Get(providers.Platform(account.Platform))
		if err != nil {
			errMsg := fmt.Sprintf("provider not found: %s", account.Platform)
			_ = s.posts.UpdatePlatformStatus(pp.ID, models.PostStatusFailed, nil, &errMsg)
			continue
		}

		oauthCreds, err := s.creds.Decrypt(account.CredentialsEnc)
		if err != nil {
			errMsg := "failed to decrypt credentials"
			_ = s.posts.UpdatePlatformStatus(pp.ID, models.PostStatusFailed, nil, &errMsg)
			continue
		}

		caption := post.Caption
		if pp.CaptionOverride != nil {
			caption = *pp.CaptionOverride
		}

		result, err := provider.Publish(ctx, oauthCreds, &providers.PublishContent{
			Caption:   caption,
			MediaURLs: []string(post.MediaURLs),
		})
		if err != nil {
			errMsg := err.Error()
			_ = s.posts.UpdatePlatformStatus(pp.ID, models.PostStatusFailed, nil, &errMsg)
			continue
		}

		allFailed = false
		_ = s.posts.UpdatePlatformStatus(pp.ID, models.PostStatusPublished, &result.PlatformPostID, nil)
	}

	finalStatus := models.PostStatusPublished
	if allFailed {
		finalStatus = models.PostStatusFailed
	}
	return s.posts.UpdateStatus(postID, finalStatus, nil)
}

func (s *PostService) enqueuePublish(postID uuid.UUID, at time.Time) error {
	task, err := jobs.NewPublishPostTask(postID, asynq.ProcessAt(at), asynq.MaxRetry(3))
	if err != nil {
		return err
	}
	_, err = s.asynq.Enqueue(task)
	return err
}
