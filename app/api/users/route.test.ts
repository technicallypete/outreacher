import { describe, it, expect } from "bun:test";

const BASE_URL = process.env.APP_URL ?? "http://localhost:3000";

describe("POST /api/users", () => {
  it("returns 400 when name is missing", async () => {
    const res = await fetch(`${BASE_URL}/api/users`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ slug: "no-name" }),
    });
    expect(res.status).toBe(400);
    const data = await res.json();
    expect(data).toHaveProperty("error");
  });

  it("returns 400 when slug is missing", async () => {
    const res = await fetch(`${BASE_URL}/api/users`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: "No Slug" }),
    });
    expect(res.status).toBe(400);
  });

  it("creates a user and returns MCP content envelope", async () => {
    const slug = `test-user-${Date.now()}`;
    const res = await fetch(`${BASE_URL}/api/users`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: "Test User", slug }),
    });
    expect(res.status).toBe(201);
    const data = await res.json();
    expect(data).toHaveProperty("content");

    const user = JSON.parse(data.content[0].text);
    expect(user.slug).toBe(slug);
    expect(user.name).toBe("Test User");
    expect(user.is_system).toBe(false);
  });
});
