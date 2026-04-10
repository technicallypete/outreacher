import pool from "@/lib/db";
import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";

const CreateOrgSchema = z.object({
  name: z.string().min(1),
  slug: z.string().optional(),
});

function slugify(s: string): string {
  return s.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "");
}

export async function POST(req: NextRequest) {
  const parsed = CreateOrgSchema.safeParse(await req.json());
  if (!parsed.success) {
    return NextResponse.json({ error: parsed.error.issues }, { status: 400 });
  }

  const { name } = parsed.data;
  const slug = parsed.data.slug ?? slugify(name);

  const { rows: [org] } = await pool.query(
    `INSERT INTO app.organizations (name, slug, is_system) VALUES ($1, $2, false) RETURNING id, name, slug, is_system`,
    [name, slug]
  );
  const { rows: [campaign] } = await pool.query(
    `INSERT INTO app.campaigns (organization_id, name, slug, is_default) VALUES ($1, 'Default', 'default', true) RETURNING id, name, slug, is_default`,
    [org.id]
  );

  return NextResponse.json(
    { content: [{ type: "text", text: JSON.stringify({ organization: org, default_campaign: campaign }) }] },
    { status: 201 }
  );
}
