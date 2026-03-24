# neknow-bot

A personal project for learning how RAG (Retrieval-Augmented Generation) works in practice, built with Go and Python.

---

## Project Overview

neknow-bot is a chatbot platform where you can create custom AI bots, upload documents as their knowledge base, and chat with them. The bot answers questions based on the content of the uploaded documents rather than just relying on the LLM's general knowledge.

It's not production-ready software. It's a learning project — built to understand how RAG systems actually work under the hood, not just theoretically.

---

## Motivation

I kept reading about RAG in blog posts and papers but never fully understood how all the pieces fit together in a real backend. Things like: how do you actually store embeddings? How does retrieval work at query time? How do you connect a Go API to a Python ML pipeline without it feeling like duct tape?

So I built this to find out. The goal was never to ship a product — it was to learn by doing and understand each layer deeply enough to explain it to someone else.

---

## What I Wanted to Learn

- How RAG pipelines work end-to-end, from document ingestion to answer generation
- Document parsing, chunking strategies, and why chunk size and overlap matter
- How embeddings are generated and stored as vectors
- Vector similarity search and hybrid search (dense + sparse)
- How re-ranking improves retrieval accuracy before passing context to the LLM
- How to integrate a Python ML worker with a Go backend cleanly
- How to structure a Go backend using Clean Architecture without over-engineering it
- Async processing with message queues (why you don't want to block the API on a 30-second embedding job)

---

## Architecture

The system has two main parts: a Go API that handles everything user-facing, and a Python worker that handles the heavy ML work asynchronously.

```
┌──────────────────────────────────────────────┐
│              Client (HTTP / SSE)             │
└───────────────────┬──────────────────────────┘
                    │
┌───────────────────▼──────────────────────────┐
│              Go API Server                   │
│        Auth │ Bot │ Files │ Chat             │
│       Handler → Service → Repository         │
└──────┬─────────────────────┬─────────────────┘
       │                     │
┌──────▼──────┐   ┌──────────▼───────────────┐
│  PostgreSQL │   │        RabbitMQ          │
│ + pgvector  │   └──────────┬───────────────┘
└──────┬──────┘              │
       │             ┌───────▼──────────────────┐
       │             │     Python Worker        │
       │             │                          │
       │             │  unstructured → chunk    │
       │             │  → embed → pgvector      │
       └─────────────┘                          │
                     └──────────────────────────┘
```

The Go API is intentionally kept thin, it handles HTTP, auth, validation, and database CRUD. The Python worker handles everything ML-related: parsing documents, generating embeddings, and storing vectors. They communicate through RabbitMQ for indexing (async) and HTTP for retrieval (sync, per query).

The reason for separating them into different processes rather than calling OpenAI directly from Go is that document indexing can take 10–30 seconds for large files. You don't want to block an HTTP request for that long. So the API publishes a job to the queue and returns immediately, while the worker picks it up in the background.

---

## Tech Stack

### Go API
| | |
|---|---|
| **Gin** | HTTP framework — straightforward, minimal boilerplate |
| **GORM** | ORM — good enough for this scale, avoids raw SQL for common queries |
| **PostgreSQL** | Main database for all relational data |
| **pgvector** | PostgreSQL extension for vector storage — chose this over a dedicated vector DB to avoid running yet another service |
| **Redis** | Reserved for caching and rate limiting (not fully implemented yet) |
| **RabbitMQ** | Message queue for async document indexing |
| **MinIO** | S3-compatible object storage for uploaded files, self-hosted |


### Python Worker
| | |
|---|---|
| **uv** | Package manager — significantly faster than pip |
| **aio-pika** | Async RabbitMQ client |
| **unstructured** | Document parsing — handles PDF, DOCX, HTML without writing format-specific parsers |
| **langchain-text-splitters** | Recursive chunking — respects paragraph and sentence boundaries |
| **tiktoken** | Token counting to make sure chunks don't exceed model limits |
| **text-embedding-3-small** | OpenAI embedding model — cheap, fast, good enough |
| **pgvector (Python)** | Adapter for inserting vectors into PostgreSQL |
| **Cohere Rerank** | Re-ranking API — improved retrieval accuracy without running a local model |
| **gpt-4o-mini** | LLM for response generation — faster and cheaper than gpt-4o for RAG use cases |

The main tradeoff I made was using pgvector instead of a dedicated vector database like Qdrant. pgvector is slower at very large scale, but it keeps the infrastructure simple and means I only need one database to manage. For learning purposes and small-to-medium scale, it's fine.

---

## Project Structure

```
neknow-bot/
├── cmd/
│   └── api/
│       └── main.go           # entrypoint, wires everything together
├── internal/
│   ├── auth/                 # registration, login, token refresh
│   ├── bot/                  # bot CRUD, model config
│   ├── files/                # file upload, indexing status
│   └── chat/                 # conversations, messages, RAG retrieval
│       ├── domain/           # interfaces + core types (no dependencies)
│       ├── dto/              # request/response structs for HTTP layer
│       ├── handler/          # gin handlers, HTTP only
│       ├── repository/       # database queries via GORM
│       └── service/          # business logic, orchestrates repo + external calls
├── pkg/
│   ├── config/               # env loading
│   ├── database/             # postgres connection + migration
│   │   └── models/           # GORM model structs
│   ├── middleware/            # JWT auth middleware
│   └── response/             # standard JSON response helpers
├── worker/                   # Python ML worker (separate process)
│   ├── src/
│   │   ├── consumer/         # RabbitMQ consumer
│   │   ├── processor/        # parse, chunk, embed logic
│   │   └── storage/          # MinIO + pgvector I/O
│   └── pyproject.toml
└── scripts/
    └── test_api.sh           # end-to-end API tests via curl
```

Each `internal/` module follows the same pattern: `domain` defines interfaces, `repository` implements DB access, `service` holds business logic, `handler` deals with HTTP. The domain layer has no external dependencies — it only knows about its own types. This makes it easy to swap out the repository implementation or test the service in isolation.

---

## How It Works

### Indexing (when a file is uploaded)

1. User uploads a PDF or DOCX via `POST /bots/:id/files`
2. Go saves the file to MinIO and writes a record to PostgreSQL with status `pending`
3. Go publishes a message to RabbitMQ: `{ file_id, bot_id, object_key }`
4. Python worker consumes the message, downloads the file from MinIO
5. `unstructured` parses the file and extracts clean text
6. `RecursiveCharacterTextSplitter` splits the text into 512-token chunks with 50-token overlap
7. `tiktoken` validates each chunk fits within model limits
8. OpenAI `text-embedding-3-small` generates a 1536-dimension vector for each chunk
9. Vectors + chunk text are inserted into PostgreSQL via pgvector
10. File status is updated to `indexed`

### Retrieval (when a user sends a message)

1. User sends a message via `POST /conversations/:id/messages`
2. Go passes the query to the Python RAG service via HTTP
3. Python embeds the query using the same embedding model
4. Hybrid search runs against pgvector: cosine similarity (HNSW) + BM25 keyword search, fused via RRF → top 20 chunks
5. Cohere Rerank re-scores the top 20 → keeps top 5 most relevant chunks
6. A prompt is assembled: bot system prompt + retrieved context + last 5 messages of chat history + user query
7. OpenAI `gpt-4o-mini` generates a response, streamed back via SSE
8. Go forwards the stream to the client and saves the full response to the database

---

## Running the Project

### Prerequisites

- Go 1.22+
- Python 3.11+ with [uv](https://github.com/astral-sh/uv)
- Docker (for PostgreSQL, Redis, RabbitMQ, MinIO)

### Setup

```bash
git clone https://github.com/Zeropeepo/neknow-bot.git
cd neknow-bot

# Copy and fill in env vars
cp .env.example .env

# Start infrastructure
docker-compose up -d

# Run Go API
go mod tidy
go run cmd/api/main.go

# Run Python worker (separate terminal)
cd worker
uv sync
uv run python -m src.main
```

### Running Tests

```bash
export DB_PASSWORD=your_password
export DB_USER=neknowbot
export DB_NAME=neknow_bot_db

chmod +x scripts/test_api.sh
./scripts/test_api.sh
```

---

## Current Limitations

- **No streaming yet** — SSE for chat responses is planned but not implemented
- **No rate limiting** — Redis is wired up but rate limiting middleware isn't in place
- **Single-tenant retrieval** — vector search queries all chunks for a bot, not per-user per-conversation
- **No file deduplication** — uploading the same file twice will index it twice
- **Worker has no retry logic** — if embedding fails midway, the file stays in `indexing` state
- **No evaluation** — there's no way to measure retrieval quality or answer accuracy yet
- **pgvector at scale** — HNSW index works well up to a few million vectors, but this hasn't been stress tested

---

## Future Improvements (maybe)

- Add streaming responses via SSE
- Implement proper retry and dead-letter queue handling in the worker
- Add a simple eval loop to measure retrieval precision (e.g. using RAGAS)
- Experiment with smaller, faster embedding models for lower latency
- Try Qdrant as a drop-in replacement for pgvector to compare performance
- Add metadata filtering to vector search (e.g. only search chunks from specific files)
- Build a simple frontend to make this actually usable as a demo
- Add support for more document types (Markdown, plain text, web URLs)

---

## License

