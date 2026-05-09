# Graph Report - .  (2026-05-09)

## Corpus Check
- Corpus is ~44,087 words - fits in a single context window. You may not need a graph.

## Summary
- 728 nodes · 874 edges · 64 communities (47 shown, 17 thin omitted)
- Extraction: 87% EXTRACTED · 13% INFERRED · 0% AMBIGUOUS · INFERRED: 113 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Repository Components|Repository Components]]
- [[_COMMUNITY_Handler Components|Handler Components]]
- [[_COMMUNITY_Quiz Components|Quiz Components]]
- [[_COMMUNITY_Message Components|Message Components]]
- [[_COMMUNITY_Pdf Chunking Components|Pdf Chunking Components]]
- [[_COMMUNITY_Chat Components|Chat Components]]
- [[_COMMUNITY_User Service Components|User Service Components]]
- [[_COMMUNITY_File Chunk Components|File Chunk Components]]
- [[_COMMUNITY_Openrouter Components|Openrouter Components]]
- [[_COMMUNITY_In-Memory Client Components|In-Memory Client Components]]
- [[_COMMUNITY_User Handlers Components|User Handlers Components]]
- [[_COMMUNITY_Auth Handlers Components|Auth Handlers Components]]
- [[_COMMUNITY_JWE Service Components|JWE Service Components]]
- [[_COMMUNITY_Flashcard Components|Flashcard Components]]
- [[_COMMUNITY_Streak Service Components|Streak Service Components]]
- [[_COMMUNITY_Fiber Middleware Components|Fiber Middleware Components]]
- [[_COMMUNITY_OAuth User Components|OAuth User Components]]
- [[_COMMUNITY_Plan Limits Subscription|Plan Limits Subscription]]
- [[_COMMUNITY_Qdrant Vector DB Components|Qdrant Vector DB Components]]
- [[_COMMUNITY_PDF Processor Components|PDF Processor Components]]
- [[_COMMUNITY_File Handlers Fiber|File Handlers Fiber]]
- [[_COMMUNITY_OAuth Provider Components|OAuth Provider Components]]
- [[_COMMUNITY_Fiber Dependency Components|Fiber Dependency Components]]
- [[_COMMUNITY_Subscription Service Components|Subscription Service Components]]
- [[_COMMUNITY_S3 Service Components|S3 Service Components]]
- [[_COMMUNITY_Auth Handlers Components|Auth Handlers Components]]
- [[_COMMUNITY_Chat Service Components|Chat Service Components]]
- [[_COMMUNITY_Image Processor Components|Image Processor Components]]
- [[_COMMUNITY_Main Application Components|Main Application Components]]
- [[_COMMUNITY_OAuth Handlers Components|OAuth Handlers Components]]
- [[_COMMUNITY_Mailersend Email Components|Mailersend Email Components]]
- [[_COMMUNITY_System Handler Components|System Handler Components]]
- [[_COMMUNITY_Healthcheck Handler|Healthcheck Handler]]
- [[_COMMUNITY_Fiber Cookie Service|Fiber Cookie Service]]
- [[_COMMUNITY_Fiber Handlers Components|Fiber Handlers Components]]
- [[_COMMUNITY_Jina Reranker Components|Jina Reranker Components]]
- [[_COMMUNITY_Auth Cache Components|Auth Cache Components]]
- [[_COMMUNITY_Mobile OAuth Components|Mobile OAuth Components]]
- [[_COMMUNITY_Slog Logger Components|Slog Logger Components]]
- [[_COMMUNITY_Streak Repository|Streak Repository]]
- [[_COMMUNITY_Subscription DB Components|Subscription DB Components]]
- [[_COMMUNITY_Notification Service|Notification Service]]
- [[_COMMUNITY_Validator Components|Validator Components]]
- [[_COMMUNITY_FCM Client Components|FCM Client Components]]
- [[_COMMUNITY_Interfaces Mail Service|Interfaces Mail Service]]
- [[_COMMUNITY_Error Components|Error Components]]
- [[_COMMUNITY_Jobs Cron Components|Jobs Cron Components]]
- [[_COMMUNITY_Flashcard Handler Components|Flashcard Handler Components]]
- [[_COMMUNITY_Postgres DB Components|Postgres DB Components]]
- [[_COMMUNITY_Email Claims Components|Email Claims Components]]
- [[_COMMUNITY_Validator Domain Components|Validator Domain Components]]
- [[_COMMUNITY_Logger Domain Components|Logger Domain Components]]

## God Nodes (most connected - your core abstractions)
1. `Repository` - 75 edges
2. `SetupRoutes()` - 34 edges
3. `GetUserID()` - 25 edges
4. `PdfChunkingService` - 24 edges
5. `NewAuthDependencyContainer()` - 19 edges
6. `OpenRouterService` - 11 edges
7. `InMemoryClient` - 11 edges
8. `UserHandler` - 10 edges
9. `UserService` - 9 edges
10. `QdrantVectorDB` - 9 edges

## Surprising Connections (you probably didn't know these)
- `NewAuthDependencyContainer()` --calls--> `NewNotificationService()`  [INFERRED]
  infrastructure/fiber/dependency_containers.go → services/notification.go
- `NewAuthDependencyContainer()` --calls--> `NewSubscriptionService()`  [INFERRED]
  infrastructure/fiber/dependency_containers.go → services/subscription.go
- `SetupRoutes()` --calls--> `NewPdfProcessor()`  [INFERRED]
  infrastructure/fiber/router.go → services/file.go
- `SetupRoutes()` --calls--> `NewImageProcessor()`  [INFERRED]
  infrastructure/fiber/router.go → services/file.go
- `SetupRoutes()` --calls--> `NewFlashcardService()`  [INFERRED]
  infrastructure/fiber/router.go → services/flashcard.go

## Communities (64 total, 17 thin omitted)

### Community 1 - "Handler Components"
Cohesion: 0.07
Nodes (11): GetUserID(), FileHandler, FlashcardHandler, NewMessageHandler(), MessageConfig, MessageHandler, QuizHandler, NewSubscriptionHandler() (+3 more)

### Community 2 - "Quiz Components"
Cohesion: 0.05
Nodes (32): Answer, AnswerFeedback, AttemptAnswerDetail, AttemptDetailResponse, ChunkService, GenerateQuizRequest, GenerateQuizResponse, GetAttemptByIdRequest (+24 more)

### Community 3 - "Message Components"
Cohesion: 0.09
Nodes (24): Attachment, NewChat(), ContextSnapshot, Message, NewMessage(), MessageRepository, MessageService, SendMessageRequest (+16 more)

### Community 4 - "Pdf Chunking Components"
Cohesion: 0.11
Nodes (10): ChunkingService, extractedData, pageRange, PdfChunkingService, QualityMetrics, section, getPagesForRange(), NewChunkingService() (+2 more)

### Community 5 - "Chat Components"
Cohesion: 0.06
Nodes (27): NewChatHandler(), ChatConfig, ChatHandler, AIService, Chat, ChatMessage, ChatRepository, ChatRequest (+19 more)

### Community 6 - "User Service Components"
Cohesion: 0.08
Nodes (12): NotificationPreferences, User, IsPasswordTooShort(), NewUser(), ValidateUsername(), UserPreferences, UserRepository, UserService (+4 more)

### Community 7 - "File Chunk Components"
Cohesion: 0.07
Nodes (27): Chunk, Chunker, ChunkMetadata, ChunkOptions, DeleteFileRequest, DeleteFileResponse, Embedding, EmbeddingBatchRequest (+19 more)

### Community 8 - "Openrouter Components"
Cohesion: 0.1
Nodes (17): ChatCompletionChoice, ChatCompletionMessage, ChatCompletionRequest, ChatCompletionResponse, ChatMessageImageURL, ChatMessagePart, ChatMessagePartType, Config (+9 more)

### Community 9 - "In-Memory Client Components"
Cohesion: 0.13
Nodes (8): NewInMemoryClient(), InMemoryClient, InMemoryConfig, FacebookVerifier, GoogleVerifier, TokenVerifier, NewFacebookVerifier(), NewGoogleVerifier()

### Community 10 - "User Handlers Components"
Cohesion: 0.09
Nodes (21): ChangePasswordRequest, ChangePasswordResponse, DeleteAccountRequest, DeleteAccountResponse, ForgotPasswordRequest, ForgotPasswordResponse, GetCurrentUserRequest, GetCurrentUserResponse (+13 more)

### Community 11 - "Auth Handlers Components"
Cohesion: 0.11
Nodes (13): NewAuthHandler(), AuthConfig, AuthHandler, ExchangeRequest, ExchangeResponse, LoginRequest, LoginResponse, LogoutRequest (+5 more)

### Community 12 - "JWE Service Components"
Cohesion: 0.18
Nodes (9): AuthClaims, Config, NewJWETokenService(), EmailClaims, decryptClaims(), encryptClaims(), is32ByteKey(), isExpired() (+1 more)

### Community 13 - "Flashcard Components"
Cohesion: 0.11
Nodes (17): AddCardRequest, AddCardResponse, CreateDeckRequest, CreateDeckResponse, Deck, Flashcard, FlashcardRepository, FlashcardReview (+9 more)

### Community 14 - "Streak Service Components"
Cohesion: 0.13
Nodes (5): NewFlashcardService(), flashcardService, appendRecentDate(), NewStreakService(), StreakService

### Community 15 - "Fiber Middleware Components"
Cohesion: 0.16
Nodes (8): ContextKey, GetUserIDFromString(), FiberContextKey, messageTypeProbe, NewMiddlewareManager(), MiddlewareManager, detectMessageActionType(), getUsageLimitError()

### Community 16 - "OAuth User Components"
Cohesion: 0.14
Nodes (5): NewOAuthUser(), OAuthCloudUserInfo, OAuthSession, OAuthUser, OAuthService

### Community 17 - "Plan Limits Subscription"
Cohesion: 0.13
Nodes (13): ActionType, GetUsageStatusRequest, LimitPeriod, PlanLimits, PlanType, RevenueCatWebhookEvent, Subscription, SubscriptionRepository (+5 more)

### Community 18 - "Qdrant Vector DB Components"
Cohesion: 0.2
Nodes (5): convertPageNumbers(), NewQdrantVectorDB(), pointerUint32(), Config, QdrantVectorDB

### Community 19 - "PDF Processor Components"
Cohesion: 0.21
Nodes (4): GetFilesResponse, FileProcessorFactory, FileService, PdfProcessor

### Community 20 - "File Handlers Fiber"
Cohesion: 0.15
Nodes (9): handlerAdapter, handlerAdapter[R, Res], SetupRoutes(), NewFileHandler(), FileConfig, NewQuizHandler(), QuizConfig, NewFileProcessorFactory() (+1 more)

### Community 21 - "OAuth Provider Components"
Cohesion: 0.21
Nodes (4): generateSessionKey(), OAuthDependenciesConfig, OAuthProvider, generateState()

### Community 22 - "Fiber Dependency Components"
Cohesion: 0.22
Nodes (8): DefaultUsageLimitsConfig(), IsProdEnv(), getEnv(), NewAuthDependencyContainer(), DependencyContainer, MultiModelService, NewFCMClient(), NewOpenRouterService()

### Community 24 - "S3 Service Components"
Cohesion: 0.22
Nodes (4): S3Config, S3Service, hasExtension(), NewS3Service()

### Community 27 - "Image Processor Components"
Cohesion: 0.33
Nodes (5): getFileExtension(), isImageFile(), NewImageProcessor(), NewPdfProcessor(), ImageProcessor

### Community 28 - "Main Application Components"
Cohesion: 0.43
Nodes (5): gracefulShutdown(), main(), startServer(), registerMiddlewares(), SetupServer()

### Community 29 - "OAuth Handlers Components"
Cohesion: 0.33
Nodes (5): NewFacebookOAuthService(), NewGoogleOAuthService(), NewOAuthProvider(), OAuthTokens, NewOAuthService()

### Community 31 - "System Handler Components"
Cohesion: 0.33
Nodes (4): NewSystemConfigHandler(), SystemConfigHandler, SystemConfigRequest, VersionResponse

### Community 32 - "Healthcheck Handler"
Cohesion: 0.33
Nodes (4): NewHealthcheckHandler(), HealthcheckHandler, HealthcheckRequest, HealthcheckResponse

### Community 34 - "Fiber Handlers Components"
Cohesion: 0.4
Nodes (5): Handle(), handleError(), HandlerInterface, Request, Response

### Community 35 - "Jina Reranker Components"
Cohesion: 0.33
Nodes (4): RerankRequest, RerankResponse, Service, NewService()

### Community 36 - "Auth Cache Components"
Cohesion: 0.33
Nodes (5): AuthCache, AuthRepository, AuthService, OAuthUserRegistrationData, SignupData

### Community 37 - "Mobile OAuth Components"
Cohesion: 0.4
Nodes (3): NewMobileOAuthHandler(), MobileLoginRequest, MobileOAuthHandler

### Community 39 - "Streak Repository"
Cohesion: 0.4
Nodes (4): StreakRepository, StreakResponse, StreakService, UsageStats

### Community 43 - "FCM Client Components"
Cohesion: 0.5
Nodes (3): FCMClient, NotificationRepository, NotificationService

### Community 44 - "Interfaces Mail Service"
Cohesion: 0.5
Nodes (3): MailService, TokenPayload, TokenService

## Knowledge Gaps
- **216 isolated node(s):** `questionResult`, `searchResult`, `RerankRequest`, `RerankResponse`, `ChunkingService` (+211 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **17 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `SetupRoutes()` connect `File Handlers Fiber` to `Handler Components`, `Quiz Components`, `Message Components`, `Chat Components`, `User Service Components`, `In-Memory Client Components`, `User Handlers Components`, `Auth Handlers Components`, `Streak Service Components`, `Fiber Middleware Components`, `Fiber Dependency Components`, `Chat Service Components`, `Image Processor Components`, `Main Application Components`, `OAuth Handlers Components`, `System Handler Components`, `Healthcheck Handler`, `Mobile OAuth Components`, `Jobs Cron Components`, `Flashcard Handler Components`?**
  _High betweenness centrality (0.585) - this node is a cross-community bridge._
- **Why does `NewAuthDependencyContainer()` connect `Fiber Dependency Components` to `Jina Reranker Components`, `Pdf Chunking Components`, `Slog Logger Components`, `In-Memory Client Components`, `Validator Components`, `Notification Service`, `JWE Service Components`, `Postgres DB Components`, `Qdrant Vector DB Components`, `File Handlers Fiber`, `Subscription Service Components`, `S3 Service Components`, `Auth Handlers Components`, `Mailersend Email Components`?**
  _High betweenness centrality (0.496) - this node is a cross-community bridge._
- **Why does `Repository` connect `Repository Components` to `Subscription DB Components`, `Auth Handlers Components`?**
  _High betweenness centrality (0.176) - this node is a cross-community bridge._
- **Are the 32 inferred relationships involving `SetupRoutes()` (e.g. with `main()` and `NewAuthDependencyContainer()`) actually correct?**
  _`SetupRoutes()` has 32 INFERRED edges - model-reasoned connections that need verification._
- **Are the 24 inferred relationships involving `GetUserID()` (e.g. with `.HandleCreateDeck()` and `.HandleGetDecks()`) actually correct?**
  _`GetUserID()` has 24 INFERRED edges - model-reasoned connections that need verification._
- **Are the 17 inferred relationships involving `NewAuthDependencyContainer()` (e.g. with `SetupRoutes()` and `NewDB()`) actually correct?**
  _`NewAuthDependencyContainer()` has 17 INFERRED edges - model-reasoned connections that need verification._
- **What connects `questionResult`, `searchResult`, `RerankRequest` to the rest of the system?**
  _216 weakly-connected nodes found - possible documentation gaps or missing edges._