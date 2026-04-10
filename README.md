# Outreacher

A lead management system with an MCP server for Claude Desktop and a Next.js SaaS frontend.

> **All dev tooling runs via Docker.** You do not need Go, Node, Bun, goose, or sqlc installed locally.
>
> **Developers:** see `CLAUDE.md` for commands, conventions, and test instructions.

---

## Architecture

```
Claude Desktop
  └── bin/outreacher-mcp-*  (Go stdio binary, WSL)
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
| `app` | `app` | Runtime — DML only (SELECT/INSERT/UPDATE/DELETE) |

`DATABASE_URL` → `app` user. `DATABASE_ADMIN_URL` → `admin` user (goose only).

---

## Schema (`app`)

### Multi-tenancy

| Table | Key fields |
|---|---|
| `organizations` | id, name, slug (unique), is_system |
| `brands` | id, organization_id, name, slug, is_default |
| `users` | id, email†, slug (unique), name, is_system |
| `organization_memberships` | organization_id, user_id, role (owner\|admin\|member) |
| `brand_memberships` | brand_id, user_id, role (admin\|member\|viewer) |

### Domain (scoped to `brand_id`)

| Table | Key fields |
|---|---|
| `companies` | id, brand_id, name, domain, industry, linkedin_url |
| `signals` | id, description — global, no brand scope |
| `signal_keywords` | signal_id, keyword — global |
| `company_signals` | company_id, signal_id |
| `leads` | id, brand_id, name, email†, linkedin_url†, company_id, title, status, score, location, phone |
| `notes` | id, lead_id, content, created_at |
| `contact_identifiers` | brand_id, type, value — dedup key, PK is (brand_id, type, value) |

† nullable. Leads are deduplicated by `contact_identifiers`, not by email.

Lead status flow: `new → contacted → qualified → disqualified → converted`

---

## MCP Tools

### Domain tools (all accept optional `brand_id`; defaults to startup brand)

| Tool | Description |
|---|---|
| `search_leads` | Filter by name, email, status, company, brand_id |
| `get_lead` | Full lead detail with company and notes |
| `update_lead_status` | Advance a lead through the status flow |
| `create_followup_note` | Append a note to a lead |
| `import_leads` | Import from Gojiberry CSV text — batch max 20 rows per call |
| `import_leads_file` | Import from a CSV file path on disk |

### Org / brand / user management

| Tool | Description |
|---|---|
| `list_organizations` | List all orgs the current user belongs to |
| `list_brands` | List all brands for the current org (or specified org_id) |
| `create_organization` | Create a new org + Default brand; assign current user as owner |
| `rename_brand` | Rename a brand's display name (slug unchanged) |
| `create_brand` | Create a new brand under the current org |
| `create_user` | Create a new user |
| `assign_user_to_org` | Add a user to an org with a role |
| `assign_user_to_brand` | Add a user to a brand with a role |

---

## Mode 1 — Claude Desktop (Go binary via WSL)

**Build the binary:**

```bash
bun run build:mcp
# outputs bin/outreacher-mcp-<version>  (stdio)
#         bin/outreacher-server-<version> (SSE)
```

**Start standalone postgres:**

```bash
docker run -d --name outreacher-postgres --restart always \
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
docker run --rm \
  -v "$(pwd)/mcp:/src" -w /src \
  golang:1.24-alpine \
  sh -c "go install github.com/pressly/goose/v3/cmd/goose@v3.22.0 && \
         goose -dir migrations postgres 'postgresql://admin:admin@host.docker.internal:5432/outreacher' up"
```

**Claude Desktop config** (`Settings → Developer → Edit Config`):

```json
{
  "mcpServers": {
    "outreacher": {
      "command": "wsl.exe",
      "args": ["~/Code/VitruvianTech/outreacher/bin/outreacher-mcp-0.1.0"],
      "env": {
        "DATABASE_URL": "postgresql://app:app@localhost:5432/outreacher"
      }
    }
  }
}
```

Restart Claude Desktop — the hammer icon confirms tools are connected. The binary idempotently bootstraps `system_default_org`, `system_default_user`, and the Default brand on first run.

---

## Mode 2 — Docker Compose (dev stack)

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
  sh -c "go install github.com/pressly/goose/v3/cmd/goose@v3.22.0 && \
         goose -dir migrations postgres 'postgresql://admin:admin@postgres:5432/outreacher' up"
```

If the mcp container exited while waiting for migrations, restart it:

```bash
docker compose restart mcp
```

---

## Imports

Sample imports live in `imports/<source>/` with date-stamped filenames (e.g. `imports/gojiberry/gojiberry-selected-contacts-2026-04-04.csv`). Add new source exports as new subdirectories.
