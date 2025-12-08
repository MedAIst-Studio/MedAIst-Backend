package minio

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"mime"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioService struct {
	minioClient *minio.Client
	BucketName  string
}

type MinioConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	BucketName      string
	Location        string
}

func NewMinioService(config *MinioConfig) *MinioService {
	ctx := context.Background()

	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "localhost:9000"
	}

	log.Printf("MinIO Connecting to: %s (SSL: %v)", endpoint, config.UseSSL)

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		Secure: config.UseSSL,
	})
	if err != nil {
		log.Println("MinIO Client creation failed:", err)
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
		minioClient: minioClient,
		BucketName:  bucketName,
	}
}

func (s *MinioService) UploadFile(ctx context.Context, file *domain.File, data []byte) error {

	contentType := "application/octet-stream"
	if mimeType := mime.TypeByExtension("pdf"); mimeType != "" {
		contentType = mimeType
	}

	objectName := fmt.Sprintf("uploads/%s%s", file.Id.String(), ".pdf")

	_, err := s.minioClient.PutObject(ctx,
		s.BucketName,
		objectName,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType},
	)

	if err != nil {
		return fmt.Errorf("upload to minio failed: %w", err)
	}
	return nil
}

func (s *MinioService) DeleteFile(ctx context.Context, filePath string) error {
	err := s.minioClient.RemoveObject(ctx, s.BucketName, filePath, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file from minio: %w", err)
	}
	return nil
}

func (s *MinioService) GetFile(ctx context.Context, fileId uuid.UUID) ([]byte, error) {
	objectName := fmt.Sprintf("uploads/%s%s", fileId.String(), ".pdf")

	object, err := s.minioClient.GetObject(ctx, s.BucketName, objectName, minio.GetObjectOptions{})
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

func (s *MinioService) DeleteFiles(ctx context.Context, fileIds []string) error {
	if len(fileIds) == 0 {
		return nil
	}

	var errors []error
	for _, fileId := range fileIds {
		objectName := formatFilePath(fileId)
		err := s.minioClient.RemoveObject(ctx, s.BucketName, objectName, minio.RemoveObjectOptions{})
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to delete file %s: %w", objectName, err))
		}
	}

	// Return combined errors if any occurred
	if len(errors) > 0 {
		return fmt.Errorf("failed to delete %d file(s): %v", len(errors), errors)
	}

	return nil
}

func formatFilePath(fileId string) string {
	return "/uploads/" + fileId + ".pdf"
}
