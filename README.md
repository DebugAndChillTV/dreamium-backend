# Dreamium Backend

A Go REST API that analyzes dreams using OpenAI GPT-4o. It validates dream input, detects language, and returns psychological, symbolic, and psychologist-specific interpretations. Built with Fiber, backed by Supabase for auth and dream storage.

## Tech Stack

- **Go** + [Fiber v2](https://github.com/gofiber/fiber) — HTTP framework
- **OpenAI GPT-4o / GPT-4o-mini** — dream analysis
- **Supabase** — authentication (JWT) and database (dreams table)
- **go-cache** — in-memory rate limiting

## Project Structure

```
dreamium-backend/
├── main.go                  # Entry point, server setup
├── routes/
│   ├── routes.go            # Route registration
│   └── openai.go            # OpenAI handlers + Supabase dream storage
├── middleware/
│   ├── auth.go              # Supabase JWT auth middleware
│   └── rate-limit.go        # IP-based rate limiter (10 req/min)
├── go.mod
└── go.sum
```

## API Endpoints

All routes are prefixed with `/api` and require a Supabase JWT in the `Authorization: Bearer <token>` header.

| Method | Endpoint                            | Description                                      |
|--------|-------------------------------------|--------------------------------------------------|
| POST   | `/api/isDreamInput`                 | Validates input as a dream, detects language, extracts keywords & mood, saves to DB |
| POST   | `/api/generateDreamAnalysis`        | Psychological analysis (Freud, Jung, Adler, etc.) with pattern context from past dreams |
| POST   | `/api/generateSymbolicInterpretation` | Symbolic/folkloric interpretation of the dream |
| POST   | `/api/generatePsychologistInterpretation` | Analysis from a specific psychologist's perspective |

### Request Bodies

**`/api/isDreamInput`**
```json
{
  "userInput": "I was flying over a dark ocean...",
  "userId": "uuid"
}
```

**`/api/generateDreamAnalysis`**
```json
{
  "userDream": "I was flying over a dark ocean...",
  "detectedLanguage": "English",
  "userId": "uuid"
}
```

**`/api/generateSymbolicInterpretation`**
```json
{
  "userDream": "I was flying over a dark ocean...",
  "detectedLanguage": "English",
  "userId": "uuid"
}
```

**`/api/generatePsychologistInterpretation`**
```json
{
  "userDream": "I was flying over a dark ocean...",
  "detectedLanguage": "English",
  "psychologist": "Carl Jung",
  "userId": "uuid"
}
```

## Getting Started

### Prerequisites

- Go 1.23+
- A [Supabase](https://supabase.com) project with a `dreams` table
- An [OpenAI](https://platform.openai.com) API key

### Environment Variables

Create a `.env` file in the project root:

```env
OPENAI_API_KEY=your_openai_api_key
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ADMIN_KEY=your_supabase_service_role_key
PORT=8080
```

### Run Locally

```bash
go mod download
go run main.go
```

The server starts on `http://localhost:8080` (or the `PORT` you set).

### Build

```bash
go build -o dreamium-server .
./dreamium-server
```

## Supabase `dreams` Table Schema

```sql
create table dreams (
  id          uuid primary key,
  created_at  timestamptz default now(),
  user_id     uuid references auth.users(id),
  dream       text,
  dream_tags  text[],
  mood        text,
  language    text
);
```

## Rate Limiting

Requests are limited to **10 per minute per IP**. Exceeding this returns `429 Too Many Requests`.
