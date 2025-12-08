# MedAIst-Backend Architecture Documentation

## 📋 Table of Contents

- [Overview](#overview)
- [Architecture Pattern](#architecture-pattern)
- [Directory Structure](#directory-structure)
- [Layer Descriptions](#layer-descriptions)
- [Core Features](#core-features)
- [Technology Stack](#technology-stack)
- [Data Flow](#data-flow)
- [API Endpoints](#api-endpoints)

---

## Overview

**MedAIst-Backend** is an AI-powered medical education platform built with Go. The system enables users to upload PDF documents (medical texts, lecture notes), chat with AI about the content, and generate interactive quizzes based on the material.

**Key Capabilities:**

- PDF document processing and vector embedding
- RAG (Retrieval-Augmented Generation) based chat system
- AI-powered quiz generation
- User authentication & OAuth integration
- Real-time query expansion and semantic search

---

## Architecture Pattern

This project follows **Clean Architecture** (Hexagonal Architecture) principles:

```
┌─────────────────────────────────────────────────┐
│          Interface Layer (Fiber)                │
│         (HTTP, Routes, Middleware)              │
└──────────────────┬──────────────────────────────┘
                   │
┌──────────────────▼──────────────────────────────┐
│         Application Layer (app/)                │
│    (Use Cases, Handlers, Interfaces)            │
└──────────────────┬──────────────────────────────┘
                   │
┌──────────────────▼──────────────────────────────┐
│          Domain Layer (domain/)                 │
│     (Entities, Business Rules, Errors)          │
└─────────────────────────────────────────────────┘
                   ▲
┌──────────────────┴──────────────────────────────┐
│      Infrastructure Layer (infrastructure/)     │
│  (DB, AI APIs, Storage, External Services)      │
└─────────────────────────────────────────────────┘
```

**Benefits:**

- **Independence:** Domain layer has zero external dependencies
- **Testability:** Easy to mock infrastructure implementations
- **Flexibility:** Swap implementations without changing business logic
- **Maintainability:** Clear separation of concerns

---

## Directory Structure

```
MedAIst-Backend/
├── cmd/
│   └── main.go                    # Entry point, server initialization
│
├── domain/                        # 🔵 Domain Layer (Pure business entities)
│   ├── user.go                    # User entity, password hashing
│   ├── quiz.go                    # Quiz and QuizQuestion entities
│   ├── chat.go                    # Chat entity
│   ├── file.go                    # File entity, validation
│   ├── message.go                 # Message entity
│   ├── error.go                   # Domain-specific errors
│   ├── chat_api.go                # Chat request/response types
│   └── ...
│
├── app/                           # 🟢 Application Layer (Use cases)
│   ├── auth/                      # Authentication handlers
│   │   ├── signup_handler.go
│   │   ├── login_handler.go
│   │   ├── logout_handler.go
│   │   ├── refresh_token_handler.go
│   │   ├── exchange_handler.go   # OAuth token exchange
│   │   └── interfaces.go         # Repository interfaces
│   │
│   ├── user/                      # User management handlers
│   │   ├── delete_account_handler.go
│   │   ├── update_username_handler.go
│   │   ├── change_password_handler.go
│   │   ├── forgot_password_handler.go
│   │   └── ...
│   │
│   ├── file/                      # File upload/management
│   │   ├── upload_file_handler.go # PDF upload, chunking, embedding
│   │   ├── get_files_handler.go
│   │   ├── delete_file_handler.go
│   │   └── interfaces.go
│   │
│   ├── chat/                      # Chat management
│   │   ├── get_chats_handler.go
│   │   ├── get_chat_by_id_handler.go
│   │   ├── delete_chat_handler.go
│   │   └── interfaces.go
│   │
│   ├── message/                   # Message handling (RAG chat)
│   │   ├── send_message.go       # RAG pipeline, query expansion
│   │   └── interfaces.go
│   │
│   └── quiz/                      # Quiz generation & solving
│       ├── generate_quiz_handler.go # AI quiz generation
│       ├── get_quizzes_handler.go
│       ├── solve_quiz_handler.go
│       ├── get_attempts_handler.go
│       └── interfaces.go
│
├── infrastructure/                # 🟡 Infrastructure Layer (Implementations)
│   ├── fiber/                     # Web framework setup
│   │   ├── server.go             # Fiber app initialization
│   │   ├── router.go             # Route definitions
│   │   ├── handler.go            # Generic handler wrapper
│   │   ├── middleware.go         # Auth middleware
│   │   ├── dependency_containers.go # Dependency injection
│   │   └── oauth_providers.go    # OAuth implementations
│   │
│   ├── postgres/                  # PostgreSQL repository
│   │   ├── repository.go         # DB connection pool
│   │   ├── auth.go               # User CRUD operations
│   │   ├── chat.go               # Chat queries
│   │   ├── file.go               # File metadata queries
│   │   ├── message.go            # Message storage
│   │   ├── quiz.go               # Quiz data storage
│   │   └── user.go               # User profile operations
│   │
│   ├── qdrant/                    # Vector database
│   │   └── client.go             # Embedding storage & retrieval
│   │
│   ├── openai/                    # OpenAI integration
│   │   └── service.go            # Chat, quiz generation, query classification
│   │
│   ├── jina/                      # Jina AI reranker
│   │   └── service.go            # Chunk reranking
│   │
│   ├── embedding/                 # Embedding service
│   │   └── service.go            # Text → vector conversion
│   │
│   ├── chunker/                   # PDF chunking
│   │   └── service.go            # PDF text extraction & chunking
│   │
│   ├── minio/                     # Object storage
│   │   └── service.go            # File upload/download
│   │
│   ├── jwe/                       # JSON Web Encryption
│   │   └── service.go            # Token generation/validation
│   │
│   ├── mailersend/                # Email service
│   │   └── service.go            # Email verification, password reset
│   │
│   ├── in-memory/                 # In-memory cache
│   │   └── client.go             # OAuth state, sessions
│   │
│   ├── slog/                      # Logging
│   │   └── logger.go
│   │
│   └── validator/                 # Input validation
│       └── validator.go
│
├── docs/                          # Swagger documentation (auto-generated)
├── docker-compose.yml             # Local development environment
├── Dockerfile                     # Container build
├── Makefile                       # Build commands
└── go.mod                         # Dependencies
```

---

## Layer Descriptions

### 1️⃣ Domain Layer (`domain/`)

**Pure business entities with zero external dependencies.**

#### Key Files:

- **`user.go`**: User entity with password hashing (bcrypt), username validation
- **`quiz.go`**: Quiz, QuizQuestion, Option entities
- **`chat.go`**: Chat entity linking users to PDFs
- **`file.go`**: File entity with PDF validation (max 10MB)
- **`message.go`**: Message entity for chat history
- **`error.go`**: Centralized domain errors (`ErrInvalidCredentials`, `ErrFileSizeExceeded`, etc.)
- **`chat_api.go`**: Request/response types for AI chat (`ChatRequest`, `QueryType`, prompts)

#### Example Entity:

```go
type Quiz struct {
    Id         uuid.UUID
    UserId     uuid.UUID
    DocumentId uuid.UUID
    Title      string
    Questions  []QuizQuestion
}
```

---

### 2️⃣ Application Layer (`app/`)

**Orchestrates business logic using domain entities and infrastructure interfaces.**

#### Pattern:

Each feature module defines:

- **Handlers**: Process HTTP requests
- **Interfaces**: Define required repositories/services (Dependency Inversion)
- **DTOs**: Request/response models

#### Example Handler Flow (`app/quiz/generate_quiz_handler.go`):

1. Extract user ID from context
2. Fetch chunks from vector DB via `ChunkService`
3. Generate questions in parallel using `AIService`
4. Save quiz using `Repository`
5. Return quiz ID

#### Key Modules:

- **auth/**: Signup, login, OAuth, token refresh
- **file/**: PDF upload → chunking → embedding → Qdrant storage
- **message/**: RAG pipeline (query expansion → embedding → vector search → reranking → LLM)
- **quiz/**: Quiz generation (chunk → OpenAI → structured questions)

---

### 3️⃣ Infrastructure Layer (`infrastructure/`)

**Concrete implementations of application interfaces.**

#### Database (`postgres/`)

- Uses `lib/pq` driver with connection pooling
- SQL queries for CRUD operations
- Transaction support for quiz attempts + answers

#### Vector Database (`qdrant/`)

- gRPC client for Qdrant
- Stores PDF chunks as embeddings (1024-dim vectors)
- Semantic search with cosine similarity (min score: 0.5)
- Configurable chunk limit (default: 10)

#### AI Services (`openai/`)

**Uses GPT-4o and GPT-4o-mini for:**

1. **Chat**: Standard conversation
2. **Query Classification**: Determines if query needs PDF search
3. **Query Expansion**: Generates 3 related queries for better retrieval
4. **Chat Title Generation**: Creates chat titles
5. **Quiz Generation**: Structured JSON output with 5 options per question

#### Jina Reranker (`jina/`)

- Reranks retrieved chunks by relevance to user query
- Uses `jina-reranker-v3` model
- Returns top-3 chunks

#### File Storage (`minio/`)

- S3-compatible object storage
- Bucket: `files`
- File paths: `uploads/{uuid}`

#### Authentication (`jwe/`)

- JWE tokens (not JWT) for enhanced security
- Access token: 15 min expiry
- Refresh token: 30 day expiry
- Secure email tokens: 11 min expiry

---

### 4️⃣ Interface Layer (`cmd/`, `infrastructure/fiber/`)

#### Routing (`infrastructure/fiber/router.go`)

**Base URL: `http://localhost:3000`**

```go
/auth/signup           POST
/auth/login            POST
/auth/logout           POST
/auth/refresh          POST
/auth/google           GET  (OAuth redirect)
/auth/facebook         GET  (OAuth redirect)

/users/profile         GET  (requires auth)
/users/account         DELETE, PATCH
/users/password        PATCH
/users/forgot-password POST
/users/reset-password  POST

/files/upload          POST (requires auth)
/files/                GET
/files/:id             GET, DELETE

/messages/             POST (requires auth, RAG chat)

/chats/                GET
/chats/:id             GET, DELETE

/quizzes/              POST (generate), GET (list)
/quizzes/:id           GET
/quizzes/:id/solve     POST
/quizzes/attempts      GET
/quizzes/attempts/:id  GET

/swagger/*             GET (dev only)
```

#### Dependency Injection (`dependency_containers.go`)

Wires all services together on app startup:

```go
vectorDB := qdrantInfra.NewQdrantVectorDB(...)
openai := openaiInfra.NewOpenAIService(...)
repo := postgresInfra.NewRepository(...)
quizHandler := quiz.NewGenerateQuizHandler(openai, repo, vectorDB)
```

---

## Core Features

### 📄 1. PDF Processing Pipeline

1. **Upload**: User uploads PDF (max 10MB)
2. **Chunking**: PDF split into semantic chunks (max 1024 chars, 128 char overlap)
3. **Embedding**: Each chunk → 1024-dim embedding vector
4. **Storage**: Chunks + embeddings stored in Qdrant

### 💬 2. RAG Chat System

1. **Query Classification**: Is it general or needs PDF search?
2. **Query Expansion**: Generate 3 related queries
3. **Parallel Search**: Search Qdrant for each query
4. **Deduplication**: Remove duplicate chunks
5. **Reranking**: Jina reranks by relevance (top-3)
6. **Context Injection**: Add chunks to LLM context
7. **Response**: GPT-4o generates answer

### 📝 3. Quiz Generation

1. Fetch random chunks from PDF (configurable limit, default 10)
2. Parallel OpenAI calls: chunk → structured quiz question
3. Each question: 5 options (A-E), 1 correct, explanation
4. Save quiz with all questions
5. User can later solve quiz and view results

---

## Technology Stack

### Core

- **Language**: Go 1.24
- **Web Framework**: Fiber v2 (Express-like HTTP framework)
- **Database**: PostgreSQL (via `lib/pq`)
- **Vector DB**: Qdrant (gRPC client)
- **Object Storage**: MinIO (S3-compatible)

### AI/ML

- **LLM**: OpenAI GPT-4o, GPT-4o-mini
- **Embeddings**: Custom API (1024-dim vectors)
- **Reranking**: Jina AI Reranker v3

### Authentication

- **Tokens**: JWE (JSON Web Encryption)
- **OAuth**: Google, Facebook (OAuth2)
- **Password**: bcrypt hashing

### Additional Services

- **Email**: MailerSend
- **Validation**: go-playground/validator
- **Documentation**: Swagger (swaggo)
- **Logging**: slog (structured logging)

---

## Data Flow

### Example: Send Message (RAG Chat)

```
User → POST /messages
  ↓
[Handler] Send Message Handler
  ↓
[Domain] Check if new chat → Generate title
  ↓
[AI] Classify query type (General/PDFSearch/Clarification)
  ↓
[AI] Expand query → 3 related queries
  ↓
[Infrastructure] Generate embeddings in parallel
  ↓
[Qdrant] Vector search for each query
  ↓
[Jina] Rerank chunks by relevance (top-3)
  ↓
[OpenAI] Chat with context (chat history + chunks)
  ↓
[PostgreSQL] Save user message + AI response
  ↓
Response → {chat_id, ai_response, message_ids}
```

### Example: Generate Quiz

```
User → POST /quizzes {fileId, quizName}
  ↓
[Handler] Generate Quiz Handler
  ↓
[Qdrant] Get random chunks (limit = 10)
  ↓
[OpenAI] Parallel request for each chunk
  ├─ Chunk 1 → Question 1
  ├─ Chunk 2 → Question 2
  └─ ...
  ↓
[Domain] Quiz entity with all questions
  ↓
[PostgreSQL] Save quiz + questions
  ↓
Response → {quizId}
```

---

## API Endpoints

### Authentication

| Method | Endpoint         | Description               | Auth |
| ------ | ---------------- | ------------------------- | ---- |
| POST   | `/auth/signup`   | Register new user         | ❌   |
| POST   | `/auth/login`    | Login with email/password | ❌   |
| POST   | `/auth/logout`   | Logout                    | ❌   |
| POST   | `/auth/refresh`  | Refresh access token      | ❌   |
| GET    | `/auth/google`   | OAuth redirect            | ❌   |
| GET    | `/auth/facebook` | OAuth redirect            | ❌   |

### User Management

| Method | Endpoint                 | Description        | Auth |
| ------ | ------------------------ | ------------------ | ---- |
| GET    | `/users/profile`         | Get current user   | ✅   |
| DELETE | `/users/account`         | Delete account     | ✅   |
| PATCH  | `/users/account`         | Update username    | ✅   |
| PATCH  | `/users/password`        | Change password    | ✅   |
| POST   | `/users/forgot-password` | Request reset link | ❌   |
| POST   | `/users/reset-password`  | Reset password     | ❌   |

### Files

| Method | Endpoint        | Description              | Auth |
| ------ | --------------- | ------------------------ | ---- |
| POST   | `/files/upload` | Upload PDF               | ✅   |
| GET    | `/files/`       | List user's files        | ✅   |
| GET    | `/files/:id`    | Get file                 | ✅   |
| DELETE | `/files/:id`    | Delete file + embeddings | ✅   |

### Chat & Messages

| Method | Endpoint     | Description            | Auth |
| ------ | ------------ | ---------------------- | ---- |
| POST   | `/messages/` | Send message (RAG)     | ✅   |
| GET    | `/chats/`    | List user's chats      | ✅   |
| GET    | `/chats/:id` | Get chat with messages | ✅   |
| DELETE | `/chats/:id` | Delete chat            | ✅   |

### Quizzes

| Method | Endpoint                | Description            | Auth |
| ------ | ----------------------- | ---------------------- | ---- |
| POST   | `/quizzes/`             | Generate quiz from PDF | ✅   |
| GET    | `/quizzes/`             | List user's quizzes    | ✅   |
| GET    | `/quizzes/:id`          | Get quiz questions     | ✅   |
| POST   | `/quizzes/:id/solve`    | Submit quiz answers    | ✅   |
| GET    | `/quizzes/attempts`     | List quiz attempts     | ✅   |
| GET    | `/quizzes/attempts/:id` | Get attempt details    | ✅   |

---

## Configuration

**Environment Variables** (see `.env`):

- `DATABASE_URL`: PostgreSQL connection string
- `QDRANT_URL`: Qdrant server address
- `QDRANT_COLLECTION_NAME`: Vector collection name
- `OPENAI_API_KEY`: OpenAI API key
- `JINA_API_KEY`: Jina AI API key
- `EMBEDDING_API_KEY`: Embedding service key
- `EMBEDDING_API_DOMAIN`: Embedding service URL
- `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`: MinIO credentials
- `MAILERSEND_API_KEY`: Email service key
- `PORT`: Server port (default: 3000)

---

## Summary

**MedAIst-Backend** exemplifies Clean Architecture in a production Go application. By separating domain logic, use cases, and infrastructure, the codebase remains maintainable and testable at scale. The RAG-based chat system and AI-powered quiz generation showcase modern AI integration patterns while maintaining clean code principles.
