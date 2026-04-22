package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/allposty/allposty-backend/internal/models"
	"github.com/allposty/allposty-backend/internal/repository"
	"github.com/allposty/allposty-backend/internal/storage"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maxUploadBytes = 100 * 1024 * 1024 // 100 MB

var (
	ErrMediaNotFound   = errors.New("media file not found")
	ErrFileTooLarge    = errors.New("file exceeds 100 MB limit")
)

type MediaService struct {
	media *repository.MediaRepository
	orgs  *OrgService
	r2    *storage.R2Client
}

func NewMediaService(media *repository.MediaRepository, orgs *OrgService, r2 *storage.R2Client) *MediaService {
	return &MediaService{media: media, orgs: orgs, r2: r2}
}

func (s *MediaService) Upload(ctx context.Context, userID, workspaceID uuid.UUID, fh *multipart.FileHeader, folder *string) (*models.MediaFile, error) {
	if err := s.orgs.RequireWorkspaceAccess(workspaceID, userID); err != nil {
		return nil, ErrForbidden
	}
	if fh.Size > maxUploadBytes {
		return nil, ErrFileTooLarge
	}

	f, err := fh.Open()
	if err != nil {
		return nil, fmt.Errorf("open upload: %w", err)
	}
	defer f.Close()

	key := storage.MediaKey(workspaceID, fh.Filename)
	ct := storage.DetectContentType(fh.Filename)

	result, err := s.r2.Upload(ctx, storage.UploadInput{
		Key:         key,
		Body:        f.(io.ReadCloser),
		ContentType: ct,
		SizeBytes:   fh.Size,
	})
	if err != nil {
		return nil, fmt.Errorf("r2 upload: %w", err)
	}

	record := &models.MediaFile{
		WorkspaceID: workspaceID,
		UploadedBy:  userID,
		Filename:    fh.Filename,
		MimeType:    ct,
		SizeBytes:   fh.Size,
		R2Key:       result.Key,
		PublicURL:   result.PublicURL,
		Folder:      folder,
	}
	if err := s.media.Create(record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *MediaService) List(userID, workspaceID uuid.UUID, folder *string) ([]models.MediaFile, error) {
	if err := s.orgs.RequireWorkspaceAccess(workspaceID, userID); err != nil {
		return nil, ErrForbidden
	}
	return s.media.FindByWorkspace(workspaceID, folder)
}

func (s *MediaService) Delete(ctx context.Context, userID, fileID uuid.UUID) error {
	file, err := s.media.FindByID(fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMediaNotFound
		}
		return err
	}
	if err := s.orgs.RequireWorkspaceAccess(file.WorkspaceID, userID); err != nil {
		return ErrForbidden
	}

	// Delete from R2 first, then DB
	if err := s.r2.Delete(ctx, file.R2Key); err != nil {
		return fmt.Errorf("r2 delete: %w", err)
	}
	return s.media.Delete(fileID)
}
