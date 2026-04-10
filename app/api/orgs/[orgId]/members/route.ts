import pool from "@/lib/db";
import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";

const AssignOrgMemberSchema = z.object({
  user_id: z.number().int().positive(),
  role: z.enum(["owner", "admin", "member"]),
});

export async function POST(
  req: NextRequest,
  { params }: { params: Promise<{ orgId: string }> }
) {
  const { orgId } = await params;
  const parsed = AssignOrgMemberSchema.safeParse(await req.json());
  if (!parsed.success) {
    return NextResponse.json({ error: parsed.error.issues }, { status: 400 });
  }

  const { user_id, role } = parsed.data;
  const org_id = Number(orgId);

  await pool.query(
    `INSERT INTO app.organization_memberships (organization_id, user_id, role)
     VALUES ($1, $2, $3)
     ON CONFLICT (organization_id, user_id) DO NOTHING`,
    [org_id, user_id, role]
  );

  return NextResponse.json({
    content: [{ type: "text", text: JSON.stringify({ org_id, user_id, role }) }],
  });
}
