# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GoWind Admin (风行) is an enterprise-grade full-stack admin system. Go backend (Kratos microservice framework) + dual frontend (Vue primary, React alternative). Protocol-driven development: APIs defined in protobuf, code generated for Go, TypeScript, and OpenAPI.

## Repository Structure

```
backend/                    # Go backend
  app/admin/service/        # Admin service (single service, microservice-ready)
    cmd/server/             # Entry point (main.go)
    internal/data/          # Data layer (ENT ORM schemas, repositories)
    internal/server/        # Server setup (HTTP, Asynq task queue, SSE)
    internal/service/       # Business logic layer
    configs/                # Runtime config files
  api/                      # Protobuf definitions + generated code
    protos/                 # .proto source files
    gen/                    # Generated Go/TS code
  pkg/                      # Shared packages (auth, crypto, middleware, oss, etc.)
  sql/                      # Database seed data (PostgreSQL + MySQL)

frontend/admin/vue/         # Vue frontend (pnpm monorepo, Turbo orchestration)
  apps/admin/               # Main admin app (Ant Design Vue + Vben Admin)
  packages/                 # Shared packages (@core, stores, locales, utils, etc.)

frontend/admin/react/       # React frontend (Ant Design Pro, UMI/Max)
```

## Build & Development Commands

### Backend (run from `backend/`)

```bash
make init          # Install protoc plugins and CLI tools (first-time setup)
make gen           # Generate all code: ent + wire + api + openapi
make ent           # Generate ENT ORM code from schemas
make wire          # Generate Wire dependency injection code
make api           # Generate Go code from protobuf via buf
make openapi       # Generate OpenAPI v3 docs from protobuf
make ts            # Generate TypeScript clients from protobuf
make build         # Build binaries (runs api + openapi first)
make build_only    # Build binaries without code generation
make test          # Run Go tests: go test ./...
make cover         # Run tests with coverage report
make lint          # Run golangci-lint
make docker-libs   # Start only dependencies (PostgreSQL, Redis, MinIO)
make compose-up    # Start all services including backend via Docker Compose
```

Run the service directly (from `backend/app/admin/service/`):
```bash
make run           # go run ./cmd/server -c ./configs
```

### Vue Frontend (run from `frontend/admin/vue/`)

```bash
pnpm install       # Install dependencies
pnpm dev           # Start dev server (port 5666)
pnpm build         # Production build (Turbo-orchestrated)
pnpm run lint      # Lint code
pnpm run check:type  # TypeScript type checking
pnpm run test:unit   # Unit tests (Vitest)
pnpm run test:e2e    # E2E tests (Playwright)
```

### React Frontend (run from `frontend/admin/react/`)

```bash
pnpm install       # Install dependencies
pnpm dev           # Start dev server (port 5666)
pnpm build         # Production build
pnpm run lint      # Lint (Biome + tsc)
pnpm run test      # Tests (Jest)
```

## Key Architecture Patterns

**Backend layered architecture (Kratos):** proto definitions -> generated HTTP handlers -> service layer (business logic) -> data layer (ENT ORM repositories). Dependency injection via Google Wire (`wire.go` -> generated `wire_gen.go`).

**Code generation pipeline:** Protobuf is the source of truth for APIs. `buf` generates Go server stubs, TypeScript HTTP clients (separate templates for Vue and React), and OpenAPI docs. ENT generates ORM code from schemas in `internal/data/ent/schema/`.

**Multiple transports:** The backend serves HTTP (port 7788), Asynq (distributed task queue via Redis), and SSE (Server-Sent Events, port 7789).

**Auth:** JWT authentication (kratos-authn) + Casbin/OPA authorization (kratos-authz).

**Database:** PostgreSQL (primary) or MySQL. ENT ORM with features: privacy, entql, sql/modifier, sql/upsert, sql/lock.

**Vue monorepo:** Turbo for build orchestration. Shared packages under `packages/` (stores, locales, types, utils). Core UI kit in `packages/@core/`.

## Local Development URLs

- Frontend dev: `http://localhost:5666`
- Backend API: `http://localhost:7788`
- SSE endpoint: `http://localhost:7789/events`
- OpenAPI docs: `http://localhost:7788/docs/openapi.yaml`
- Default credentials: `admin` / `admin`

## Dependencies (Docker Compose)

PostgreSQL, Redis, MinIO (object storage). Start with `make docker-libs` from `backend/`.
