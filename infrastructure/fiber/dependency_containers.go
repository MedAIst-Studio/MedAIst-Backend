package fiber

import (
	"fmt"
	"os"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/file"
	chunkerInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/chunker"
	embeddingInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/embedding"
	"github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/jina"

	inMemoryInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/in-memory"
	jwe "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/jwe"
	mailersendInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/mailersend"
	minioInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/minio"
	openaiInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/openai"
	postgresInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/postgres"
	qdrantInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/qdrant"
	slogInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/slog"
	validatorInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/validator"
)

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
		Endpoint:        os.Getenv("MINIO_ENDPOINT"),
		AccessKeyID:     os.Getenv("MINIO_ACCESS_KEY"),
		SecretAccessKey: os.Getenv("MINIO_SECRET_KEY"),
		UseSSL:          os.Getenv("MINIO_USE_SSL") == "true",
		BucketName:      os.Getenv("MINIO_BUCKET_NAME"),
		Location:        "us-east-1",
	})
	fmt.Println("Connected to MinIO at", os.Getenv("MINIO_ENDPOINT"))

	chunker := chunkerInfra.NewChunkingService(&file.ChunkOptions{
		MaxChunkSize:    1024,
		OverlapSize:     128,
		PreserveContext: true,
	})

	embeddingService := embeddingInfra.NewEmbeddingService(os.Getenv("OPENAI_API_KEY"))

	vectorDB := qdrantInfra.NewQdrantVectorDB(qdrantInfra.Config{
		Addr:           os.Getenv("QDRANT_URL"),
		CollectionName: os.Getenv("QDRANT_COLLECTION_NAME"),
		ApiKey:         os.Getenv("QDRANT_API_KEY"),
		MinScore:       0.3,
		VectorSize:     1536,
		ChunkLimit:     10,
	})

	openai := openaiInfra.NewOpenAIService(openaiInfra.OpenAIConfig{
		ApiKey:                 os.Getenv("OPENAI_API_KEY"),
		ChatModel:              "gpt-4o",
		GenerateQuestionModel:  "gpt-4o-mini",
		ExpandQuestionModel:    "gpt-4o-mini",
		CheckUserQueryModel:    "gpt-4o-mini",
		GenerateChatTitleModel: "gpt-4o-mini",
		OcrModel:               getEnv("OPENAI_OCR_MODEL", "gpt-4o-mini"),
		ReasoningModel:         getEnv("OPENAI_REASONING_MODEL", "o4-mini"),
	})

	jina := jina.NewService(os.Getenv("JINA_API_KEY"))

	aiService := &MultiModelService{
		openai,
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
	*openaiInfra.OpenAIService
	*jina.Service
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
