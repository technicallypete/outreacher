# Next.js API Routes + Integration Tests Plan

## Overview

The Next.js app exposes HTTP API routes that proxy to MCP tools via the SSE server.
Each route connects an MCP client, calls one tool, closes the connection, and returns
the result. Zod validates all request bodies at the boundary so MCP errors never
surface as unhandled 500s.

---

## Shared helper: `app/lib/mcp.ts`

Extracts the repeated SSE client boilerplate (create transport → connect → call tool →
close) so every route stays to a few lines. All routes import `callTool` from here;
no route imports the MCP SDK directly.

```typescript
export async function callTool(
  name: string,
  args: Record<string, unknown> = {}
): Promise<MCPToolResult>
```

---

## Routes

### `GET /api/leads`

Updated from hardcoded `status: "new"` to accept optional query params:

| Param | MCP arg | Notes |
|---|---|---|
| `status` | `status` | enum: new \| contacted \| qualified \| disqualified \| converted |
| `query` | `query` | name/email substring |
| `company` | `company` | company name substring |
| `brand_id` | `brand_id` | defaults to startup brand on the MCP side |

---

### `POST /api/orgs`

Creates a new organization. MCP side effect: always creates a "Default" brand and
assigns the operating user as owner + brand admin.

Request body (JSON):
```json
{ "name": "Acme Corp", "slug": "acme-corp" }
```
`slug` is optional — the MCP tool derives it from `name` if omitted.

---

### `POST /api/users`

Creates a new user.

Request body (JSON):
```json
{ "name": "Alice", "slug": "alice", "email": "alice@example.com" }
```
`email` is optional.

---

### `POST /api/orgs/[orgId]/members`

Assigns a user to an organization with a role.

Request body (JSON):
```json
{ "user_id": 2, "role": "owner" }
```
`role` enum: `owner | admin | member`

---

### `POST /api/brands/[brandId]/members`

Assigns a user to a brand with a role.

Request body (JSON):
```json
{ "user_id": 2, "role": "admin" }
```
`role` enum: `admin | member | viewer`

---

### `POST /api/leads/import`

Accepts CSV content via **either** multipart file upload **or** JSON body — both
converge to the same `import_leads` MCP tool call.

#### Multipart (browser file picker / integration test)

```
Content-Type: multipart/form-data
Fields:
  file     — CSV file (required)
  brand_id — number (optional)
```

#### JSON (programmatic / scripted)

```json
{ "csv": "<raw csv text>", "brand_id": 3 }
```

The route sniffs `Content-Type` and branches. Any other content type returns 415.

Both paths call `import_leads` with the extracted CSV text. `import_leads_file` is
not used from the Next.js side — it remains available as a Claude Desktop tool where
Claude has direct filesystem access.

Returns the import summary: `{ companies, signals, keywords, leads, skipped }`.

---

## File layout

```
app/
  lib/
    mcp.ts                              NEW — callTool helper
  api/
    leads/
      route.ts                          EDIT — accept query params
      route.test.ts                     EDIT — pass status=new explicitly
      import/
        route.ts                        NEW — dual content-type import
        route.test.ts                   NEW
    orgs/
      route.ts                          NEW
      route.test.ts                     NEW
      [orgId]/
        members/
          route.ts                      NEW
    users/
      route.ts                          NEW
      route.test.ts                     NEW
    brands/
      [brandId]/
        members/
          route.ts                      NEW
    integration.test.ts                 NEW — full sequential flow
```

---

## Integration test: `app/api/integration.test.ts`

Single file, sequential state machine. Each step captures IDs used by subsequent
steps. Tests are nested in a single `describe` block so Bun runs them in order.

```
Step 1  POST /api/orgs          → create "Test Org"       → capture orgId, brandId
Step 2  POST /api/users         → create "alice"           → capture userId
Step 3  POST /api/orgs/:id/members   → assign alice as owner
Step 4  POST /api/brands/:id/members → assign alice as admin to Default brand
Step 5  POST /api/leads/import  → multipart upload of imports/gojiberry sample
          → assert summary.leads > 0
Step 6  GET  /api/leads?brand_id=:id → assert leads returned for that brand
```

The multipart upload in step 5 reads the sample file from disk and constructs
`FormData` — simulating exactly what a browser would send:

```typescript
const csv = await Bun.file(
  "imports/gojiberry/gojiberry-selected-contacts-2026-04-04.csv"
).text();
const form = new FormData();
form.append("file", new Blob([csv], { type: "text/csv" }), "leads.csv");
```

No test-only code path in the route itself. The route is identical in test and
production.

`process.cwd()` in Bun resolves to the project root, so the relative path works
without manipulation.

---

## Colocated route tests

Each new route has a colocated `route.test.ts` covering:
- Happy path (correct input → 200/201 with expected shape)
- Missing required field → 400
- Invalid enum value → 400

`/api/orgs/[orgId]/members` and `/api/brands/[brandId]/members` are covered only in
the integration test — standalone unit tests would require hard-coded IDs and are
fragile without a running seeded DB.

---

## Validation

All POST routes use Zod schemas defined at the top of the route file. On parse
failure, return `Response.json({ error: e.issues }, { status: 400 })`.
