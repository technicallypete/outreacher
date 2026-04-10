import pool from "@/lib/db";
import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";

const AssignCampaignMemberSchema = z.object({
  user_id: z.number().int().positive(),
  role: z.enum(["admin", "member", "viewer"]),
});

export async function POST(
  req: NextRequest,
  { params }: { params: Promise<{ campaignId: string }> }
) {
  const { campaignId } = await params;
  const parsed = AssignCampaignMemberSchema.safeParse(await req.json());
  if (!parsed.success) {
    return NextResponse.json({ error: parsed.error.issues }, { status: 400 });
  }

  const { user_id, role } = parsed.data;
  const campaign_id = Number(campaignId);

  await pool.query(
    `INSERT INTO app.campaign_memberships (campaign_id, user_id, role)
     VALUES ($1, $2, $3)
     ON CONFLICT (campaign_id, user_id) DO NOTHING`,
    [campaign_id, user_id, role]
  );

  return NextResponse.json({
    content: [{ type: "text", text: JSON.stringify({ campaign_id, user_id, role }) }],
  });
}
