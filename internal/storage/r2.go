package storage

// r2.go — Cloudflare R2 client (S3-compatible).
// R2 has no egress fees, making it ideal for media-heavy SaaS.

import (
	"context"
	"fmt"
	"mime"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/allposty/allposty-backend/internal/config"
	"github.com/google/uuid"
)

type R2Client struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

func NewR2Client(cfg *config.Config) (*R2Client, error) {
	r2Endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.R2.AccountID)

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.R2.AccessKeyID,
				cfg.R2.SecretAccessKey,
				"",
			),
		),
		awsconfig.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("r2: load config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(r2Endpoint)
	})

	return &R2Client{
		client:    client,
		bucket:    cfg.R2.Bucket,
		publicURL: strings.TrimRight(cfg.R2.PublicURL, "/"),
	}, nil
}

type UploadInput struct {
	// Key is the storage path, e.g. "workspaces/{id}/media/{uuid}.jpg"
	Key         string
	Body        interface{ Read([]byte) (int, error) }
	ContentType string
	SizeBytes   int64
}

type UploadResult struct {
	Key       string
	PublicURL string
}

func (r *R2Client) Upload(ctx context.Context, input UploadInput) (*UploadResult, error) {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(r.bucket),
		Key:           aws.String(input.Key),
		Body:          input.Body.(interface {
			Read([]byte) (int, error)
			Close() error
		}),
		ContentType:   aws.String(input.ContentType),
		ContentLength: aws.Int64(input.SizeBytes),
	})
	if err != nil {
		return nil, fmt.Errorf("r2: upload %s: %w", input.Key, err)
	}

	return &UploadResult{
		Key:       input.Key,
		PublicURL: fmt.Sprintf("%s/%s", r.publicURL, input.Key),
	}, nil
}

func (r *R2Client) Delete(ctx context.Context, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	return err
}

// MediaKey generates the R2 storage key for a workspace media file.
func MediaKey(workspaceID uuid.UUID, originalFilename string) string {
	ext := filepath.Ext(originalFilename)
	return fmt.Sprintf("workspaces/%s/media/%s%s", workspaceID, uuid.New().String(), ext)
}

// DetectContentType returns the MIME type from a filename extension.
func DetectContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ct := mime.TypeByExtension(ext); ct != "" {
		return ct
	}
	return "application/octet-stream"
}
