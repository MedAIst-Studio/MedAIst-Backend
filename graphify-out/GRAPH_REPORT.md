# Graph Report - .  (2026-05-09)

## Corpus Check
- Corpus is ~41,363 words - fits in a single context window. You may not need a graph.

## Summary
- 679 nodes · 821 edges · 46 communities (36 shown, 10 thin omitted)
- Extraction: 87% EXTRACTED · 13% INFERRED · 0% AMBIGUOUS · INFERRED: 105 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Repository Operations|Repository Operations]]
- [[_COMMUNITY_App Entry and Handlers|App Entry and Handlers]]
- [[_COMMUNITY_Infrastructure Services|Infrastructure Services]]
- [[_COMMUNITY_File Processing Domain|File Processing Domain]]
- [[_COMMUNITY_Quiz Domain|Quiz Domain]]
- [[_COMMUNITY_Chat and Messages|Chat and Messages]]
- [[_COMMUNITY_PDF Chunking Service|PDF Chunking Service]]
- [[_COMMUNITY_Chat Domain|Chat Domain]]
- [[_COMMUNITY_Core Handlers|Core Handlers]]
- [[_COMMUNITY_User Auth Domain|User Auth Domain]]
- [[_COMMUNITY_OpenRouter LLM Integration|OpenRouter LLM Integration]]
- [[_COMMUNITY_In-Memory Caching|In-Memory Caching]]
- [[_COMMUNITY_User Account Handlers|User Account Handlers]]
- [[_COMMUNITY_Auth Handlers|Auth Handlers]]
- [[_COMMUNITY_JWE Token Config|JWE Token Config]]
- [[_COMMUNITY_Fiber Middlewares|Fiber Middlewares]]
- [[_COMMUNITY_OAuth User Logic|OAuth User Logic]]
- [[_COMMUNITY_Subscription Domain|Subscription Domain]]
- [[_COMMUNITY_Qdrant Vector DB|Qdrant Vector DB]]
- [[_COMMUNITY_OAuth Callback Flow|OAuth Callback Flow]]
- [[_COMMUNITY_Subscription Service|Subscription Service]]
- [[_COMMUNITY_Repository Auth|Repository Auth]]
- [[_COMMUNITY_Chat Service|Chat Service]]
- [[_COMMUNITY_Streak Service|Streak Service]]
- [[_COMMUNITY_Cookie Service|Cookie Service]]
- [[_COMMUNITY_Fiber Handler Interfaces|Fiber Handler Interfaces]]
- [[_COMMUNITY_Auth Cache Domain|Auth Cache Domain]]
- [[_COMMUNITY_Streak Domain|Streak Domain]]
- [[_COMMUNITY_Subscription Repository|Subscription Repository]]
- [[_COMMUNITY_Notification Domain|Notification Domain]]
- [[_COMMUNITY_Service Interfaces|Service Interfaces]]
- [[_COMMUNITY_Error Domain|Error Domain]]
- [[_COMMUNITY_Email Domain|Email Domain]]
- [[_COMMUNITY_Validator Domain|Validator Domain]]
- [[_COMMUNITY_Logger Domain|Logger Domain]]

## God Nodes (most connected - your core abstractions)
1. `Repository` - 64 edges
2. `SetupRoutes()` - 32 edges
3. `PdfChunkingService` - 24 edges
4. `NewAuthDependencyContainer()` - 19 edges
5. `GetUserID()` - 19 edges
6. `OpenRouterService` - 11 edges
7. `InMemoryClient` - 11 edges
8. `UserHandler` - 10 edges
9. `UserService` - 9 edges
10. `QdrantVectorDB` - 9 edges

## Surprising Connections (you probably didn't know these)
- `NewAuthDependencyContainer()` --calls--> `NewSubscriptionService()`  [INFERRED]
  infrastructure/fiber/dependency_containers.go → services/subscription.go
- `SetupRoutes()` --calls--> `NewQuizService()`  [INFERRED]
  infrastructure/fiber/router.go → services/quiz.go
- `SetupRoutes()` --calls--> `NewUserService()`  [INFERRED]
  infrastructure/fiber/router.go → services/user.go
- `SetupRoutes()` --calls--> `NewChatService()`  [INFERRED]
  infrastructure/fiber/router.go → services/chat.go
- `SetupRoutes()` --calls--> `NewMessageService()`  [INFERRED]
  infrastructure/fiber/router.go → services/message.go

## Communities (46 total, 10 thin omitted)

### Community 1 - "App Entry and Handlers"
Cohesion: 0.05
Nodes (37): gracefulShutdown(), main(), startServer(), getEnv(), handlerAdapter, handlerAdapter[R, Res], NewFacebookOAuthService(), NewGoogleOAuthService() (+29 more)

### Community 2 - "Infrastructure Services"
Cohesion: 0.05
Nodes (23): DefaultUsageLimitsConfig(), NewAuthDependencyContainer(), DependencyContainer, MultiModelService, RerankRequest, RerankResponse, Service, NewService() (+15 more)

### Community 3 - "File Processing Domain"
Cohesion: 0.05
Nodes (31): Chunk, Chunker, ChunkMetadata, ChunkOptions, DeleteFileRequest, DeleteFileResponse, Embedding, EmbeddingBatchRequest (+23 more)

### Community 4 - "Quiz Domain"
Cohesion: 0.05
Nodes (32): Answer, AnswerFeedback, AttemptAnswerDetail, AttemptDetailResponse, ChunkService, GenerateQuizRequest, GenerateQuizResponse, GetAttemptByIdRequest (+24 more)

### Community 5 - "Chat and Messages"
Cohesion: 0.09
Nodes (24): Attachment, NewChat(), ContextSnapshot, Message, NewMessage(), MessageRepository, MessageService, SendMessageRequest (+16 more)

### Community 6 - "PDF Chunking Service"
Cohesion: 0.11
Nodes (10): ChunkingService, extractedData, pageRange, PdfChunkingService, QualityMetrics, section, getPagesForRange(), NewChunkingService() (+2 more)

### Community 7 - "Chat Domain"
Cohesion: 0.06
Nodes (27): NewChatHandler(), ChatConfig, ChatHandler, AIService, Chat, ChatMessage, ChatRepository, ChatRequest (+19 more)

### Community 8 - "Core Handlers"
Cohesion: 0.09
Nodes (9): GetUserID(), FileHandler, NewMessageHandler(), MessageConfig, MessageHandler, NewQuizHandler(), QuizConfig, QuizHandler (+1 more)

### Community 9 - "User Auth Domain"
Cohesion: 0.08
Nodes (12): NotificationPreferences, User, IsPasswordTooShort(), NewUser(), ValidateUsername(), UserPreferences, UserRepository, UserService (+4 more)

### Community 10 - "OpenRouter LLM Integration"
Cohesion: 0.1
Nodes (17): ChatCompletionChoice, ChatCompletionMessage, ChatCompletionRequest, ChatCompletionResponse, ChatMessageImageURL, ChatMessagePart, ChatMessagePartType, Config (+9 more)

### Community 11 - "In-Memory Caching"
Cohesion: 0.13
Nodes (8): NewInMemoryClient(), InMemoryClient, InMemoryConfig, FacebookVerifier, GoogleVerifier, TokenVerifier, NewFacebookVerifier(), NewGoogleVerifier()

### Community 12 - "User Account Handlers"
Cohesion: 0.09
Nodes (21): ChangePasswordRequest, ChangePasswordResponse, DeleteAccountRequest, DeleteAccountResponse, ForgotPasswordRequest, ForgotPasswordResponse, GetCurrentUserRequest, GetCurrentUserResponse (+13 more)

### Community 13 - "Auth Handlers"
Cohesion: 0.11
Nodes (13): NewAuthHandler(), AuthConfig, AuthHandler, ExchangeRequest, ExchangeResponse, LoginRequest, LoginResponse, LogoutRequest (+5 more)

### Community 14 - "JWE Token Config"
Cohesion: 0.18
Nodes (9): AuthClaims, Config, NewJWETokenService(), EmailClaims, decryptClaims(), encryptClaims(), is32ByteKey(), isExpired() (+1 more)

### Community 15 - "Fiber Middlewares"
Cohesion: 0.13
Nodes (10): ContextKey, GetUserIDFromString(), IsProdEnv(), FiberContextKey, messageTypeProbe, NewMiddlewareManager(), MiddlewareManager, detectMessageActionType() (+2 more)

### Community 16 - "OAuth User Logic"
Cohesion: 0.12
Nodes (7): NewOAuthUser(), OAuthCloudUserInfo, OAuthSession, OAuthUser, OAuthService, OAuthTokens, NewOAuthService()

### Community 17 - "Subscription Domain"
Cohesion: 0.13
Nodes (13): ActionType, GetUsageStatusRequest, LimitPeriod, PlanLimits, PlanType, RevenueCatWebhookEvent, Subscription, SubscriptionRepository (+5 more)

### Community 18 - "Qdrant Vector DB"
Cohesion: 0.2
Nodes (5): convertPageNumbers(), NewQdrantVectorDB(), pointerUint32(), Config, QdrantVectorDB

### Community 19 - "OAuth Callback Flow"
Cohesion: 0.21
Nodes (4): generateSessionKey(), OAuthDependenciesConfig, OAuthProvider, generateState()

### Community 23 - "Streak Service"
Cohesion: 0.4
Nodes (3): appendRecentDate(), NewStreakService(), StreakService

### Community 25 - "Fiber Handler Interfaces"
Cohesion: 0.4
Nodes (5): Handle(), handleError(), HandlerInterface, Request, Response

### Community 26 - "Auth Cache Domain"
Cohesion: 0.33
Nodes (5): AuthCache, AuthRepository, AuthService, OAuthUserRegistrationData, SignupData

### Community 27 - "Streak Domain"
Cohesion: 0.4
Nodes (4): StreakRepository, StreakResponse, StreakService, UsageStats

### Community 29 - "Notification Domain"
Cohesion: 0.5
Nodes (3): FCMClient, NotificationRepository, NotificationService

### Community 30 - "Service Interfaces"
Cohesion: 0.5
Nodes (3): MailService, TokenPayload, TokenService

## Knowledge Gaps
- **198 isolated node(s):** `questionResult`, `searchResult`, `RerankRequest`, `RerankResponse`, `ChunkingService` (+193 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **10 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `SetupRoutes()` connect `App Entry and Handlers` to `Infrastructure Services`, `Quiz Domain`, `Chat and Messages`, `Chat Domain`, `Core Handlers`, `User Auth Domain`, `In-Memory Caching`, `User Account Handlers`, `Auth Handlers`, `Fiber Middlewares`, `OAuth User Logic`, `Chat Service`, `Streak Service`?**
  _High betweenness centrality (0.611) - this node is a cross-community bridge._
- **Why does `NewAuthDependencyContainer()` connect `Infrastructure Services` to `App Entry and Handlers`, `PDF Chunking Service`, `In-Memory Caching`, `JWE Token Config`, `Fiber Middlewares`, `Qdrant Vector DB`, `Subscription Service`, `Repository Auth`?**
  _High betweenness centrality (0.523) - this node is a cross-community bridge._
- **Why does `NewRepository()` connect `Repository Auth` to `Infrastructure Services`?**
  _High betweenness centrality (0.167) - this node is a cross-community bridge._
- **Are the 30 inferred relationships involving `SetupRoutes()` (e.g. with `main()` and `NewAuthDependencyContainer()`) actually correct?**
  _`SetupRoutes()` has 30 INFERRED edges - model-reasoned connections that need verification._
- **Are the 17 inferred relationships involving `NewAuthDependencyContainer()` (e.g. with `SetupRoutes()` and `NewDB()`) actually correct?**
  _`NewAuthDependencyContainer()` has 17 INFERRED edges - model-reasoned connections that need verification._
- **Are the 18 inferred relationships involving `GetUserID()` (e.g. with `.HandleUploadFile()` and `.HandleGetFiles()`) actually correct?**
  _`GetUserID()` has 18 INFERRED edges - model-reasoned connections that need verification._
- **What connects `questionResult`, `searchResult`, `RerankRequest` to the rest of the system?**
  _198 weakly-connected nodes found - possible documentation gaps or missing edges._