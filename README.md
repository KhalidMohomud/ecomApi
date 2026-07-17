# E-commerce API

A production-oriented E-commerce REST API built with Go, following Clean Architecture and the Repository Pattern. Built incrementally, step by step, as a learning project — see [Project Status](#project-status) for what's implemented so far.

## Tech Stack

| Concern | Choice |
|---|---|
| Language | Go 1.26 |
| HTTP framework | [Gin](https://github.com/gin-gonic/gin) |
| Database | PostgreSQL ([Neon](https://neon.tech) in development; any Postgres in production) |
| ORM | [GORM](https://gorm.io) |
| Migrations | [golang-migrate](https://github.com/golang-migrate/migrate) (plain SQL, not GORM AutoMigrate) |
| Auth | JWT access tokens (stateless) + opaque, revocable refresh tokens (stored hashed) |
| Password hashing | bcrypt |
| Config | Viper + godotenv |
| Validation | go-playground/validator (via Gin's binding) |
| Logging | `log/slog` (structured JSON) |
| API docs | Swagger/OpenAPI via [swaggo](https://github.com/swaggo/swag) |
| Testing | `testing` + [testify](https://github.com/stretchr/testify) |
| Containerization | Docker + Docker Compose |

## Architecture

Clean Architecture with one-way dependencies: **Handler → Service → Repository → Database**. No layer skips the one below it — a handler never touches `*gorm.DB` directly, and a repository never contains a business rule.

```
cmd/server/          Composition root — wires every dependency and starts the HTTP server. No business logic.
internal/
  config/            Loads and validates all configuration from environment variables.
  database/          Opens and configures the PostgreSQL connection pool.
  domain/
    entity/          Domain structs (User, Product, Category, ...) and shared sentinel errors.
    repository/      One interface + one GORM-backed implementation per entity.
  service/           All business logic. Depends on repository interfaces, never concrete types.
  handler/           HTTP boundary: binds requests, calls one service method, formats the response.
  middleware/        Auth (JWT), RequireRole (RBAC), request logging.
  dto/               Request/response shapes — the only structs ever serialized to/from JSON.
  utils/             Password hashing, JWT/refresh-token generation, slugs, the response envelope.
  validator/         Formats validation errors into the API's error response shape.
  routes/            The only place that maps URL paths to handler methods.
migrations/          Plain SQL migrations (golang-migrate), the single source of truth for schema.
docs/                Generated Swagger spec (swag init) — do not hand-edit.
```

Every dependency is constructed explicitly in `cmd/server/main.go` and passed down (constructor functions, not globals) — see that file for the full wiring order.

## Project Status

Built as a sequence of steps, each with real (not toy) code, tests, and live verification against a real database before moving on.

- [x] Project setup, configuration, Neon connection
- [x] Users: entity, migration, repository
- [x] Auth: register, login, logout, JWT access + rotating refresh tokens
- [x] JWT middleware, role-based authorization, user profile (get/update/change password/delete account)
- [x] Admin: dashboard, list/block/unblock/delete users
- [x] Categories & Brands: CRUD, hierarchy (categories), slugs
- [x] Products: CRUD, search, filter, sort, pagination
- [x] Docker & Docker Compose
- [ ] Inventory / stock management
- [ ] Cart, Wishlist
- [ ] Orders, mock Payments, Coupons
- [ ] Reviews & ratings
- [ ] Addresses
- [ ] Product image upload
- [ ] Rate limiting, CORS, request ID middleware
- [ ] Forgot/reset password, verify email

## Getting Started

### Prerequisites

- Go 1.26+
- A PostgreSQL database — either a free [Neon](https://neon.tech) project, or the local Postgres container in `docker-compose.yml` (no separate install needed for the latter)
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI: `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`
- Docker + Docker Compose (only if you want the containerized workflow)

### Installation

```bash
git clone <this-repo>
cd api
go mod download
```

### Environment Variables

Copy the template and fill in real values — **never commit `.env`** (it's gitignored; only `.env.example` is tracked):

```bash
cp .env.example .env
```

| Variable | Required | Description |
|---|---|---|
| `APP_ENV` | no (default `development`) | `development` or `production` |
| `APP_PORT` | no (default `8080`) | HTTP port |
| `DATABASE_URL` | **yes** | Full Postgres connection string. Neon requires `?sslmode=require`. |
| `DB_MAX_OPEN_CONNS` | no (default `25`) | Connection pool size |
| `DB_MAX_IDLE_CONNS` | no (default `5`) | Idle connections kept warm |
| `DB_CONN_MAX_LIFETIME_MINUTES` | no (default `5`) | Max age of a pooled connection |
| `JWT_SECRET` | **yes** | HMAC signing key for access tokens. Must be ≥32 characters — the app refuses to start otherwise. Generate with `openssl rand -base64 48`. |
| `JWT_ACCESS_TOKEN_TTL_MINUTES` | no (default `15`) | Access token lifetime |
| `JWT_REFRESH_TOKEN_TTL_DAYS` | no (default `30`) | Refresh token lifetime |

### Database Migration

Migrations are plain SQL files in `migrations/`, applied with the `migrate` CLI — GORM's AutoMigrate is never used against this schema (see the comment on `entity.User` for why: no rollback path, unsafe column changes, no support for the partial unique indexes this schema relies on).

```bash
# Apply all migrations
migrate -path migrations -database "$DATABASE_URL" up

# Roll back the last migration
migrate -path migrations -database "$DATABASE_URL" down 1

# Create a new migration
migrate create -ext sql -dir migrations -seq create_something_table
```

`DATABASE_URL` must use the `postgres://` scheme for the `migrate` CLI specifically (Neon's dashboard gives you `postgresql://`, which works fine for the app itself but needs the scheme swapped for this command):

```bash
DB_URL=$(grep '^DATABASE_URL=' .env | cut -d '=' -f2- | sed 's#^postgresql://#postgres://#')
migrate -path migrations -database "$DB_URL" up
```

Current migrations, in order: `create_users_table`, `create_refresh_tokens_table`, `create_categories_table`, `create_brands_table`, `create_products_table`.

### Running

```bash
go run ./cmd/server
```

The server starts on `http://localhost:8080`. Try it:

```bash
curl http://localhost:8080/health
```

### Docker

Two ways to use Docker here:

**1. Build and run just the app image** (pointing at any Postgres, e.g. Neon):

```bash
docker build -t ecomapi .
docker run -p 8080:8080 --env-file .env -e DATABASE_URL="$DATABASE_URL" ecomapi
```

**2. Full local stack** (app + a local Postgres container — no cloud dependency at all):

```bash
docker compose up -d db                     # start Postgres first
DB_URL="postgresql://ecomapi:ecomapi@localhost:5432/ecomapi?sslmode=disable"
migrate -path migrations -database "$DB_URL" up   # run migrations against it, from the host
docker compose up -d app                    # start the app
curl http://localhost:8080/health
```

`docker-compose.yml`'s `app` service loads `.env` for secrets like `JWT_SECRET`, then overrides `DATABASE_URL`/`APP_PORT`/`APP_ENV` to point at the compose-network `db` service instead of whatever `.env` has configured for local `go run` (typically Neon). Note the local Postgres connection uses `sslmode=disable` — Neon requires `sslmode=require` over the public internet, but traffic between two containers on the same Compose network never leaves the host, so there's nothing to encrypt.

Tear down with `docker compose down -v` (the `-v` also removes the named volume holding Postgres's data — omit it to keep your local data between runs).

### API Documentation

Every endpoint has Swagger annotations, generated into `docs/`. With the server running:

```
http://localhost:8080/swagger/index.html
```

Protected endpoints are marked with a lock icon — click **Authorize** in the UI and paste `Bearer <access_token>` to call them interactively.

Regenerate the spec after adding or changing an endpoint's annotations:

```bash
swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
```

(`go install github.com/swaggo/swag/cmd/swag@latest` if you don't have `swag` yet.)

### Testing

Two kinds of tests exist side by side, deliberately:

- **Repository tests** (`internal/domain/repository/*_test.go`) are integration tests against a real database — they exist to prove the actual SQL is correct (partial unique indexes, soft-delete filtering, transactional cleanup on delete), which a mock can't verify.
- **Service tests** (`internal/service/*_test.go`) and **middleware tests** (`internal/middleware/*_test.go`) are true unit tests against hand-written in-memory fakes — no database, no network, run in milliseconds.

```bash
# Everything (repository tests need DATABASE_URL set — via .env)
go test ./...

# Just the fast, no-database tests
go test ./internal/service/... ./internal/middleware/...

# Verbose, single package
go test ./internal/domain/repository/... -v
```

## Response Format

Every endpoint returns one of two JSON envelopes:

```json
// success
{ "success": true, "message": "Product created successfully", "data": { ... } }

// failure
{ "success": false, "message": "Validation failed", "errors": ["Email is required"] }
```

## Security Notes

- Passwords hashed with bcrypt (cost 12), never logged, never returned in any response.
- Access tokens are short-lived JWTs; refresh tokens are opaque random strings stored **hashed** (SHA-256) in the database, so a database leak alone cannot be used to authenticate. Refresh tokens rotate on every use.
- Money is stored as integer cents (`price_cents`), never a float, to avoid currency rounding errors.
- Every list/sort endpoint validates sort values against a fixed allow-list before they ever reach a SQL `ORDER BY` clause — user input never gets interpolated directly into one.
- `.env` is gitignored and excluded from the Docker build context (`.dockerignore`) — secrets are injected via environment variables, never baked into an image layer.
