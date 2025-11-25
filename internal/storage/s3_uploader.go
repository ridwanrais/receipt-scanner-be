package storage

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// S3Uploader handles uploading images to S3-compatible storage
type S3Uploader struct {
	s3Client *s3.S3
	bucket   string
	endpoint string
}

// Config holds configuration for S3 uploader
type Config struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	Bucket          string
	Region          string
}

// NewS3Uploader creates a new S3 uploader
func NewS3Uploader(config *Config) (*S3Uploader, error) {
	if config.Endpoint == "" || config.AccessKeyID == "" || config.AccessKeySecret == "" {
		return nil, fmt.Errorf("S3 configuration is incomplete")
	}

	if config.Bucket == "" {
		return nil, fmt.Errorf("S3 bucket is not configured")
	}

	sess := session.Must(session.NewSession(&aws.Config{
		Region:           aws.String(config.Region),
		Endpoint:         aws.String(config.Endpoint + "/storage/v1/s3"),
		Credentials:      credentials.NewStaticCredentials(config.AccessKeyID, config.AccessKeySecret, ""),
		S3ForcePathStyle: aws.Bool(true),
		DisableSSL:       aws.Bool(false),
	}))

	return &S3Uploader{
		s3Client: s3.New(sess),
		bucket:   config.Bucket,
		endpoint: config.Endpoint,
	}, nil
}

// UploadImage uploads an image to S3 and returns the public URL
func (u *S3Uploader) UploadImage(imageData []byte, filename string) (string, error) {
	// Upload the file to S3
	_, err := u.s3Client.PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(u.bucket),
		Key:           aws.String(filename),
		Body:          bytes.NewReader(imageData),
		ContentType:   aws.String("image/png"),
		ContentLength: aws.Int64(int64(len(imageData))),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Construct the public URL
	// Format: https://{project-ref}.storage.supabase.co/storage/v1/object/public/{bucket}/{filename}
	baseURL := strings.Replace(u.endpoint, "/storage/v1/s3", "", 1)
	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", baseURL, u.bucket, filename)

	return publicURL, nil
}
