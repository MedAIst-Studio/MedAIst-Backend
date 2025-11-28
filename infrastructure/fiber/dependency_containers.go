package fiber

import (
	"fmt"
	"os"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/file"
	"github.com/MedAIst-Studio/MedAIist-Backend/app/message"
	chunkerInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/chunker"
	embeddingInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/embedding"
	"github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/jina"

	grokInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/grok"
	inMemoryInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/in-memory"
	jwe "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/jwe"
	mailersendInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/mailersend"
	minioInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/minio"
	postgresInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/postgres"
	qdrantInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/qdrant"
	slogInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/slog"
	validatorInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/validator"
)

type AIService interface {
	message.AIService
}

type DependencyContainer struct {
	Repo            *postgresInfra.Repository
	Chunker         *chunkerInfra.PdfChunkingService
	Embedding       *embeddingInfra.EmbeddingService
	VectorDB        *qdrantInfra.QdrantVectorDB
	FileStorage     *minioInfra.MinioService
	jweTokenService *jwe.Service
	inMemoryClient  *inMemoryInfra.InMemoryClient
	slogLogger      *slogInfra.SlogLogger
	Validator       *validatorInfra.Validator
	mailerSender    *mailersendInfra.MailerSendService
	AIService       AIService
}

func NewAuthDependencyContainer() *DependencyContainer {
	postgresRepo := postgresInfra.NewRepository(os.Getenv("DATABASE_URL"))
	fmt.Println("Connected to database")

	jweTokenService := jwe.NewJWETokenService(&jwe.Config{
		AccessTokenEncryptionKey:  "12345678901234567890123456789012",
		RefreshTokenEncryptionKey: "12345678901234567890123456789012",
		SecureEmailEncryptionKey:  "12345678901234567890123456789012",
		AuthAccessTokenDuration:   time.Minute * 15,
		AuthRefreshTokenDuration:  time.Hour * 24 * 30,
		SecureEmailTokenDuration:  time.Minute * 11,
	})

	mailersendService := mailersendInfra.NewMailerSendService(os.Getenv("MAILERSEND_API_KEY"), os.Getenv("MAILERSEND_SENDER_EMAIL"), os.Getenv("MAILERSEND_SENDER_NAME"))

	minioInfraService := minioInfra.NewMinioService(&minioInfra.MinioConfig{
		Endpoint:        "minio:9000",
		AccessKeyID:     os.Getenv("MINIO_ACCESS_KEY"),
		SecretAccessKey: os.Getenv("MINIO_SECRET_KEY"),
		UseSSL:          false,
		BucketName:      "files",
		Location:        "us-east-1",
	})
	fmt.Println("Connected to MinIO at", "localhost:9000")

	chunker := chunkerInfra.NewChunkingService(&file.ChunkOptions{
		MaxChunkSize:    1024,
		OverlapSize:     128,
		PreserveContext: true,
	})

	embeddingService := embeddingInfra.NewEmbeddingService(os.Getenv("EMBEDDING_API_KEY"), os.Getenv("EMBEDDING_API_DOMAIN"))

	vectorDB := qdrantInfra.NewQdrantVectorDB(qdrantInfra.Config{
		Addr:           os.Getenv("QDRANT_URL"),
		CollectionName: os.Getenv("QDRANT_COLLECTION_NAME"),
		MinScore:       0.3,
		VectorSize:     1024,
		ChunkLimit:     10,
	})

	grok := grokInfra.NewGrokService(os.Getenv("GROK_API_KEY"), "grok-3")

	jina := jina.NewService(os.Getenv("JINA_API_KEY"))

	aiService := &MultiModelService{
		grok,
		jina,
	}

	slogLogger := slogInfra.NewLogger()
	validator := validatorInfra.NewValidator(slogLogger)

	inMemoryClient := inMemoryInfra.NewInMemoryClient(inMemoryInfra.InMemoryConfig{
		OAuthStateTTL:   time.Minute * 1,
		OAuthSessionTTL: time.Minute * 15,
	})

	return &DependencyContainer{
		Repo:            postgresRepo,
		Chunker:         chunker,
		Embedding:       embeddingService,
		VectorDB:        vectorDB,
		FileStorage:     minioInfraService,
		jweTokenService: jweTokenService,
		inMemoryClient:  inMemoryClient,
		Validator:       validator,
		slogLogger:      slogLogger,
		mailerSender:    mailersendService,
		AIService:       aiService,
	}
}

type MultiModelService struct {
	*grokInfra.GrokService
	*jina.Service
}
