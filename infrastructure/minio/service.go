package minio

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"mime"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
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

	accessKeyID := config.AccessKeyID
	secretAccessKey := config.SecretAccessKey

	useSSL := config.UseSSL

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	bucketName := config.BucketName
	location := config.Location

	err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location})
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			log.Printf("We already own %s\n", bucketName)
		} else {
			log.Fatalln(err)
		}
	} else {
		log.Printf("Successfully created %s\n", bucketName)
	}

	return &MinioService{
		minioClient: minioClient,
		BucketName:  bucketName}
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
