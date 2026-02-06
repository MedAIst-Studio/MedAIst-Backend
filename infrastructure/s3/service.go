package s3

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"mime"
	"path/filepath"
	"strings"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Service struct {
	client       *s3.Client
	BucketName   string
	PublicDomain string
}

type S3Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	BucketName      string
	Location        string
	PublicDomain    string
}

func NewS3Service(config *S3Config) *S3Service {
	// Simple validation
	if config.Endpoint == "" {
		config.Endpoint = "s3.amazonaws.com"
	}
	endpoint := config.Endpoint
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		if config.UseSSL {
			endpoint = "https://" + endpoint
		} else {
			endpoint = "http://" + endpoint
		}
	}

	cfg := aws.Config{
		Region:      config.Location,
		Credentials: credentials.NewStaticCredentialsProvider(config.AccessKeyID, config.SecretAccessKey, ""),
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	return &S3Service{
		client:       client,
		BucketName:   config.BucketName,
		PublicDomain: config.PublicDomain,
	}
}

func (s *S3Service) UploadFile(ctx context.Context, file *domain.File, data []byte) (string, error) {
	// Determine object path
	var objectName string
	if file.Path != "" && hasExtension(file.Path) {
		objectName = file.Path
	} else {
		objectName = fmt.Sprintf("uploads/%s.pdf", file.Id.String())
	}

	// Detect Content-Type
	contentType := mime.TypeByExtension(filepath.Ext(objectName))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.BucketName),
		Key:         aws.String(objectName),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})

	if err != nil {
		log.Printf("[ERROR] UploadFile - PutObject Failed! Bucket: '%s', Object: '%s'. Error: %v", s.BucketName, objectName, err)
		return "", fmt.Errorf("upload to s3 failed: %w", err)
	}

	var fileUrl string
	if s.PublicDomain != "" {
		baseUrl := strings.TrimRight(s.PublicDomain, "/")
		if !strings.Contains(baseUrl, "storage/v1/object/public") {
			fileUrl = fmt.Sprintf("%s/storage/v1/object/public/%s/%s", baseUrl, s.BucketName, objectName)
		} else {
			fileUrl = fmt.Sprintf("%s/%s/%s", baseUrl, s.BucketName, objectName)
		}
	} else {
		fileUrl = fmt.Sprintf("%s/%s", s.BucketName, objectName)
	}

	return fileUrl, nil
}

func hasExtension(path string) bool {
	return filepath.Ext(path) != ""
}

func (s *S3Service) DeleteFile(ctx context.Context, filePath string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(filePath),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from s3: %w", err)
	}
	return nil
}

func (s *S3Service) GetFile(ctx context.Context, filePath string) ([]byte, error) {
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(filePath),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file from s3: %w", err)
	}
	defer output.Body.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(output.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	return buf.Bytes(), nil
}

func (s *S3Service) DeleteFiles(ctx context.Context, filePaths []string) error {
	if len(filePaths) == 0 {
		return nil
	}

	// Concurrent deletion could be faster, loop is safe default
	var errorMsgs []string
	for _, filePath := range filePaths {
		_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(s.BucketName),
			Key:    aws.String(filePath),
		})
		if err != nil {
			errorMsgs = append(errorMsgs, fmt.Sprintf("%s: %v", filePath, err))
		}
	}

	if len(errorMsgs) > 0 {
		return fmt.Errorf("failed to delete some files: %s", strings.Join(errorMsgs, "; "))
	}

	return nil
}
