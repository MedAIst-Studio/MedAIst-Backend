//	@title		MedAIist-Backend API
//	@version	1.0
//	@description
//	@description	## How to use the API
//	@description	1- Click which endpoint you want to use.
//	@description	2- Click "Try it out" button.
//	@description	3- Add your request body or your parameters which are showed and required by the endpoint.
//	@description	4- Click "Execute" button.
//	@description	5- You will see the response.
//	@description
//	@description	Some endpoints require authentication. In this case, you need to log in first.
//	@description	I created two types of users for this project: admin and regular user.
//	@description	Just send a POST request as below at [here](http://localhost:3000/swagger/index.html/).
//	@description	After login, you will get a JWE token in cookies.
//	@description	If you're using cookie-based auth, the cookie will be sent automatically.
//	@description	Alternatively, you can use Bearer Token authentication via the "Authorize" button.
//	@description
//	@description	### Login Request For Admin
//	@description	```json
//	@description	{
//	@description	"email": "admin@admin.com",
//	@description	"password": "admin123"
//	@description	}
//	@description	```
//	@description
//	@description	### Login Request For User
//	@description	```json
//	@description	{
//	@description	"email": "user@user.com",
//	@description	"password": "user1234"
//	@description	}
//	@description	```
//	@description
//	@description	## Error Handling
//	@description	All error responses will follow this JSON format:
//	@description
//	@description	```json
//	@description	{
//	@description	"message": string,
//	@description	"code": int
//	@description	}
//	@description	```
//	@description	### Example
//	@description	```json
//	@description	{
//	@description	"message": "invalid request",
//	@description	"code": 400
//	@description	}
//	@description	```
//	@description
//	@description	## Reminder
//	@description	I did not use `/api` prefix for the endpoint routes. Because I love to host my API on "api" subdomain.
//	@description	Status code with `2xx` is a success code.
//	@description	Status code with `4xx` is a client error code.
//	@description	Status code with `5xx` is a server error code.
//

// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Enter your Bearer token in the format **Bearer &lt;token&gt;**
//
// @host						localhost:3000
package fiber

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/auth"
	"github.com/MedAIst-Studio/MedAIist-Backend/app/chat"
	"github.com/MedAIst-Studio/MedAIist-Backend/app/file"
	"github.com/MedAIst-Studio/MedAIist-Backend/app/healthcheck"
	"github.com/MedAIst-Studio/MedAIist-Backend/app/message"
	"github.com/MedAIst-Studio/MedAIist-Backend/app/user"
	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	chunkerInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/chunker"
	embeddingInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/embedding"
	googleInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/google"
	grokInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/grok"
	inMemoryInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/in-memory"
	jwe "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/jwe"
	mailersendInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/mailersend"
	minioInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/minio"
	postgresInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/postgres"
	qdrantInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/qdrant"
	slogInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/slog"
	validatorInfra "github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/validator"

	fiberSwagger "github.com/swaggo/fiber-swagger"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
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
		MinScore:       0.7,
		VectorSize:     1024,
		ChunkLimit:     5,
	})

	grok := grokInfra.NewGrokService(os.Getenv("GROK_API_KEY"), "grok-3")

	slogLogger := slogInfra.NewLogger()
	sl := slogLogger
	validator := validatorInfra.NewValidator(slogLogger)

	inMemoryClient := inMemoryInfra.NewInMemoryClient(inMemoryInfra.InMemoryConfig{
		OAuthStateTTL:   time.Minute * 1,
		OAuthSessionTTL: time.Minute * 15,
	})

	middlewareManager := NewMiddlewareManager(jweTokenService, slogLogger)

	healthcheckHandler := healthcheck.NewHealthcheckHandler()

	if !domain.IsProdEnv() {
		domain.CookieSecure = false
	}

	signupHandler := auth.NewSignupHandler(&auth.SignupConfig{
		Repo:         postgresRepo,
		TokenService: jweTokenService,
		EmailService: mailersendService,
		Validator:    validator,
		Logger:       slogLogger,
	})

	loginHandler := auth.NewLoginHandler(&auth.LoginConfig{
		Repo:         postgresRepo,
		TokenService: jweTokenService,
		Validator:    validator,
		Logger:       slogLogger,
	})

	exchangeHandler := auth.NewExchangeHandler(inMemoryClient)

	googleOAuthService := googleInfra.NewGoogleOAuthService(&googleInfra.GoogleOAuthDependenciesConfig{
		Repo:         postgresRepo,
		TokenService: jweTokenService,
		Cache:        inMemoryClient,
		FrontendURL:  os.Getenv("FRONTEND_URL"),
		ClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_OAUTH_REDIRECT_URL"),
		Logger:       slogLogger,
	})

	logoutHandler := auth.NewLogoutHandler(postgresRepo)
	refreshTokenHandler := auth.NewRefreshTokenHandler(postgresRepo, jweTokenService)
	deleteAccountHandler := user.NewDeleteAccountHandler(postgresRepo, sl)
	updateUsernameHandler := user.NewUpdateUsernameHandler(postgresRepo, validator)
	getCurrentUserHandler := user.NewGetCurrentUserHandler(postgresRepo)
	updatePasswordHandler := user.NewChangePasswordHandler(postgresRepo, validator)
	forgotPasswordHandler := user.NewForgotPasswordHandler(postgresRepo, mailersendService, jweTokenService, sl, validator)
	resetPasswordHandler := user.NewResetPasswordHandler(postgresRepo, jweTokenService, sl, validator)

	getChatsHandler := chat.NewGetChatsHandler(postgresRepo)
	getChatByIDHandler := chat.NewGetChatByIDHandler(postgresRepo)
	deleteChatHandler := chat.NewDeleteChatHandler(postgresRepo)

	uploadFileHandler := file.NewUploadFileHandler(postgresRepo, minioInfraService, vectorDB, embeddingService, chunker, 48, true)
	getFilesHandler := file.NewGetFilesHandler(postgresRepo)
	deleteFileByIdHandler := file.NewDeleteFileHandler(postgresRepo, minioInfraService, vectorDB)
	getFileHandler := file.NewGetFileHandler(postgresRepo, minioInfraService)

	sendMessageHandler := message.NewSendMessageHandler(message.Config{
		PastMessagesLimit: 5,
		Stream:            false,
		Repo:              postgresRepo,
		Ems:               embeddingService,
		Vb:                vectorDB,
		ChatSvc:           grok,
	})

	app.Get("/healthcheck", Handle(healthcheckHandler, sl))
	app.Use(contextMiddleware)

	authApp := app.Group("/auth")
	authApp.Post("/signup", Handle(signupHandler, sl))
	authApp.Post("/login", Handle(loginHandler, sl))
	authApp.Post("/logout", Handle(logoutHandler, sl))
	authApp.Post("/refresh", Handle(refreshTokenHandler, sl))
	authApp.Post("/exchange", Handle(exchangeHandler, sl))
	authApp.Get("/google", googleOAuthService.RedirectHandler)

	app.Get("/api/auth/google/callback", googleOAuthService.Callback)

	usersPublicApp := app.Group("/users")
	usersPublicApp.Post("/forgot-password", Handle(forgotPasswordHandler, sl))
	usersPublicApp.Post("/reset-password", Handle(resetPasswordHandler, sl))

	usersApp := app.Group("/users", middlewareManager.AuthMiddleware)
	usersApp.Get("/profile", Handle(getCurrentUserHandler, sl))
	usersApp.Delete("/account", Handle(deleteAccountHandler, sl))
	usersApp.Patch("/account", Handle(updateUsernameHandler, sl))
	usersApp.Patch("/password", Handle(updatePasswordHandler, sl))

	chatsApp := app.Group("/chats", middlewareManager.AuthMiddleware)
	chatsApp.Get("/", Handle(getChatsHandler, sl))
	chatsApp.Get("/:id", Handle(getChatByIDHandler, sl))
	chatsApp.Delete("/:id", Handle(deleteChatHandler, sl))

	filesApp := app.Group("/files", middlewareManager.AuthMiddleware)
	filesApp.Post("/upload", Handle(uploadFileHandler, sl))
	filesApp.Get("/", Handle(getFilesHandler, sl))
	filesApp.Delete("/:id", Handle(deleteFileByIdHandler, sl))
	filesApp.Get("/:id", Handle(getFileHandler, sl))

	messagesApp := app.Group("/messages", middlewareManager.AuthMiddleware)
	messagesApp.Post("/", Handle(sendMessageHandler, sl))

	if !domain.IsProdEnv() {
		app.Get("/swagger/*", fiberSwagger.WrapHandler)
	}

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Welcome to MedAIist-Backend API",
			"code":    http.StatusOK,
		})
	})

	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(domain.Error{
			Message: "endpoint not found",
			Code:    http.StatusNotFound,
		})
	})
}
