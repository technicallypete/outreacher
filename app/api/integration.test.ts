/**
 * Full end-to-end integration test for the multi-tenant API flow.
 *
 * Runs sequentially within a single describe block. Each step captures IDs
 * consumed by subsequent steps. Requires the full Docker Compose stack to be
 * running (Next.js + MCP SSE server + Postgres with migrations applied).
 *
 * Run via: bun test app/api/integration.test.ts
 */
import { describe, it, expect, beforeAll } from "bun:test";

const BASE_URL = process.env.APP_URL ?? "http://localhost:3000";

// State captured across steps
let orgId: number;
let campaignId: number;
let userId: number;

const suffix = Date.now();
const ORG_NAME = `Integration Org ${suffix}`;
const USER_SLUG = `integration-user-${suffix}`;

describe("Multi-tenant API integration flow", () => {
  // ── Step 1: Create org ──────────────────────────────────────────────────
  it("Step 1 — POST /api/orgs creates org with Default campaign", async () => {
    const res = await fetch(`${BASE_URL}/api/orgs`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: ORG_NAME }),
    });
    expect(res.status).toBe(201);

    const data = await res.json();
    expect(data).toHaveProperty("content");

    const result = JSON.parse(data.content[0].text);
    expect(result).toHaveProperty("organization");
    expect(result).toHaveProperty("default_campaign");

    orgId = result.organization.id;
    campaignId = result.default_campaign.id;

    expect(orgId).toBeNumber();
    expect(campaignId).toBeNumber();
    expect(result.default_campaign.is_default).toBe(true);
  });

  // ── Step 2: Create user ─────────────────────────────────────────────────
  it("Step 2 — POST /api/users creates a user", async () => {
    const res = await fetch(`${BASE_URL}/api/users`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: "Integration User",
        slug: USER_SLUG,
        email: `${USER_SLUG}@example.com`,
      }),
    });
    expect(res.status).toBe(201);

    const data = await res.json();
    const user = JSON.parse(data.content[0].text);

    userId = user.id;
    expect(userId).toBeNumber();
    expect(user.slug).toBe(USER_SLUG);
    expect(user.is_system).toBe(false);
  });

  // ── Step 3: Assign user to org as owner ─────────────────────────────────
  it("Step 3 — POST /api/orgs/:id/members assigns user as owner", async () => {
    const res = await fetch(`${BASE_URL}/api/orgs/${orgId}/members`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ user_id: userId, role: "owner" }),
    });
    expect(res.status).toBe(200);

    const data = await res.json();
    expect(data).toHaveProperty("content");
    const result = JSON.parse(data.content[0].text);
    expect(result.org_id).toBe(orgId);
    expect(result.user_id).toBe(userId);
    expect(result.role).toBe("owner");
  });

  // ── Step 4: Assign user to campaign as admin ────────────────────────────
  it("Step 4 — POST /api/campaigns/:id/members assigns user as admin", async () => {
    const res = await fetch(`${BASE_URL}/api/campaigns/${campaignId}/members`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ user_id: userId, role: "admin" }),
    });
    expect(res.status).toBe(200);

    const data = await res.json();
    expect(data).toHaveProperty("content");
    const result = JSON.parse(data.content[0].text);
    expect(result.campaign_id).toBe(campaignId);
    expect(result.user_id).toBe(userId);
    expect(result.role).toBe("admin");
  });

  // ── Step 5: Import leads via multipart (small inline CSV) ──────────────
  it("Step 5 — POST /api/leads/import uploads sample CSV into the new campaign", async () => {
    // Use a small inline CSV so the test doesn't depend on LLM extraction speed.
    const csv = [
      "First Name,Last Name,Email,Email 2,Email 3,Phone,Phone 2,Phone 3,Location,Job Title,Industry,Company,Company URL,Website,Import Date,Intent,Profile URL,Total Score,Intent Keyword,Personnalized Email message,Personnalized LinkedIn message",
      `Jane,Integration,jane.integration.${suffix}@example.com,,,,,,,New York,CTO,Software,IntegrationCo ${suffix},https://www.linkedin.com/company/integrationco${suffix}/,https://integrationco.example.com,Apr 04 2026,Just engaged with a LinkedIn post,https://www.linkedin.com/in/janeintegration${suffix},2.00,enterprise,`,
    ].join("\n");

    const form = new FormData();
    form.append("file", new Blob([csv], { type: "text/csv" }), "leads.csv");
    form.append("campaign_id", String(campaignId));

    const res = await fetch(`${BASE_URL}/api/leads/import`, {
      method: "POST",
      body: form,
    });
    expect(res.status).toBe(200);

    const data = await res.json();
    expect(data).toHaveProperty("content");

    const summary = JSON.parse(data.content[0].text);
    expect(summary).toHaveProperty("leads");
    expect(summary).toHaveProperty("companies");
    expect(summary).toHaveProperty("skipped");
    // Sample file has data rows — at least some leads should be imported
    expect(summary.leads + summary.skipped).toBeGreaterThan(0);
  });

  // ── Step 6: Get leads for the new campaign ──────────────────────────────
  it("Step 6 — GET /api/leads?campaign_id=:id returns leads for the new campaign", async () => {
    const res = await fetch(`${BASE_URL}/api/leads?campaign_id=${campaignId}`);
    expect(res.status).toBe(200);

    const data = await res.json();
    expect(data).toHaveProperty("content");

    const leads = JSON.parse(data.content[0].text);
    expect(Array.isArray(leads)).toBe(true);
    expect(leads.length).toBeGreaterThan(0);

    for (const lead of leads) {
      expect(lead.id).toBeNumber();
      expect(lead.name).toBeString();
      expect(lead.status).toBe("new");
    }
  });
});
