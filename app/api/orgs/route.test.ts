import { describe, it, expect } from "bun:test";

const BASE_URL = process.env.APP_URL ?? "http://localhost:3000";

describe("POST /api/orgs", () => {
  it("returns 400 when name is missing", async () => {
    const res = await fetch(`${BASE_URL}/api/orgs`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({}),
    });
    expect(res.status).toBe(400);
    const data = await res.json();
    expect(data).toHaveProperty("error");
  });

  it("creates an org and returns MCP content envelope", async () => {
    const res = await fetch(`${BASE_URL}/api/orgs`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: `Test Org ${Date.now()}` }),
    });
    expect(res.status).toBe(201);
    const data = await res.json();
    expect(data).toHaveProperty("content");
    expect(Array.isArray(data.content)).toBe(true);

    const result = JSON.parse(data.content[0].text);
    expect(result).toHaveProperty("organization");
    expect(result).toHaveProperty("default_campaign");
    expect(result.organization.name).toBeString();
    expect(result.default_campaign.slug).toBe("default");
  });
});
