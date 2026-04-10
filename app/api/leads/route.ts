import { callTool } from "@/lib/mcp";
import { auth } from "@/lib/auth";
import { NextRequest, NextResponse } from "next/server";

export async function GET(req: NextRequest) {
  const session = await auth();
  const { searchParams } = req.nextUrl;

  // Session campaign takes precedence; explicit query param allows API/test access.
  const campaignId = session?.campaignId ?? Number(searchParams.get("campaign_id") ?? 0);

  const args: Record<string, unknown> = { campaign_id: campaignId };
  const status = searchParams.get("status");
  const query = searchParams.get("query");
  const company = searchParams.get("company");

  if (status) args.status = status;
  if (query) args.query = query;
  if (company) args.company = company;

  const result = await callTool("search_leads", args);
  return NextResponse.json(result);
}
