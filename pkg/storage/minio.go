package storage

import (
	"bytes"
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/Zeropeepo/neknow-bot/pkg/config"
)

type MinIOStorage struct {
	client *minio.Client
	bucket string
}

func NewMinIOStorage(cfg *config.Config) (*MinIOStorage, error) {
	client, err := minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKey, cfg.MinIO.SecretKey, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	// Buat bucket kalau belum ada
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.MinIO.Bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.MinIO.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}

	return &MinIOStorage{client: client, bucket: cfg.MinIO.Bucket}, nil
}

func (s *MinIOStorage) Upload(ctx context.Context, objectKey string, content []byte, mimeType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, objectKey, bytes.NewReader(content),
		int64(len(content)), minio.PutObjectOptions{ContentType: mimeType})
	return err
}

func (s *MinIOStorage) Delete(ctx context.Context, objectKey string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{})
}

func (s *MinIOStorage) GetURL(objectKey string) string {
	return fmt.Sprintf("%s/%s/%s", s.bucket, s.bucket, objectKey)
}
