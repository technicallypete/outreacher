import { callTool } from "@/lib/mcp";
import { auth } from "@/lib/auth";
import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";

const ImportJsonSchema = z.object({
  csv: z.string().min(1),
  campaign_id: z.number().int().positive().optional(),
});

export async function POST(req: NextRequest) {
  const session = await auth();
  const ct = req.headers.get("content-type") ?? "";

  let csv: string;
  // Session campaign takes precedence; explicit param allows API/test access.
  let campaignId: number = session?.campaignId ?? 0;

  if (ct.includes("multipart/form-data")) {
    const form = await req.formData();
    const file = form.get("file");
    if (!file || typeof file === "string") {
      return NextResponse.json({ error: "file is required" }, { status: 400 });
    }
    csv = await file.text();
    const c = form.get("campaign_id");
    if (c && typeof c === "string") campaignId = Number(c); // explicit override
  } else if (ct.includes("application/json")) {
    const parsed = ImportJsonSchema.safeParse(await req.json());
    if (!parsed.success) {
      return NextResponse.json({ error: parsed.error.issues }, { status: 400 });
    }
    csv = parsed.data.csv;
    if (parsed.data.campaign_id) campaignId = parsed.data.campaign_id; // explicit override
  } else {
    return NextResponse.json({ error: "unsupported content-type" }, { status: 415 });
  }

  const args: Record<string, unknown> = { csv, campaign_id: campaignId };

  const result = await callTool("import_leads", args);
  return NextResponse.json(result);
}
