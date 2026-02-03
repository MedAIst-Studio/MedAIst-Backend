package fiber

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/file"

	chunkerInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/chunker"
	"github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/jina"
	"github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/openrouter"

	inMemoryInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/in-memory"
	jwe "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/jwe"
	mailersendInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/mailersend"
	minioInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/minio"

	postgresInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/postgres"
	qdrantInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/qdrant"
	slogInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/slog"
	validatorInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/validator"
)

type DependencyContainer struct {
	Repo            *postgresInfra.Repository
	Chunker         *chunkerInfra.PdfChunkingService
	Embedding       *openrouter.OpenRouterService
	VectorDB        *qdrantInfra.QdrantVectorDB
	FileStorage     *minioInfra.MinioService
	jweTokenService *jwe.Service
	inMemoryClient  *inMemoryInfra.InMemoryClient
	slogLogger      *slogInfra.SlogLogger
	Validator       *validatorInfra.Validator
	mailerSender    *mailersendInfra.MailerSendService
	AIService       *MultiModelService
}

func NewAuthDependencyContainer() *DependencyContainer {
	postgresRepo := postgresInfra.NewRepository(os.Getenv("DATABASE_URL"))
	fmt.Println("Connected to database")

	jweTokenService := jwe.NewJWETokenService(&jwe.Config{
		AccessTokenEncryptionKey:  os.Getenv("ACCESS_TOKEN_ENCRYPTION_KEY"),
		RefreshTokenEncryptionKey: os.Getenv("REFRESH_TOKEN_ENCRYPTION_KEY"),
		SecureEmailEncryptionKey:  os.Getenv("SECURE_EMAIL_ENCRYPTION_KEY"),
		AuthAccessTokenDuration:   time.Minute * 15,
		AuthRefreshTokenDuration:  time.Hour * 24 * 30,
		SecureEmailTokenDuration:  time.Minute * 11,
	})

	mailersendService := mailersendInfra.NewMailerSendService(os.Getenv("MAILERSEND_API_KEY"), os.Getenv("MAILERSEND_SENDER_EMAIL"), os.Getenv("MAILERSEND_SENDER_NAME"))

	minioInfraService := minioInfra.NewMinioService(&minioInfra.MinioConfig{
		Endpoint:        os.Getenv("S3_ENDPOINT"),
		AccessKeyID:     os.Getenv("S3_ACCESS_KEY"),
		SecretAccessKey: os.Getenv("S3_SECRET_KEY"),
		UseSSL:          os.Getenv("S3_USE_SSL") == "true",
		BucketName:      os.Getenv("S3_BUCKET_NAME"),
		Location:        os.Getenv("S3_REGION"),
		PublicDomain:    os.Getenv("S3_PUBLIC_URL"),
	})
	fmt.Println("Connected to S3 Storage at", os.Getenv("S3_ENDPOINT"))

	chunker := chunkerInfra.NewChunkingService(&file.ChunkOptions{
		MaxChunkSize:    1024,
		OverlapSize:     128,
		PreserveContext: true,
	})

	vectorDB := qdrantInfra.NewQdrantVectorDB(qdrantInfra.Config{
		Host:           os.Getenv("QDRANT_HOST"),
		Port:           6334,
		CollectionName: os.Getenv("QDRANT_COLLECTION_NAME"),
		ApiKey:         os.Getenv("QDRANT_API_KEY"),
		MinScore:       0.3,
		VectorSize:     1536,
		ChunkLimit:     10,
	})

	openRouter := openrouter.NewOpenRouterService(openrouter.Config{
		ApiKey:                 getEnv("OPENROUTER_API_KEY", ""),
		ChatModel:              getEnv("OPENROUTER_CHAT_MODEL", "openai/gpt-4o-mini"),
		GenerateQuestionModel:  getEnv("OPENROUTER_QUESTION_MODEL", "openai/gpt-4o-mini"),
		ExpandQuestionModel:    getEnv("OPENROUTER_EXPAND_MODEL", "openai/gpt-4o-mini"),
		CheckUserQueryModel:    getEnv("OPENROUTER_TITLE_MODEL", "openai/gpt-4o-mini"),
		GenerateChatTitleModel: getEnv("OPENROUTER_TITLE_MODEL", "openai/gpt-4o-mini"),
		ReasoningModel:         getEnv("OPENROUTER_REASONING_MODEL", "google/gemini-3-flash-preview"),
		VisionModel:            getEnv("OPENROUTER_VISION_MODEL", "google/gemini-3-flash-preview"),
		EmbeddingModel:         getEnv("OPENROUTER_EMBEDDING_MODEL", "openai/text-embedding-3-small"),
	})

	jina := jina.NewService(os.Getenv("JINA_API_KEY"))

	aiService := &MultiModelService{
		OpenRouterService: openRouter,
		Service:           jina,
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
		Embedding:       openRouter, // Using OpenRouter for embeddings
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
	*openrouter.OpenRouterService
	*jina.Service
}

func (s *MultiModelService) ExtractTextFromImage(ctx context.Context, imageUrl string) (string, error) {
	return s.OpenRouterService.ExtractTextFromImage(ctx, imageUrl)
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
