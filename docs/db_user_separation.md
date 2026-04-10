# DB User Separation

## Goal

Split the single `outreacher` superuser into two roles with least-privilege access:

| User | Password | Role | Used by |
|---|---|---|---|
| `admin` | `admin` | Owns schema, full DDL | goose migrations only |
| `app` | `app` | DML only (SELECT/INSERT/UPDATE/DELETE) | MCP binary, Next.js app |

`DATABASE_URL` → `app` user (runtime)
`DATABASE_ADMIN_URL` → `admin` user (migrations)

---

## Implementation

### 1. `postgres/init/01_users_and_grants.sql` (new file)

Runs once at postgres container first-start via `docker-entrypoint-initdb.d/`, as the postgres superuser.

```sql
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'app') THEN
    CREATE ROLE app WITH LOGIN PASSWORD 'app';
  END IF;
END
$$;

GRANT CONNECT ON DATABASE outreacher TO app;
GRANT USAGE ON SCHEMA app TO app;

-- DML on existing tables (covers restarts against populated volumes)
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA app TO app;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA app TO app;
GRANT USAGE ON ALL TYPES IN SCHEMA app TO app;

-- DML on all future tables/sequences/types created by admin
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app;

ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  GRANT USAGE ON SEQUENCES TO app;

ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  GRANT USAGE ON TYPES TO app;
```

The `DEFAULT PRIVILEGES` lines are the keystone — they fire automatically whenever goose (running as `admin`) creates a new table, sequence, or type. No `GRANT` statements needed in individual migrations.

> **Note:** The init script only runs on a fresh volume. For existing instances, run it manually:
> ```bash
> docker exec -i outreacher-postgres psql -U admin -d outreacher \
>   < postgres/init/01_users_and_grants.sql
> ```

### 2. `docker-compose.yml`

- Mount init script: `./postgres/init:/docker-entrypoint-initdb.d`
- Change `POSTGRES_USER` / `POSTGRES_PASSWORD` → `admin` / `admin`
- Change `DATABASE_URL` in `app` and `mcp` services → `app:app`
- Add `DATABASE_ADMIN_URL` to `mcp` service → `admin:admin`
- Update healthcheck: `-U admin`

### 3. `mcp/internal/db/db.go`

Change hardcoded fallback DSN from `outreacher:outreacher` to `app:app`.

### 4. `README.md`

- Standalone `docker run` postgres: `POSTGRES_USER=admin`, add init mount
- All goose commands: use `admin:admin`
- Seed commands: use `app:app` (DML only)
- Claude Desktop config: `DATABASE_URL` → `app:app`
- Test commands: `DATABASE_URL` → `app:app`
- Env vars table: add `DATABASE_ADMIN_URL`

### 5. No changes needed

- `mcp/migrations/*.sql` — goose runs as `admin`, `DEFAULT PRIVILEGES` handles grants automatically
- `mcp/seed.sql` — pure DML, no embedded connection string
- All Go source in `mcp/internal/` — only issues DML

---

## Sequencing (fresh environment)

```
1. postgres starts
   └── 01_users_and_grants.sql runs once
       creates app role, sets DEFAULT PRIVILEGES for admin

2. goose up (as admin)
   └── creates schema, tables, types, sequences
   └── DEFAULT PRIVILEGES fires per object → app gets DML grants automatically

3. MCP binary / Next.js app connect as app
   └── search_path=app appended to DSN by NewPool()
   └── DML succeeds, DDL denied

4. seed.sql runs as app (optional)
```

---

## Environment Variables

| Variable | Value (compose) | Value (standalone) |
|---|---|---|
| `DATABASE_URL` | `postgresql://app:app@postgres:5432/outreacher` | `postgresql://app:app@localhost:5432/outreacher` |
| `DATABASE_ADMIN_URL` | `postgresql://admin:admin@postgres:5432/outreacher` | `postgresql://admin:admin@localhost:5432/outreacher` |
