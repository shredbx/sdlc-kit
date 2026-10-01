package image

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// =============================================================================
// R2 OBJECT STORE — binary storage on Cloudflare R2 (SI layer)
// =============================================================================

// R2ObjectStore stores binary image data in Cloudflare R2 using S3-compatible API.
// This is the SI-layer adapter linking the Image foundation type to R2 storage,
// mirroring how Document links to PostgreSQL on the MD layer.
type R2ObjectStore struct {
	client       *s3.Client
	bucket       string
	customDomain string // e.g., "media.bestays.com" for permanent public URLs
}

// NewR2ObjectStore creates an R2-backed object store for binary image data.
// customDomain is used for URL generation (e.g., "https://{customDomain}/{key}").
// If customDomain is empty, URLs are constructed from the R2 endpoint.
func NewR2ObjectStore(bucket, accountID, accessKeyID, secretAccessKey, customDomain string) (*R2ObjectStore, error) {
	if bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}
	if accountID == "" {
		return nil, fmt.Errorf("account_id is required")
	}
	if accessKeyID == "" || secretAccessKey == "" {
		return nil, fmt.Errorf("access_key_id and secret_access_key are required")
	}

	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	client := s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: aws.String(endpoint),
		Credentials: credentials.NewStaticCredentialsProvider(
			accessKeyID,
			secretAccessKey,
			"",
		),
	})

	return &R2ObjectStore{
		client:       client,
		bucket:       bucket,
		customDomain: customDomain,
	}, nil
}

// immutableCacheControl is the Cache-Control written on every stored object.
// Image keys are content-addressed (a fresh UUID/hash per upload — a key's bytes
// never change), so the bytes are safe to cache forever. Without this header the
// CDN falls back to a short default (~4h), which trips the Lighthouse/SEO
// "serve static assets with an efficient cache policy" audit. One year + immutable
// is the standard long-lived-asset policy.
const immutableCacheControl = "public, max-age=31536000, immutable"

// Upload writes binary data to R2 with a content type and a long-lived immutable
// cache policy (see immutableCacheControl) so the CDN can cache the object
// aggressively — required for the SEO cache-policy audit to pass.
func (s *R2ObjectStore) Upload(ctx context.Context, key string, data io.Reader, contentType string) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Buffer the reader to get content length (required for PutObject)
	buf, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("%w: read input: %v", ErrUploadFailed, err)
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(buf),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(int64(len(buf))),
		CacheControl:  aws.String(immutableCacheControl),
	})
	if err != nil {
		return fmt.Errorf("%w: R2 put: %v", ErrUploadFailed, err)
	}

	return nil
}

// Download retrieves binary data from R2.
func (s *R2ObjectStore) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	// The returned body is a STREAM bound to ctx — its bytes are read by the caller
	// AFTER this function returns. So the timeout context must NOT be cancelled here
	// (a deferred cancel would abort the read with "context canceled"); cancel is tied
	// to the body's Close instead, so the 30s budget bounds the whole download.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)

	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("%w: R2 get: %v", ErrBackendError, err)
	}

	return &cancelOnClose{ReadCloser: out.Body, cancel: cancel}, nil
}

// cancelOnClose ties a context cancel to the body's Close so a streamed download's
// context outlives the Download call (the caller reads the body afterwards) yet is
// still released when the caller is done.
type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelOnClose) Close() error {
	err := c.ReadCloser.Close()
	c.cancel()
	return err
}

// Delete removes binary data from R2.
func (s *R2ObjectStore) Delete(ctx context.Context, key string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("%w: R2 delete: %v", ErrBackendError, err)
	}

	return nil
}

// URL returns the public CDN URL for an object.
// If customDomain is set, uses https://{customDomain}/{key}.
// Otherwise, falls back to R2 public bucket URL pattern.
func (s *R2ObjectStore) URL(key string) string {
	if s.customDomain != "" {
		return "https://" + s.customDomain + "/" + key
	}
	return "https://" + s.bucket + ".r2.dev/" + key
}
