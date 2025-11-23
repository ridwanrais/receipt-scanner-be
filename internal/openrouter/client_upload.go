package openrouter

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// UploadImageToSupabase uploads an image to Supabase S3-compatible storage and returns the public URL
func (c *Client) UploadImageToSupabase(imageData []byte, filename string) (string, error) {
	// Check if S3 client is configured
	if c.s3Client == nil {
		return "", &OpenRouterError{
			Op:  "check_s3_config",
			Err: fmt.Errorf("S3 client is not configured. Please check SUPABASE_S3_ENDPOINT, SUPABASE_ACCESS_KEY_ID, and SUPABASE_ACCESS_KEY_SECRET"),
		}
	}

	// Check if bucket is configured
	if c.supabaseBucket == "" {
		return "", &OpenRouterError{
			Op:  "check_bucket_config",
			Err: fmt.Errorf("Supabase bucket is not configured"),
		}
	}

	// Upload the file to S3
	_, err := c.s3Client.PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(c.supabaseBucket),
		Key:           aws.String(filename),
		Body:          bytes.NewReader(imageData),
		ContentType:   aws.String("image/png"),
		ContentLength: aws.Int64(int64(len(imageData))),
	})
	if err != nil {
		return "", &OpenRouterError{
			Op:  "upload_to_s3",
			Err: fmt.Errorf("failed to upload to S3: %w", err),
		}
	}

	// Construct the public URL
	// Format: https://{project-ref}.storage.supabase.co/storage/v1/object/public/{bucket}/{filename}
	baseURL := strings.Replace(c.s3Endpoint, "/storage/v1/s3", "", 1)
	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", baseURL, c.supabaseBucket, filename)

	return publicURL, nil
}
