# Outreacher

> [!TIP]
> This project demonstrates and act as a foundation boilerplate for 2-mode operation of an MCP as:
> 1) Single-tenant Claude Desktop Go binary (individual use)
> 2) Multi-tenant Next.js SaaS + chat LLM + MCP Go server (hosted environment)
>
> The project architecture includes:
> * AI SDK + assistant-ui chat
> * Example MCP development
> * Shared backend code for both binary and server operation
> * Claude Code (and other development agent) usage and best practices
> * Docker best practices for development and production
> * Database security role/schema best practices (adopted early for scale)

A lead management system with an MCP server for Claude Desktop and a Next.js SaaS frontend.

> **All dev tooling runs via Docker.** You do not need Go, Node, Bun, goose, or sqlc installed locally.
>
> **Developers:** see `CLAUDE.md` for commands, conventions, and test instructions.

---

## Architecture

```
Claude Desktop
  └── bin/outreacher-mcp-0.1.0-linux-amd64  (Go stdio binary, WSL)
        └── localhost:5432  (standalone postgres, volume: outreacher_standalone_data)

Browser / Next.js frontend
  └── app (Next.js, :3000)
        └── mcp (:3001, Go SSE server, air hot reload)
              └── postgres (internal only, volume: outreacher_postgres-data)
```

Two runtime modes — standalone binary for Claude Desktop, Docker Compose for development.
**Important:** use separate named volumes for each mode to avoid data directory conflicts.

---

## Tech Stack

- **Next.js 15** (App Router, Bun)
- **Go MCP server** (`mcp/`) — stdio + SSE transports, pgx/v5, sqlc, goose migrations
- **PostgreSQL 16** — `app` schema, named Docker volume
- **Docker Compose** — dev stack with air hot reload on the Go server

---

## DB Users

| User | Password | Role |
|---|---|---|
| `admin` | `admin` | Owns schema, runs DDL and migrations |
| `app` | `app` | Next.js runtime — member of `app_crud` + direct nextauth grants |
| `mcp` | `mcp` | MCP server runtime — member of `app_crud` |
| `reporter` | `reporter` | Read-only access — member of `app_read` |

Group roles (no LOGIN) — DEFAULT PRIVILEGES auto-cover new tables/sequences/types:
- `app_crud` — SELECT/INSERT/UPDATE/DELETE on app schema
- `app_read` — SELECT on app schema

`DATABASE_URL` → `mcp` user (MCP server) or `app` user (Next.js). `DATABASE_ADMIN_URL` → `admin` user (goose only).

---

## Schema (`app`)

### Multi-tenancy

| Table | Key fields |
|---|---|
| `organizations` | id, name, slug (unique), is_system |
| `campaigns` | id, organization_id, name, slug, is_default |
| `users` | id, email†, slug (unique), name, is_system |
| `organization_memberships` | organization_id, user_id, role (owner\|admin\|member) |
| `campaign_memberships` | campaign_id, user_id, role (admin\|member\|viewer) |

### Domain (scoped to `campaign_id`)

| Table | Key fields |
|---|---|
| `companies` | id, campaign_id, name, domain, industry, linkedin_url |
| `signals` | id, description — global, no campaign scope |
| `signal_keywords` | signal_id, keyword — global |
| `company_signals` | company_id, signal_id |
| `leads` | id, campaign_id, name, email†, linkedin_url†, company_id, title, status, score, location, phone |
| `notes` | id, lead_id, content, created_at |
| `contact_identifiers` | campaign_id, type, value — dedup key, PK is (campaign_id, type, value) |

† nullable. Leads are deduplicated by `contact_identifiers`, not by email.

Lead status flow: `new → contacted → qualified → disqualified → converted`

---

## MCP Tools

### Domain tools (all accept optional `campaign_id`; defaults to startup campaign)

| Tool | Description |
|---|---|
| `search_leads` | Filter by name, email, status, company, campaign_id |
| `get_lead` | Full lead detail with company and notes |
| `update_lead_status` | Advance a lead through the status flow |
| `create_followup_note` | Append a note to a lead |
| `import_csv` | Import from CSV text (auto-detects Gojiberry, Revli formats) |
| `search_companies` | Search companies by name or domain |
| `get_company` | Full company detail with signals |

### Campaign management (stdio binary only)

| Tool | Description |
|---|---|
| `list_campaigns` | List all campaigns for the current org |
| `get_campaign` | Get campaign details by id |
| `create_campaign` | Create a new campaign under the current org |
| `rename_campaign` | Rename a campaign's display name |

---

## Mode 1 — Claude Desktop (Go binary via WSL)

**Build the binary:**

```bash
npm run build:mcp
# outputs bin/outreacher-mcp-0.1.0-linux-amd64   (stdio, WSL)
#         bin/outreacher-mcp-0.1.0-darwin-arm64   (stdio, macOS Apple Silicon)
#         bin/outreacher-mcp-0.1.0-darwin-amd64   (stdio, macOS Intel)
#         bin/outreacher-mcp-0.1.0-windows-amd64.exe
#         bin/outreacher-server-0.1.0             (SSE server)
```

**Start standalone postgres:**

```bash
docker run -d --name outreacher-pg --restart always \
  -p 5432:5432 \
  -e POSTGRES_DB=outreacher \
  -e POSTGRES_USER=admin \
  -e POSTGRES_PASSWORD=admin \
  -v outreacher_standalone_data:/var/lib/postgresql/data \
  -v "$(pwd)/postgres/init:/docker-entrypoint-initdb.d" \
  postgres:16-alpine
```

**Run migrations:**

```bash
PG_IP=$(docker inspect outreacher-pg --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}')
docker run --rm \
  -v "$(pwd)/mcp:/src" -w /src \
  --add-host=postgres:$PG_IP \
  golang:1.24-alpine \
  sh -c 'go run github.com/pressly/goose/v3/cmd/goose@latest \
    -dir migrations postgres \
    "postgresql://admin:admin@postgres:5432/outreacher" up'
```

**Claude Desktop config** (`Settings → Developer → Edit Config`):

```json
{
  "mcpServers": {
    "outreacher": {
      "command": "wsl.exe",
      "args": ["/home/<user>/Code/VitruvianTech/outreacher/bin/outreacher-mcp-0.1.0-linux-amd64"],
      "env": {
        "DATABASE_URL": "postgresql://mcp:mcp@localhost:5432/outreacher",
        "ANTHROPIC_API_KEY": "sk-ant-...",
        "MCP_LLM_PROVIDER": "anthropic"
      }
    }
  }
}
```

Replace `<user>` with your WSL username. `ANTHROPIC_API_KEY` enables LLM-based CSV extraction for unknown formats; omit to use the built-in parsers only. Restart Claude Desktop after saving — the hammer icon confirms tools are connected.

The binary idempotently bootstraps `system_default_org`, `system_default_user`, and the Default campaign on first run.

---

## Mode 2 — Docker Compose (dev stack)

Copy `.env.example` to `.env` and fill in API keys, then:

```bash
docker compose up --build -d
```

| Service | Host port | Notes |
|---|---|---|
| `app` | 3000 | Next.js dev server |
| `mcp` | 3001 | Go SSE server, air hot reload |
| `postgres` | — | Internal only |

**First-time migrations:**

```bash
docker run --rm --network outreacher_default \
  -v "$(pwd)/mcp:/src" -w /src \
  golang:1.24-alpine \
  sh -c 'go run github.com/pressly/goose/v3/cmd/goose@latest \
    -dir migrations postgres \
    "postgresql://admin:admin@outreacher-postgres-1:5432/outreacher" up'
```

If the mcp container exited while waiting for migrations, restart it:

```bash
docker compose restart mcp
```

---

## Imports

`fixtures/imports/` is gitignored — it contains live personal data used for local development and debugging only. Supported formats: Gojiberry, Revli startup contacts, Revli investor contacts, Revli startup companies, Revli investor companies.
