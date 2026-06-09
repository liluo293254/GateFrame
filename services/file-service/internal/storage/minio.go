package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/gateframe/file-service/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	mc     *minio.Client
	bucket string
}

func New(cfg config.MinIOConfig) (*Client, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}
	return &Client{mc: mc, bucket: cfg.Bucket}, nil
}

func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.mc.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("bucket exists: %w", err)
	}
	if exists {
		return nil
	}
	if err := c.mc.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("make bucket: %w", err)
	}
	return nil
}

func (c *Client) PutObject(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err := c.mc.PutObject(ctx, c.bucket, objectKey, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}
	return nil
}

func (c *Client) GetObject(ctx context.Context, objectKey string) (io.ReadCloser, string, int64, error) {
	obj, err := c.mc.GetObject(ctx, c.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", 0, fmt.Errorf("get object: %w", err)
	}
	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, "", 0, fmt.Errorf("stat object: %w", err)
	}
	contentType := info.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return obj, contentType, info.Size, nil
}

func (c *Client) RemoveObject(ctx context.Context, objectKey string) error {
	if err := c.mc.RemoveObject(ctx, c.bucket, objectKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("remove object: %w", err)
	}
	return nil
}
