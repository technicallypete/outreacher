import { auth } from "@/lib/auth";
import pool from "@/lib/db";
import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";

const Schema = z.object({
  orgName: z.string().min(1).max(255),
});

function slugify(s: string) {
  return s
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "")
    .slice(0, 60);
}

export async function POST(req: NextRequest) {
  const session = await auth();
  if (!session?.userId) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const parsed = Schema.safeParse(await req.json());
  if (!parsed.success) {
    return NextResponse.json({ error: "orgName is required" }, { status: 400 });
  }

  const { orgName } = parsed.data;
  const userId = session.userId;
  const client = await pool.connect();

  try {
    await client.query("BEGIN");

    // Create organization (append suffix if slug already taken)
    const baseSlug = slugify(orgName);
    const { rows: slugRows } = await client.query<{ count: string }>(
      `SELECT COUNT(*) FROM app.organizations WHERE slug LIKE $1`,
      [`${baseSlug}%`]
    );
    const count = parseInt(slugRows[0].count, 10);
    const orgSlug = count === 0 ? baseSlug : `${baseSlug}_${count}`;
    const { rows: orgRows } = await client.query<{ id: number }>(
      `INSERT INTO app.organizations (name, slug)
       VALUES ($1, $2)
       RETURNING id`,
      [orgName, orgSlug]
    );
    const orgId = orgRows[0].id;

    // Create default campaign
    const { rows: campaignRows } = await client.query<{ id: number }>(
      `INSERT INTO app.campaigns (organization_id, name, slug, is_default)
       VALUES ($1, 'Default', 'default', true)
       RETURNING id`,
      [orgId]
    );
    const campaignId = campaignRows[0].id;

    // Assign user as org owner
    await client.query(
      `INSERT INTO app.organization_memberships (organization_id, user_id, role)
       VALUES ($1, $2, 'owner'::app.org_role)`,
      [orgId, userId]
    );

    // Assign user as campaign admin
    await client.query(
      `INSERT INTO app.campaign_memberships (campaign_id, user_id, role)
       VALUES ($1, $2, 'admin'::app.campaign_role)`,
      [campaignId, userId]
    );

    await client.query("COMMIT");

    return NextResponse.json({ orgId, campaignId });
  } catch (err) {
    await client.query("ROLLBACK");
    console.error("onboarding error:", err);
    return NextResponse.json(
      { error: "Failed to create organization" },
      { status: 500 }
    );
  } finally {
    client.release();
  }
}
