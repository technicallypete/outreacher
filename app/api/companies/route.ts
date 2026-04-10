import { callTool } from "@/lib/mcp";
import { auth } from "@/lib/auth";
import { NextRequest, NextResponse } from "next/server";

export async function GET(req: NextRequest) {
  const session = await auth();
  const { searchParams } = req.nextUrl;

  // Session campaign takes precedence; explicit query param allows API/test access.
  const campaignId = session?.campaignId ?? Number(searchParams.get("campaign_id") ?? 0);

  const args: Record<string, unknown> = { campaign_id: campaignId };
  const query = searchParams.get("query");
  const industry = searchParams.get("industry");
  const fundingStage = searchParams.get("funding_stage");
  const isVc = searchParams.get("is_vc");
  const isHiring = searchParams.get("is_hiring");

  if (query) args.query = query;
  if (industry) args.industry = industry;
  if (fundingStage) args.funding_stage = fundingStage;
  if (isVc) args.is_vc = isVc;
  if (isHiring) args.is_hiring = isHiring;

  const result = await callTool("search_companies", args);
  return NextResponse.json(result);
}
