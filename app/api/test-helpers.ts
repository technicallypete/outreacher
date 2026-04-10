const BASE_URL = process.env.APP_URL ?? "http://localhost:3000";

/**
 * Creates a throwaway org + default campaign for use in tests.
 * Returns the campaign id. Each call creates a new unique org so
 * test files are fully isolated from each other.
 */
export async function createTestCampaign(): Promise<number> {
  const res = await fetch(`${BASE_URL}/api/orgs`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name: `Test Org ${Date.now()}-${Math.random()}` }),
  });
  if (!res.ok) throw new Error(`createTestCampaign: POST /api/orgs returned ${res.status}`);
  const data = await res.json();
  const { default_campaign } = JSON.parse(data.content[0].text);
  return default_campaign.id as number;
}
