# MCP Migration Plan: TypeScript → Go

Migration of the MCP server and supporting infrastructure from TypeScript/Bun to Go,
culminating in a distributable Go binary used by both Claude Desktop and the Next.js SaaS frontend.

---

## Phase 1 — Go MCP server + versioned binary build ✅

- `go/` module: `cmd/mcp` (stdio) and `cmd/server` (SSE HTTP)
- `internal/tools`: 4 MCP tools sharing the same implementation
- `internal/db`: pgx/v5 pool + sqlc-generated queries
- `migrations/001_initial_schema.sql` (goose format)
- `Dockerfile.mcp`: multi-stage distroless build (~9MB image)
- `build:mcp` script extracts versioned binaries to `bin/` using package.json name+version
- `cmd/server`: bearer token auth via `MCP_API_KEY`, base URL via `MCP_URL`
- `bin/` gitignored; Claude Desktop points at `bin/outreacher-mcp-<version>` via wsl.exe

## Phase 2 — Replace TS compose service with Go dev container ✅

- Replace `mcp` compose service (bun/TS) with Go server container
- `Dockerfile.mcp` dev stage: golang:1.24-alpine + air v1.61.1 for hot reload
- `go/.air.toml`: watches `.go` files, rebuilds `cmd/server` on change
- `./go` mounted as `/src` in the container
- `MCP_URL` env var controls SSE base URL (fixes origin mismatch between containers)
- Next.js API route (`app/api/leads/route.ts`) calls MCP server via SSEClientTransport

## Phase 3 — Remove TypeScript MCP + Drizzle, rename go/ → mcp/

- Delete `mcp/` folder (server.ts, create-server.ts, http-server.ts)
- Delete `db/` folder (index.ts, schema.ts, seed.ts) and `drizzle.config.ts`
- Remove from package.json: `drizzle-orm`, `drizzle-kit`, `postgres` deps
- Remove scripts: `mcp`, `db:push`, `db:seed`, `db:studio`
- Rename `go/` → `mcp/`
- Update `Dockerfile.mcp`: `COPY go/` → `COPY mcp/`
- Update `docker-compose.yml`: `./go:/src` → `./mcp:/src`
- Update `.gitignore`: `go/tmp/` → `mcp/tmp/`

## Phase 4 — ConnectRPC server

Add a ConnectRPC API to the Go binary so Next.js can call it directly over HTTP/2
without MCP protocol overhead.

- `proto/outreacher/v1/outreacher.proto` — service definition
- `buf.yaml` + `buf.gen.yaml` — code gen config (runs inside Docker build)
- `mcp/internal/service/` — business logic extracted from `internal/tools`
- `mcp/internal/connect/` — ConnectRPC handler (thin adapter over service layer)
- `cmd/server`: mount both MCP SSE and ConnectRPC on the same `http.ServeMux`
- Next.js: `@connectrpc/connect-web` client replaces raw MCP tool calls

## Phase 5 — Next.js SaaS frontend

Replace direct MCP tool calls in the Next.js app with typed ConnectRPC calls.

- Generate TypeScript client from proto
- Build out UI pages backed by ConnectRPC endpoints
- Auth, multi-tenancy, and deployment considerations
