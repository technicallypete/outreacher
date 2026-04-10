import { callTool } from "@/lib/mcp";
import { NextRequest, NextResponse } from "next/server";

export async function GET(
  req: NextRequest,
  { params }: { params: Promise<{ companyId: string }> }
) {
  const { companyId } = await params;
  const id = Number(companyId);
  if (!id) return NextResponse.json({ error: "invalid company id" }, { status: 400 });

  const args: Record<string, unknown> = { id };
  const campaignId = req.nextUrl.searchParams.get("campaign_id");
  if (campaignId) args.campaign_id = Number(campaignId);

  const result = await callTool("get_company", args);
  return NextResponse.json(result);
}
