import { callTool } from "@/lib/mcp";
import { auth } from "@/lib/auth";
import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";

const jsonSchema = z.object({
  format: z.string(),
  csv: z.string(),
  campaign_id: z.number().optional(),
});

export async function POST(req: NextRequest) {
  const session = await auth();
  const ct = req.headers.get("content-type") ?? "";

  let format: string;
  let csv: string;
  // Session campaign takes precedence; explicit param allows API/test access.
  const args: Record<string, unknown> = {
    campaign_id: session?.campaignId ?? 0,
  };

  if (ct.includes("multipart/form-data")) {
    const form = await req.formData();
    const file = form.get("file");
    const fmt = form.get("format");
    if (!file || typeof fmt !== "string" || !fmt) {
      return NextResponse.json(
        { error: "multipart upload requires 'file' and 'format' fields" },
        { status: 400 }
      );
    }
    format = fmt;
    csv =
      file instanceof File
        ? await file.text()
        : String(file);
    const campaignId = form.get("campaign_id");
    if (campaignId) args.campaign_id = Number(campaignId); // explicit override
  } else if (ct.includes("application/json")) {
    const body = await req.json().catch(() => null);
    const parsed = jsonSchema.safeParse(body);
    if (!parsed.success) {
      return NextResponse.json({ error: parsed.error.flatten() }, { status: 400 });
    }
    format = parsed.data.format;
    csv = parsed.data.csv;
    if (parsed.data.campaign_id) args.campaign_id = parsed.data.campaign_id; // explicit override
  } else {
    return NextResponse.json({ error: "unsupported content-type" }, { status: 415 });
  }

  args.format = format;
  args.csv = csv;

  const result = await callTool("import_csv", args);
  return NextResponse.json(result);
}
