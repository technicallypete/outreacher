import { describe, it, expect, beforeAll } from "bun:test";
import { createTestCampaign } from "../../test-helpers";

const BASE_URL = process.env.APP_URL ?? "http://localhost:3000";

const ts = Date.now();
const SAMPLE_CSV = [
  "First Name,Last Name,Email,Email 2,Email 3,Phone,Phone 2,Phone 3,Location,Job Title,Industry,Company,Company URL,Website,Import Date,Intent,Profile URL,Total Score,Intent Keyword,Personnalized Email message,Personnalized LinkedIn message",
  `Jane,Doe,jane.route.test.${ts}@example.com,,,,,,,New York,CTO,Software Development,TestCo ${ts},https://www.linkedin.com/company/testco${ts}/,https://testco.example.com,Apr 04 2026,Just engaged with a LinkedIn post,https://www.linkedin.com/in/janedoe${ts},2.00,enterprise,`,
].join("\n");

let campaignId: number;

beforeAll(async () => {
  campaignId = await createTestCampaign();
});

describe("POST /api/leads/import", () => {
  it("returns 415 for unsupported content-type", async () => {
    const res = await fetch(`${BASE_URL}/api/leads/import`, {
      method: "POST",
      headers: { "Content-Type": "text/plain" },
      body: "some text",
    });
    expect(res.status).toBe(415);
  });

  it("returns 400 for JSON body missing csv field", async () => {
    const res = await fetch(`${BASE_URL}/api/leads/import`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ not_csv: "oops" }),
    });
    expect(res.status).toBe(400);
  });

  it("returns 400 for multipart missing file field", async () => {
    const form = new FormData();
    form.append("not_file", "oops");
    const res = await fetch(`${BASE_URL}/api/leads/import`, {
      method: "POST",
      body: form,
    });
    expect(res.status).toBe(400);
  });

  it("imports via JSON body and returns summary", async () => {
    const res = await fetch(`${BASE_URL}/api/leads/import`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ csv: SAMPLE_CSV, campaign_id: campaignId }),
    });
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(data).toHaveProperty("content");
    const summary = JSON.parse(data.content[0].text);
    expect(summary).toHaveProperty("leads");
    expect(summary).toHaveProperty("companies");
  });

  it("imports via multipart upload and returns summary", async () => {
    const form = new FormData();
    form.append(
      "file",
      new Blob([SAMPLE_CSV], { type: "text/csv" }),
      "leads.csv"
    );
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
  });
});
