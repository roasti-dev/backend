# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Development
make start                    # Run server with APP_ENV=development
SERVER_PORT=9090 DEBUG=1 go run ./cmd/server  # Run directly

# Build
make build                    # Build binary (./app)
make build-linux-amd64        # Cross-compile for Linux AMD64 (for deployment only)

# Code generation (run after editing api/spec.yaml or api/models.yaml)
make oapi                     # Regenerate server handlers, models, and test client

# Testing
make test-e2e                 # E2E tests (The Firebase emulator starts automatically)
go test -run TestName ./tests/e2e/...          # Single E2E test  (requires Firebase emulator running)
go test -run TestName ./internal/package/...   # Single unit test
make cover                    # Open HTML coverage report

# Firebase emulator (required for tests)
make firebase-emulator        # Start emulator
make wait-firebase            # Wait until emulator is ready

# Linting
make lint                     # Go linter (golangci-lint)
make openapi-lint             # OpenAPI spec linter (Redocly)

# Seeding
DATABASE_PATH=data.db go run ./cmd/seed --user=test --type recipes --file ./cmd/seed/data/recipes.json
```


## Architecture

**OpenAPI-first Go backend** with SQLite and Firebase Auth.

The API contract lives in `api/spec.yaml` and `api/models.yaml`. `make oapi` generates:
- `internal/handlers/server.gen.go` — type-safe HTTP handler interfaces
- `internal/api/models.gen.go` — Go types for all request/response schemas
- `tests/client/client.gen.go` — typed HTTP client used in E2E tests

**Request flow:**
```
HTTP request
  → middleware chain (CORS, logging, OpenAPI validation, auth)
  → generated handler dispatch (server.gen.go)
  → domain handler (internal/handlers/*.go)
  → service (internal/recipes/, users/, likes/, uploads/)
  → repository → SQLite via squirrel query builder
```

**App initialization** (`internal/app/app.go`): wires up the DB, Firebase client, services, middleware, and background goroutines (revoked token cleanup, temp file cleanup). Entry point is `cmd/server/main.go`.

**Auth** (`internal/auth/`, `internal/middleware/`): accepts Bearer token or access token cookie; validates via Firebase; maintains a local SQLite revoked-tokens table for logout. Set `FIREBASE_AUTH_EMULATOR_HOST` to point at the local emulator during development/tests.

**Domain packages** (`internal/recipes/`, `internal/likes/`, `internal/users/`, `internal/uploads/`): each has a service + repository pair. The repository layer uses squirrel for query building against SQLite.

**Database schemas** (`internal/db/schemas`): SQL DDL statements in the project are stored as elements of a slice in a Go file and executed at application startup. Each statement must be idempotent, meaning that running it multiple times should not change the database state or cause errors.

## Key Environment Variables

| Variable | Default | Notes |
|---|---|---|
| `SERVER_PORT` | `9090` | |
| `DATABASE_PATH` | `data.db` | SQLite file |
| `APP_ENV` | `development` | Set to `production` on server |
| `DEBUG` | — | Any value enables debug logging |
| `UPLOADS_PATH` | `./uploads` | Uploaded images directory |
| `FIREBASE_PROJECT_ID` | — | Required |
| `FIREBASE_API_KEY` | — | Required |
| `FIREBASE_CREDENTIALS_JSON_BASE64` | — | Base64-encoded service account JSON |
| `FIREBASE_AUTH_EMULATOR_HOST` | — | Set to `127.0.0.1:9099` for local emulator |

Copy `.env.example` to `.env` for local development.

## Adding or Changing API Endpoints

1. Edit `api/spec.yaml` (routes/operations) or `api/models.yaml` (shared schemas)
2. Run `make oapi` to regenerate code
3. Implement the new handler method in `internal/handlers/`

## Git Commits

Use Conventional Commits format: `<type>(<scope>): <description>`
For breaking changes: `<type>(<scope>)!: <description>`

**Types:** `feat`, `fix`, `refactor`, `perf`, `test`, `chore`

**Scopes:**
- `auth`, `users`, `recipes`, `likes`, `uploads` — domain changes
- `api` — api/spec.yaml, api/models.yaml

Always sign commits with `-s` flag (DCO sign-off).

