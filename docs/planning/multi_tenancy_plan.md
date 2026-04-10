# Multi-Tenancy Plan

## Hierarchy

```
Organization
  └── Brand (one org has many brands; always has a "Default" brand)
        └── Leads, Companies, Signals (all domain objects scoped to brand_id)

User
  ├── organization_memberships  (role: owner | admin | member)
  └── brand_memberships         (role: admin | member | viewer)
```

Users are many:many with orgs, and many:many with brands. A brand always belongs to exactly one org. Brands are not shared across orgs.

---

## Domain Object Scoping

| Table | Scoped to |
|---|---|
| `companies` | `brand_id` |
| `leads` | `brand_id` |
| `contact_identifiers` | `brand_id` (part of PK) |
| `company_signals` | inherits via company |
| `notes` | inherits via lead |
| `signals` | global — generic descriptors, no brand affinity |
| `signal_keywords` | global — tied to signals |

`contact_identifiers` PK changes from `(type, value)` to `(brand_id, type, value)` so the same email can exist as separate leads across brands.

`companies` unique constraint changes from `name` to `(brand_id, name)`.

---

## New Tables (migration 00004)

```sql
app.organizations       -- id, name, slug (unique), is_system, created_at
app.brands              -- id, organization_id, name, slug, is_default, created_at
app.users               -- id, email (nullable), slug (unique), name, is_system, created_at
app.organization_memberships  -- organization_id, user_id, role: owner|admin|member
app.brand_memberships         -- brand_id, user_id, role: admin|member|viewer
```

`slug` is the stable string identifier used for env var lookups and bootstrap detection (e.g. `system_default_org`, `system_default_user`).

`is_system` on orgs and users allows the SaaS UI to filter out system-only defaults from tenant-facing views.

---

## Migrations

### 00004 — multi-tenancy tables

Creates `organizations`, `brands`, `users`, `organization_memberships`, `brand_memberships`.

Tail of the `Up` block seeds the two rows needed for migration 00005's backfill:

```sql
-- Minimal seed so 00005 can backfill brand_id on existing rows.
-- The MCP binary re-runs these at startup via ON CONFLICT DO NOTHING.
INSERT INTO app.organizations (name, slug, is_system)
VALUES ('System Default', 'system_default_org', TRUE)
ON CONFLICT (slug) DO NOTHING;

INSERT INTO app.brands (organization_id, name, slug, is_default)
SELECT id, 'Default', 'default', TRUE
FROM app.organizations WHERE slug = 'system_default_org'
ON CONFLICT (organization_id, slug) DO NOTHING;
```

No users or memberships in the migration — those are created by the binary bootstrap.

### 00005 — scope domain objects to brand

Adds `brand_id NOT NULL` to `companies`, `leads`, `contact_identifiers`. Backfills existing rows to `system_default_org`'s Default brand. Updates unique constraints and `contact_identifiers` PK.

---

## System Defaults and MCP Binary Bootstrap

No SQL seed script. Instead, the binary detects and idempotently creates system defaults at startup using the same DB query functions that back the MCP tools.

### Bootstrap sequence (runs before tools are registered)

1. Does an org with slug `MCP_ORG_SLUG` exist? If not → `CreateOrganization` (which also creates its Default brand as a side effect)
2. Does a user with slug `MCP_USER_SLUG` exist? If not → `CreateUser`
3. Is the user a member of the org? If not → `AddOrgMember` (role: owner)
4. Is the user a member of the Default brand? If not → `AddBrandMember` (role: admin)
5. Resolve Default brand → store `brandID` as the startup default

Each step is checked independently to handle partial states (e.g. org exists but user doesn't).

### Env vars

| Var | Default |
|---|---|
| `MCP_ORG_SLUG` | `system_default_org` |
| `MCP_USER_SLUG` | `system_default_user` |

### `CreateOrganization` convention

Wherever an org is created — binary bootstrap, SaaS API, or MCP tool — a Default brand is always created as a side effect. This logic lives in the `CreateOrganization` DB query/service function, not in each caller.

### SSE server / SaaS

The SSE server does **not** run bootstrap. Orgs, brands, and users are created through the SaaS API. The SSE server resolves its operating brand from the authenticated request context (future work).

---

## MCP Tools (new and updated)

### Brand and tenant management tools

| Tool | Args | Description |
|---|---|---|
| `list_organizations` | — | List all orgs the current user belongs to |
| `list_brands` | `org_id?` | List all brands for the current org (default) or a specified org |
| `create_organization` | `name` | Create a new org + Default brand; assign current user as owner and Default brand admin |
| `create_brand` | `name`, `org_id?` | Create a new brand under the current org (or specified org) |
| `create_user` | `name`, `email?` | Create a new user |
| `assign_user_to_org` | `user_id`, `org_id`, `role` | Add a user to an org with a given role |
| `assign_user_to_brand` | `user_id`, `brand_id`, `role` | Add a user to a brand with a given role |

### Updated domain tools — `brand_id` parameter

Domain tools (`import_leads`, `search_leads`, `get_lead`, `update_lead_status`, `create_followup_note`) gain an optional `brand_id` parameter. When omitted, the startup default brand is used. Claude passes an explicit `brand_id` when the user has targeted a specific brand.

---

## Mid-Session Brand Switching (Claude Desktop)

The binary's resolved `brandID` is the **default fallback**, not a locked session variable. Claude holds brand context in its own reasoning and passes `brand_id` explicitly on domain tool calls when needed.

Example flow for "add these leads to Foo brand":

1. Claude calls `list_brands` — Foo not found
2. Claude: "There's no Foo brand. Would you like me to create it?"
3. User confirms → Claude calls `create_brand name=Foo`
4. Claude calls `import_leads csv=... brand_id=<new id>`

No `set_active_brand` tool or server-side session state is needed. This pattern works naturally with how Claude reasons across tool calls.

---

## `tools.Register()` signature

```go
// Before
func Register(s *server.MCPServer, q *db.Queries)

// After
func Register(s *server.MCPServer, q *db.Queries, brandID int32)
```

All tool handler closures capture `brandID` as the default. Tools with an optional `brand_id` argument override it when provided.

---

## File Change Summary

| File | Action |
|---|---|
| `mcp/migrations/00004_multi_tenancy.sql` | New — org/brand/user/membership tables + minimal seed |
| `mcp/migrations/00005_brand_scope_domain.sql` | New — add brand_id to companies/leads/identifiers; update constraints; backfill |
| `mcp/internal/db/queries/tenants.sql` | New — GetOrgBySlug, GetUserBySlug, GetDefaultBrandForOrg, CreateOrganization, CreateBrand, CreateUser, AddOrgMember, AddBrandMember, ListBrands, ListOrganizations |
| `mcp/internal/db/queries/companies.sql` | Edit — add brand_id to UpsertCompany |
| `mcp/internal/db/queries/leads.sql` | Edit — add brand_id to all queries |
| `mcp/internal/db/queries/identifiers.sql` | Edit — add brand_id to FindLeadByIdentifier and UpsertIdentifier |
| `mcp/internal/db/gen/*` | Regenerated by `sqlc generate` |
| `mcp/internal/tenant/tenant.go` | New — Bootstrap() function: detect and idempotently create system defaults, return resolved Tenant{OrgID, BrandID, UserID} |
| `mcp/internal/tools/tools.go` | Edit — Register signature adds brandID int32 |
| `mcp/internal/tools/import_leads.go` | Edit — optional brand_id arg; pass to company/lead/identifier queries |
| `mcp/internal/tools/search_leads.go` | Edit — optional brand_id arg; filter by brand |
| `mcp/internal/tools/get_lead.go` | Edit — brand-scoped lookup |
| `mcp/internal/tools/update_lead_status.go` | Edit — brand-scoped update |
| `mcp/internal/tools/create_followup_note.go` | Edit — brand-scoped lead lookup |
| `mcp/internal/tools/orgs.go` | New — list_organizations, create_organization, create_brand, list_brands |
| `mcp/internal/tools/users.go` | New — create_user, assign_user_to_org, assign_user_to_brand |
| `mcp/cmd/mcp/main.go` | Edit — call tenant.Bootstrap(), pass brandID to Register |
| `mcp/cmd/server/main.go` | Edit — call tenant.Bootstrap() (SSE dev stack uses env-var defaults for now) |
