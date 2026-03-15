package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "worksphere-api/internal/config"
)

// R2Storage implements the StorageProvider interface using Cloudflare R2.
// R2 is S3-compatible, so we use the AWS SDK for Go v2.
type R2Storage struct {
	client        *s3.Client
	bucketName    string
	publicBaseURL string
}

// Ensure R2Storage implements StorageProvider.
var _ StorageProvider = (*R2Storage)(nil)

// NewR2Storage initializes a new Cloudflare R2 storage client.
func NewR2Storage(cfg appconfig.R2Config) (*R2Storage, error) {
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" || cfg.BucketName == "" || cfg.Endpoint == "" {
		return nil, fmt.Errorf("invalid R2 configuration: missing required credentials, bucket name, or endpoint")
	}

	// Load AWS config with static credentials and "auto" region for R2.
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS SDK config for R2: %w", err)
	}

	// Create S3 client with the specific R2 endpoint.
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
	})

	return &R2Storage{
		client:        client,
		bucketName:    cfg.BucketName,
		publicBaseURL: cfg.PublicBaseURL,
	}, nil
}

// UploadFile uploads a file to the R2 bucket.
func (r *R2Storage) UploadFile(ctx context.Context, key string, body io.Reader, contentType string) (string, error) {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucketName),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to R2: %w", err)
	}

	return r.GetFileURL(key), nil
}

// DeleteFile removes a file from the R2 bucket.
func (r *R2Storage) DeleteFile(ctx context.Context, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from R2: %w", err)
	}

	return nil
}

// GetFileURL builds the public URL for a file if a public base URL is configured.
func (r *R2Storage) GetFileURL(key string) string {
	if r.publicBaseURL != "" {
		// Clean slashes for consistent URL building
		baseURL := r.publicBaseURL
		if baseURL[len(baseURL)-1] == '/' {
			baseURL = baseURL[:len(baseURL)-1]
		}
		return fmt.Sprintf("%s/%s", baseURL, key)
	}
	return key
}
