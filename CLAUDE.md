# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run locally
go run main.go

# Build binary
go build -o dreamium-server .

# Format (required before committing)
go fmt ./...

# Vet
go vet ./...

# Lint (if golangci-lint installed)
golangci-lint run

# Download dependencies
go mod download
```

No test suite exists yet. The file `test_sentry.go` uses `//go:build ignore` and is not part of the build.

## Architecture

Go + Fiber v2 REST API. Four packages:

- **`config/`** — Infisical client (`config.Init()`) and `config.GetSecret(name)`. Falls back to `os.Getenv` when Infisical is not configured (local dev).
- **`crypto/`** — `Encrypt(plaintext, masterKey []byte) (string, error)` and `Decrypt(ciphertext string, masterKey []byte) ([]byte, error)`. AES-256-GCM; output is `base64(nonce[12] || ciphertext || tag[16])`. Pure functions, no state.
- **`main.go`** — Wires everything: loads `.env`, calls `middleware.InitSupabase()`, mounts rate limiter, registers routes, starts on `$PORT` (default 8080).
- **`middleware/`** — Two pieces: Supabase JWT auth (`auth.go`) and in-memory rate limiter at 10 req/min per IP (`rate_limiter.go`). `InitSupabase()` also owns the global Supabase client and the `SaveDream` / `GetDreams` DB helpers.
- **`routes/`** — All four endpoints live here. `SetupRoutes` creates a protected `/api` group (auth middleware applied), then registers the handlers. Each handler calls OpenAI and may call Supabase helpers.

### Auth flow
Every `/api/*` request must carry `Authorization: Bearer <supabase-jwt>`. The middleware calls `supabase.Auth.WithToken(token).GetUser()` and stores `user.ID` in `c.Locals("userID")`.

### OpenAI usage
Two models are used:
- `gpt-4o-mini` (`modelFast`) — dream validation, language detection, keyword/mood extraction
- `gpt-4o` (`modelSmart`) — all three interpretation endpoints

Before generating analysis, `GetDreams` fetches the user's 5 most recent prior dreams to inject pattern context into the prompt.

### Supabase schema
```sql
CREATE TABLE dreams (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at    TIMESTAMPTZ DEFAULT NOW(),
  user_id       UUID REFERENCES auth.users(id),
  dream         TEXT,
  dream_tags    TEXT[],
  mood          TEXT,
  language      TEXT DEFAULT 'en',
  response      TEXT,
  response_tags TEXT[]
);

CREATE TABLE users (
  id         UUID UNIQUE REFERENCES auth.users(id),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  email      TEXT,
  language   TEXT,
  token      INTEGER DEFAULT 500,
  paid       BOOLEAN DEFAULT false,
  paid_date  TIMESTAMP DEFAULT NOW(),
  bonus      BOOLEAN DEFAULT false
);
```

### RLS policies

Both tables have RLS enabled. Backend uses `SUPABASE_ADMIN_KEY` (service role) which bypasses RLS — policies only apply to direct Supabase client access (e.g. iOS app).

**`dreams`** — authenticated users may only touch their own rows:
| Operation | Allowed | Rule |
|---|---|---|
| SELECT | own rows | `user_id = auth.uid()` |
| INSERT | own rows | `WITH CHECK: user_id = auth.uid()` |
| DELETE | own rows | `user_id = auth.uid()` |
| UPDATE | ❌ forbidden | no policy |

**`users`** — authenticated users may only touch their own profile:
| Operation | Allowed | Rule |
|---|---|---|
| SELECT | own row | `auth.uid() = id` |
| INSERT | own row | `WITH CHECK: auth.uid() = id` |
| UPDATE | own row | `auth.uid() = id` |
| DELETE | ❌ forbidden | no policy |

## Environment Variables

| Variable | Required | Notes |
|---|---|---|
| `OPENAI_API_KEY` | Yes | Missing → panic |
| `SUPABASE_URL` | Yes | Missing → fatal exit |
| `SUPABASE_ADMIN_KEY` | Yes | Service role key; missing → fatal exit |
| `PORT` | No | Defaults to `8080` |
| `SENTRY_DSN` | No | If set, suspicious prompt-injection attempts are reported to Sentry (dream content is never logged) |
| `DREAM_MASTER_KEY` | Yes | 64-char lowercase hex string (32 bytes). Used as AES-256-GCM key. Set in Infisical or `.env` for local dev |
| `INFISICAL_CLIENT_ID` | No* | Universal Auth client ID. If absent, secrets fall back to `os.Getenv` |
| `INFISICAL_CLIENT_SECRET` | No* | Universal Auth client secret |
| `INFISICAL_PROJECT_ID` | No* | Infisical project ID |
| `INFISICAL_ENVIRONMENT` | No* | e.g. `production`, `development` |
| `INFISICAL_HOST` | No | Defaults to `https://app.infisical.com` |

*Required in production when using Infisical. For local dev, set secrets directly in `.env`.

Copy `.env.local` → `.env` and fill in values. The `.gitignore` excludes `.env*`.

## API Endpoints

All require Supabase Bearer token. All accept/return JSON.

| Method | Path | Purpose |
|---|---|---|
| POST | `/api/isDreamInput` | Validate input, detect language, extract tags/mood, save to DB |
| POST | `/api/generateDreamAnalysis` | Psychological analysis (multi-school: Freud, Jung, Adler, etc.) |
| POST | `/api/generateSymbolicInterpretation` | Symbolic/folkloric interpretation |
| POST | `/api/generatePsychologistInterpretation` | Single-psychologist perspective (pass `psychologist` field) |


## Security Rules

Never log user dreams to sentry
All supabase tables should have RLS protection

## Deployment

Deployed on Railway. Railway injects `PORT` automatically. No Docker setup — Railway builds from source using the Go buildpack.

## PR Rules

Never push the changes to main branch
Write proper description
