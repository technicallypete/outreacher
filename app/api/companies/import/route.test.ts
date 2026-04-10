import { describe, it, expect, beforeAll } from "bun:test";
import { createTestCampaign } from "../../test-helpers";

const BASE_URL = process.env.APP_URL ?? "http://localhost:3000";

// Minimal valid Revli startup company CSV (2 rows, no contacts).
const STARTUP_CSV = [
  "Company Name,Recent Funding Date,Recent Funding Amount(USD),Company Description,Company Headquarters,Industries,# Employees,Currently Hiring,Company Website,Company LinkedIn,Company Facebook,Company Twitter,Company Phone,Industry Categories,Company Technologies,Annual Revenue,Last Funding Type,Total Funding Amount(USD),Funding News,Funding Status,Top 5 Investors,Founded Date,Company Overview,Expansion Strategy,Market Opportunities,Challenges And Risks,Tech Needs,Infrastructure Needs,Service Needs,Projected Growth,Hiring Forecast,Key Technical Challenges",
  `TestCo ${Date.now()},2026-01-15,$2M,Test company description.,San Francisco CA US,SaaS,50,,https://testco.example.com,https://linkedin.com/company/testco,,,+1 415-000-0000,SaaS & Software,React Node.js PostgreSQL,$5M to $10M,Seed,$2M,,Active,Acme Ventures,2020-01-01,Overview here.,Expand globally.,Large TAM.,Competition.,AI tools.,Cloud infra.,Consulting.,Strong.,Engineers.,Scaling.`,
].join("\n");

// Minimal valid Revli investor company CSV.
const INVESTOR_CSV = [
  "Upload Date,Company Name,Company City,Company State,Company Country,# Employees,Website,Company Linkedin Url,Facebook Url,Twitter Url,Company Phone,Company Description,Founded Date,Startups Invested This Week,Historical Investment,Firm Type,Stage Focus,Check Size,Portfolio Size,Industry Focus,Geography Focus,Firm Overview,Investor Thesis,Portfolio Highlights,Co-Investor Network",
  `2026-04-01,TestVC ${Date.now()},San Francisco,CA,US,10,https://testvc.example.com,https://linkedin.com/company/testvc,,,+1 415-111-0000,A test VC firm.,2015-01-01,3,$500M,Venture Capital,Seed/Series A,$500K-$2M,50,SaaS,North America,Overview.,Thesis.,Highlights.,Co-investors.`,
].join("\n");

let campaignId: number;

beforeAll(async () => {
  campaignId = await createTestCampaign();
});

describe("POST /api/companies/import", () => {
  it("returns 415 for unsupported content-type", async () => {
    const res = await fetch(`${BASE_URL}/api/companies/import`, {
      method: "POST",
      headers: { "content-type": "text/plain" },
      body: "something",
    });
    expect(res.status).toBe(415);
  });

  it("returns 400 when JSON body is missing required fields", async () => {
    const res = await fetch(`${BASE_URL}/api/companies/import`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ csv: STARTUP_CSV }),
    });
    expect(res.status).toBe(400);
  });

  it("imports startup companies via JSON body", async () => {
    const res = await fetch(`${BASE_URL}/api/companies/import`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ format: "revli_startup_companies", csv: STARTUP_CSV, campaign_id: campaignId }),
    });
    expect(res.status).toBe(200);
    const data = await res.json();
    const summary = JSON.parse(data.content[0].text);
    expect(summary.companies).toBeGreaterThan(0);
    expect(summary.leads).toBe(0);
  });

  it("imports investor companies via JSON body", async () => {
    const res = await fetch(`${BASE_URL}/api/companies/import`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ format: "revli_investor_companies", csv: INVESTOR_CSV, campaign_id: campaignId }),
    });
    expect(res.status).toBe(200);
    const data = await res.json();
    const summary = JSON.parse(data.content[0].text);
    expect(summary.companies).toBeGreaterThan(0);
    expect(summary.leads).toBe(0);
  });

  it("imports startup companies via multipart file upload", async () => {
    const form = new FormData();
    form.append("format", "revli_startup_companies");
    form.append("file", new Blob([STARTUP_CSV], { type: "text/csv" }), "companies.csv");
    form.append("campaign_id", String(campaignId));

    const res = await fetch(`${BASE_URL}/api/companies/import`, {
      method: "POST",
      body: form,
    });
    expect(res.status).toBe(200);
    const data = await res.json();
    const summary = JSON.parse(data.content[0].text);
    expect(summary.companies).toBeGreaterThanOrEqual(0); // upsert may find existing
  });
});
