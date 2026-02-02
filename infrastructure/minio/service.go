package minio

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"mime"
	"path/filepath"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioService struct {
	minioClient  *minio.Client
	BucketName   string
	PublicDomain string
}

type MinioConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	BucketName      string
	Location        string
	PublicDomain    string
}

func NewMinioService(config *MinioConfig) *MinioService {
	ctx := context.Background()

	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "localhost:9000"
	}

	log.Printf("S3 Storage Connecting to: %s (SSL: %v)", endpoint, config.UseSSL)

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		Secure: config.UseSSL,
	})
	if err != nil {
		log.Println("S3 Client creation failed:", err)
	}

	bucketName := config.BucketName

	location := config.Location
	if location == "" {
		location = "us-east-1"
	}

	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		log.Println("Bucket control failed:", err)
	}

	if !exists {
		log.Printf("Bucket '%s' not found, creating... (Region: %s)", bucketName, location)
		err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location})
		if err != nil {
			log.Println("Bucket creation failed:", err)
		}
		log.Printf("Bucket '%s' successfully created.", bucketName)
	} else {
		log.Printf("Bucket '%s' already exists, moving on.", bucketName)
	}

	return &MinioService{
		minioClient:  minioClient,
		BucketName:   bucketName,
		PublicDomain: config.PublicDomain,
	}
}

func (s *MinioService) UploadFile(ctx context.Context, file *domain.File, data []byte) (string, error) {

	var objectName string
	// Backward compatibility: domain.NewFile sets Path without extension for PDFs.
	// If Path contains a dot, assume it's a full path with extension.
	if file.Path != "" && hasExtension(file.Path) {
		objectName = file.Path
	} else {
		objectName = fmt.Sprintf("uploads/%s%s", file.Id.String(), ".pdf")
	}

	contentType := mime.TypeByExtension(filepath.Ext(objectName))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err := s.minioClient.PutObject(ctx,
		s.BucketName,
		objectName,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType},
	)

	if err != nil {
		return "", fmt.Errorf("upload to s3 failed: %w", err)
	}

	var fileUrl string
	if s.PublicDomain != "" {
		fileUrl = fmt.Sprintf("%s/%s/%s", s.PublicDomain, s.BucketName, objectName)
	} else {
		endpoint := s.minioClient.EndpointURL().String()
		fileUrl = fmt.Sprintf("%s/%s/%s", endpoint, s.BucketName, objectName)
	}

	return fileUrl, nil
}

func hasExtension(path string) bool {
	return filepath.Ext(path) != ""
}

func (s *MinioService) DeleteFile(ctx context.Context, filePath string) error {
	err := s.minioClient.RemoveObject(ctx, s.BucketName, filePath, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file from minio: %w", err)
	}
	return nil
}

func (s *MinioService) GetFile(ctx context.Context, filePath string) ([]byte, error) {
	// filePath is now expected to be the full object name/path in the bucket
	object, err := s.minioClient.GetObject(ctx, s.BucketName, filePath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get file from minio: %w", err)
	}
	defer object.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(object)
	if err != nil {
		return nil, fmt.Errorf("failed to read file from minio object: %w", err)
	}

	return buf.Bytes(), nil
}

func (s *MinioService) DeleteFiles(ctx context.Context, filePaths []string) error {
	if len(filePaths) == 0 {
		return nil
	}

	var errors []error
	for _, filePath := range filePaths {
		err := s.minioClient.RemoveObject(ctx, s.BucketName, filePath, minio.RemoveObjectOptions{})
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to delete file %s: %w", filePath, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to delete %d file(s): %v", len(errors), errors)
	}

	return nil
}
