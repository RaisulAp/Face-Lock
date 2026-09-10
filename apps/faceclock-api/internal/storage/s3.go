package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config holds connection parameters for S3/MinIO.
type S3Config struct {
	Endpoint             string
	Region               string
	Bucket               string
	AccessKey            string
	SecretKey            string
	ForcePathStyle       bool
	UseSSL               bool
	ServerSideEncryption string // "AES256" or empty
}

// S3Store implements Store backed by MinIO or AWS S3.
type S3Store struct {
	client *minio.Client
	bucket string
	sse    string
}

// NewS3Store initializes an S3Store. It creates the bucket if it does not exist.
func NewS3Store(ctx context.Context, cfg S3Config) (*S3Store, error) {
	endpoint := cfg.Endpoint
	useSSL := cfg.UseSSL
	if strings.HasPrefix(endpoint, "http://") {
		endpoint = strings.TrimPrefix(endpoint, "http://")
		useSSL = false
	} else if strings.HasPrefix(endpoint, "https://") {
		endpoint = strings.TrimPrefix(endpoint, "https://")
		useSSL = true
	}

	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: useSSL,
		Region: cfg.Region,
	}

	client, err := minio.New(endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("storage: minio client init: %w", err)
	}

	store := &S3Store{
		client: client,
		bucket: cfg.Bucket,
		sse:    cfg.ServerSideEncryption,
	}

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err == nil && !exists {
		_ = client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region})
	}

	return store, nil
}

func (s *S3Store) Put(ctx context.Context, key string, r io.Reader, contentType string) (string, error) {
	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}

	_, err := s.client.PutObject(ctx, s.bucket, key, r, -1, opts)
	if err != nil {
		return "", fmt.Errorf("storage: s3 put %q: %w", key, err)
	}
	return key, nil
}

func (s *S3Store) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("storage: s3 get %q: %w", key, err)
	}
	_, err = obj.Stat()
	if err != nil {
		obj.Close()
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" || strings.Contains(err.Error(), "The specified key does not exist") {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("storage: s3 stat %q: %w", key, err)
	}
	return obj, nil
}

func (s *S3Store) Delete(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("storage: s3 delete %q: %w", key, err)
	}
	return nil
}

func (s *S3Store) SignedURL(ctx context.Context, key string, ttlSeconds int) (string, error) {
	reqParams := make(url.Values)
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, time.Duration(ttlSeconds)*time.Second, reqParams)
	if err != nil {
		return "", fmt.Errorf("storage: s3 presign: %w", err)
	}
	return u.String(), nil
}

func (s *S3Store) Copy(ctx context.Context, srcKey, dstKey string) error {
	srcOpts := minio.CopySrcOptions{
		Bucket: s.bucket,
		Object: srcKey,
	}
	dstOpts := minio.CopyDestOptions{
		Bucket: s.bucket,
		Object: dstKey,
	}
	_, err := s.client.CopyObject(ctx, dstOpts, srcOpts)
	if err != nil {
		return fmt.Errorf("storage: s3 copy %q to %q: %w", srcKey, dstKey, err)
	}
	return nil
}

func (s *S3Store) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}
	for obj := range s.client.ListObjects(ctx, s.bucket, opts) {
		if obj.Err != nil {
			return nil, fmt.Errorf("storage: s3 list: %w", obj.Err)
		}
		keys = append(keys, obj.Key)
	}
	return keys, nil
}
