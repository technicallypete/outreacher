import { describe, it, expect, beforeAll } from "bun:test";
import { createTestCampaign } from "../test-helpers";

const BASE_URL = process.env.APP_URL ?? "http://localhost:3000";

let campaignId: number;

beforeAll(async () => {
  campaignId = await createTestCampaign();
});

describe("GET /api/companies", () => {
  it("returns 200 with MCP content envelope", async () => {
    const res = await fetch(`${BASE_URL}/api/companies?campaign_id=${campaignId}`);
    expect(res.status).toBe(200);

    const data = await res.json();
    expect(data).toHaveProperty("content");
    expect(Array.isArray(data.content)).toBe(true);
    expect(data.content[0].type).toBe("text");
  });

  it("returns array of companies", async () => {
    const res = await fetch(`${BASE_URL}/api/companies?campaign_id=${campaignId}`);
    const data = await res.json();
    const companies = JSON.parse(data.content[0].text);
    expect(Array.isArray(companies)).toBe(true);
  });

  it("filters by is_vc=false returns non-VC companies", async () => {
    const res = await fetch(`${BASE_URL}/api/companies?campaign_id=${campaignId}&is_vc=false`);
    expect(res.status).toBe(200);
    const data = await res.json();
    const companies = JSON.parse(data.content[0].text);
    expect(Array.isArray(companies)).toBe(true);
    for (const c of companies) {
      expect(c.is_vc).toBe(false);
    }
  });

  it("filters by is_vc=true returns only VC firms", async () => {
    const res = await fetch(`${BASE_URL}/api/companies?campaign_id=${campaignId}&is_vc=true`);
    expect(res.status).toBe(200);
    const data = await res.json();
    const companies = JSON.parse(data.content[0].text);
    expect(Array.isArray(companies)).toBe(true);
    for (const c of companies) {
      expect(c.is_vc).toBe(true);
    }
  });
});
