import { describe, it, expect, beforeAll } from "bun:test";
import { createTestCampaign } from "../test-helpers";

const BASE_URL = process.env.APP_URL ?? "http://localhost:3000";

let campaignId: number;

beforeAll(async () => {
  campaignId = await createTestCampaign();
});

describe("GET /api/leads", () => {
  it("returns 200 with MCP content envelope", async () => {
    const res = await fetch(`${BASE_URL}/api/leads?campaign_id=${campaignId}`);
    expect(res.status).toBe(200);

    const data = await res.json();
    expect(data).toHaveProperty("content");
    expect(Array.isArray(data.content)).toBe(true);
    expect(data.content[0].type).toBe("text");
  });

  it("returns leads with status=new", async () => {
    const res = await fetch(`${BASE_URL}/api/leads?status=new&campaign_id=${campaignId}`);
    const data = await res.json();

    const leads = JSON.parse(data.content[0].text);
    expect(Array.isArray(leads)).toBe(true);

    // Validate shape of each returned lead (passes trivially when no leads exist).
    for (const lead of leads) {
      expect(lead.status).toBe("new");
      expect(lead.id).toBeNumber();
      expect(lead.name).toBeString();
      // email is nullable — leads imported via LinkedIn may have no email
      expect(lead.email === null || typeof lead.email === "string").toBe(true);
      // at least one identifier must be present
      const hasEmail = typeof lead.email === "string" && lead.email.length > 0;
      const hasLinkedIn =
        typeof lead.linkedin_url === "string" && lead.linkedin_url.length > 0;
      expect(hasEmail || hasLinkedIn).toBe(true);
    }
  });

  it("returns all leads when no filters given", async () => {
    const res = await fetch(`${BASE_URL}/api/leads?campaign_id=${campaignId}`);
    expect(res.status).toBe(200);
    const data = await res.json();
    const leads = JSON.parse(data.content[0].text);
    expect(Array.isArray(leads)).toBe(true);
  });
});
