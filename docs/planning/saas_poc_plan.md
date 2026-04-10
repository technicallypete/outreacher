# SaaS POC Plan

## Overview

Convert Outreacher from a Claude Desktop MCP tool into a web-based SaaS with an AI chat interface, auth, onboarding, and a leads review UI. The backend (DB schema, MCP tools, Next.js API routes) is production-ready — this plan covers the missing SaaS layer.

## Stack

- **Auth:** NextAuth.js v5 with magic link via Resend (email provider)
- **UI components:** shadcn/ui + Tailwind CSS
- **Chat:** Vercel AI SDK (server-side streaming) + assistant-ui (pre-built chat components)
- **Runtime:** Bun + Next.js 15 App Router (existing)

## Architecture

```
Browser
  └── Next.js App Router (:3000)
        ├── (auth)/          — sign-in, magic link verify
        ├── (app)/           — protected layout (session required)
        │     ├── onboarding — create org → default brand → session
        │     ├── chat       — AI chat interface
        │     └── leads      — leads review UI
        └── api/
              ├── auth/[...nextauth]  — NextAuth handlers
              └── chat               — Vercel AI SDK streaming route
```

### Auth ↔ DB Linking

NextAuth stores sessions/accounts in its own tables (`accounts`, `sessions`, `verification_tokens`). These are linked to the existing `users` table via `users.email` (already present, unique, nullable). On first sign-in, a `users` row is created and linked to the NextAuth account. After onboarding, `brand_id` is stored in the session token so every request has brand context without a DB lookup.

### Chat Architecture

```
useChat (assistant-ui)
  └── POST /api/chat
        └── streamText (AI SDK, Claude claude-sonnet-4-6)
              └── MCP tools via callTool() (existing app/lib/mcp.ts)
```

Claude receives the user's `brand_id` as a system prompt prefix so all tool calls are automatically scoped to the right brand. Tool calls are streamed back to the UI and rendered inline by assistant-ui — users see "Searching leads...", "Importing CSV..." in real time, mirroring the Claude Desktop experience.

## Phases

### Phase 1 — Auth + Onboarding

**Goal:** A user can sign in with a magic link and create their org.

Steps:
1. Install NextAuth v5, Resend adapter, shadcn/ui, Tailwind
2. Add NextAuth tables to DB (migration) linked to `users.email`
3. `/sign-in` — email input → Resend sends magic link
4. `/onboarding` — org name input → calls `create_organization` MCP tool → sets `brandId` in session → redirect to `/chat`
5. Middleware: protect all `/(app)/` routes, redirect unauthenticated users to `/sign-in`, redirect users without an org to `/onboarding`

**Session shape:**
```ts
{
  user: { id: string; email: string; name?: string },
  userId: number,    // outreacher users.id
  brandId: number,   // default brand for this session
  orgId: number,
}
```

### Phase 2 — Chat Interface

**Goal:** Users can manage leads and import CSVs through an AI chat interface, matching the Claude Desktop flow.

Steps:
1. Install Vercel AI SDK, assistant-ui
2. `POST /api/chat` — streaming route; attaches MCP tools from `app/lib/mcp.ts`; injects `brand_id` into system prompt
3. `/chat` page — assistant-ui Thread component wired to `useChat`
4. Tool call display — show tool name + condensed result inline in the thread

**System prompt pattern:**
```
You are an outreach assistant. The user's active brand ID is {brandId}.
Always pass brand_id: {brandId} when calling any domain tool.
```

### Phase 3 — Leads UI

**Goal:** A dedicated page to review, approve, follow up on, and reject leads.

Steps:
1. `GET /leads` page — table of leads for the session brand (hits existing `GET /api/leads`)
2. `PATCH /api/leads/[id]/status` — thin wrapper around `update_lead_status` MCP tool
3. Status actions: Approve → `qualified`, Reject → `disqualified`, Follow Up → `contacted`, Convert → `converted`
4. Lead detail drawer — notes history, status timeline, company info

## What's Explicitly Out of Scope (POC)

- Brand / org CRUD UI (managed through chat or MCP tools)
- Member management
- Billing / subscriptions
- Rate limiting
- Multiple brands per session (brand switcher)
- Company browser UI (accessible through chat)

## Key Files to Create

| File | Purpose |
|---|---|
| `app/(auth)/sign-in/page.tsx` | Magic link sign-in form |
| `app/(app)/layout.tsx` | Protected layout with session check |
| `app/(app)/onboarding/page.tsx` | Org creation flow |
| `app/(app)/chat/page.tsx` | AI chat interface |
| `app/(app)/leads/page.tsx` | Leads review table |
| `app/api/auth/[...nextauth]/route.ts` | NextAuth handler |
| `app/api/chat/route.ts` | Vercel AI SDK streaming endpoint |
| `app/lib/auth.ts` | NextAuth config (Resend + DB adapter) |
| `app/middleware.ts` | Route protection + onboarding redirect |
| `mcp/migrations/00008_nextauth.sql` | NextAuth tables + users.email link |
