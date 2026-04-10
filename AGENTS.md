# Outreacher — Agent Instructions

## Project Layout

```
outreacher/
├── app/              # Next.js 15 App Router (Bun runtime)
│   └── api/          # Route handlers + integration tests
├── mcp/              # Go MCP server + CLI binary
│   ├── cmd/mcp/      # Claude Desktop stdio binary
│   ├── cmd/server/   # SSE server (Docker Compose dev stack)
│   ├── internal/
│   │   ├── db/
│   │   │   ├── queries/   # sqlc SQL source files
│   │   │   ├── gen/       # sqlc-generated Go (do not edit)
│   │   │   └── sqlc.yaml
│   │   ├── tenant/   # Bootstrap() — idempotent system defaults
│   │   └── tools/    # One file per MCP tool (or logical group)
│   └── migrations/   # goose migrations (5-digit prefix: 00001_…)
├── postgres/init/    # DB init scripts (create app user, grants)
├── imports/          # Sample CSV files for testing
├── docs/             # Architecture docs
├── bin/              # Built binaries (gitignored)
└── docker-compose.yml
```

## Architecture

**Multi-tenancy:** Organization → Brand → User
- All domain objects (companies, leads, contact_identifiers) are scoped by `brand_id`
- `contact_identifiers` PK: `(brand_id, type, value)`
- `companies` unique: `(brand_id, name)`

**Two-user DB pattern:**
- `admin` — DDL owner, runs migrations
- `app` — DML only, used by the running server

**MCP transports:**
- `cmd/mcp` — stdio (Claude Desktop)
- `cmd/server` — SSE on `:3001` (Docker Compose dev)

**Tenant bootstrap:** Both binaries call `tenant.Bootstrap(ctx, q)` at startup. It idempotently creates the system org, Default brand, and system user if they don't exist (reads `MCP_ORG_SLUG`, `MCP_USER_SLUG` env vars, defaults: `system_default_org`, `system_default_user`).

**Brand switching:** Explicit `brand_id` param on all domain MCP tools (not stateful session). Defaults to startup brand.

**Next.js → MCP:** `app/lib/mcp.ts` — `callTool(name, args)` over SSE transport (`MCP_URL` env var).

## Running Things

### Docker Compose (dev stack)
```bash
docker compose up --build -d          # start postgres + mcp + app
docker compose logs -f mcp            # tail MCP server logs
docker compose down -v                # stop and remove volumes
```

**Host requirements: Docker only.** All build, test, migration, and codegen commands run inside Docker containers. The one exception is `bun run build:mcp`, which requires Bun on the host (it shells out to `docker run` internally). Do not run `go`, `goose`, or `sqlc` directly on the host.

### Next.js tests
```bash
docker compose exec app bun test
```

### Go tests
```bash
# requires docker compose stack to be running
docker compose exec mcp go test ./...
```

### Standalone postgres (for Claude Desktop binary)
```bash
# volume name: outreacher_standalone_data  (intentionally different from compose)
docker run -d --name outreacher-pg \
  -e POSTGRES_DB=outreacher \
  -e POSTGRES_USER=admin \
  -e POSTGRES_PASSWORD=admin \
  -v outreacher_standalone_data:/var/lib/postgresql/data \
  -v $(pwd)/postgres/init:/docker-entrypoint-initdb.d \
  -p 5432:5432 \
  postgres:16-alpine
```

### Migrations (goose)

First get the postgres container IP (works for both compose and standalone containers):
```bash
# for the standalone container:
PG_IP=$(docker inspect outreacher-pg --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}')

# for the compose stack:
PG_IP=$(docker inspect outreacher-outreacher-postgres-1 --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}')
```

Then run migrations:
```bash
docker run --rm \
  -v $(pwd)/mcp:/src -w /src \
  --add-host=postgres:$PG_IP \
  golang:1.25-alpine \
  sh -c 'go run github.com/pressly/goose/v3/cmd/goose@latest \
    -dir migrations postgres \
    "postgresql://admin:admin@postgres:5432/outreacher" up'
```

Migration files: `mcp/migrations/` — 5-digit prefix (`00001_initial_schema.sql`, …). Always use `-- +goose Up` / `-- +goose Down` markers.

### Build MCP binary
```bash
# builds linux-amd64, windows-amd64, darwin-amd64/arm64 binaries into bin/
bun run build:mcp
```

### sqlc codegen
```bash
docker run --rm \
  -v $(pwd)/mcp/internal/db:/src -w /src \
  sqlc/sqlc:latest generate
```
Run this after any change to `queries/*.sql` or `migrations/*.sql`. The `gen/` directory is fully generated — never edit it manually.

## Key Conventions

- **Queries first:** Add/modify SQL in `mcp/internal/db/queries/*.sql`, run `sqlc generate`, then use the generated types in tools.
- **One tool per file** (or one logical group like `orgs.go`). Each file exports a single `register*` function called from `tools/tools.go`.
- **Tool brand_id pattern:** All domain tools accept an optional `brand_id` number param that defaults to `defaultBrandID` passed at registration.
- **Error handling:** Return `mcp.NewToolResultError(msg)` for user/input errors; return `nil, err` for infrastructure errors.
- **No mocks in Go tests** — integration tests use a real DB (`DATABASE_URL` + `DATABASE_ADMIN_URL`).
- **Import route dual content-type:** `POST /api/leads/import` accepts both `multipart/form-data` (file upload) and `application/json` `{ csv, brand_id? }`.
- **Test CSV path:** `fixtures/imports/gojiberry/gojiberry-selected-contacts-2026-04-04.csv` — used in integration test step 5.

## Environment Variables

| Var | Used by | Default |
|-----|---------|---------|
| `DATABASE_URL` | Go server, Next.js | — |
| `DATABASE_ADMIN_URL` | Go server (migrations) | — |
| `MCP_URL` | Next.js (`app/lib/mcp.ts`) | — |
| `MCP_ORG_SLUG` | Go binary (bootstrap) | `system_default_org` |
| `MCP_USER_SLUG` | Go binary (bootstrap) | `system_default_user` |
| `MCP_URL_PORT` | docker-compose port map | `3001` |
| `LLM_API_KEY` | Go MCP server (Claude extraction) | — (falls back to CSV parsers if unset) |
| `LLM_MODEL` | Go MCP server (Claude extraction) | `claude-haiku-4-5` |
