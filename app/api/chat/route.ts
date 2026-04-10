import { streamText, convertToModelMessages, jsonSchema, stepCountIs, createUIMessageStream, createUIMessageStreamResponse } from "ai";
import { createOpenAI } from "@ai-sdk/openai";
import { createAnthropic } from "@ai-sdk/anthropic";
import { auth } from "@/lib/auth";
import { callTool } from "@/lib/mcp";

function resolveChatModel() {
  const provider = (process.env.CHAT_LLM_PROVIDER ?? "anthropic").toLowerCase();
  const model = process.env.CHAT_LLM_MODEL;
  if (provider === "openai") {
    return createOpenAI({ apiKey: process.env.OPENAI_API_KEY })(model ?? "gpt-4o-mini");
  }
  return createAnthropic({ apiKey: process.env.ANTHROPIC_API_KEY })(model ?? "claude-sonnet-4-6");
}

export const maxDuration = 60;

async function safeMcpCall(name: string, args: Record<string, unknown>, campaignId: number) {
  try {
    const result = await callTool(name, { ...args, campaign_id: campaignId });
    if (result.isError) return JSON.stringify({ error: result.content[0]?.text ?? "tool error" });
    return result.content[0]?.text ?? "{}";
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    console.error(`[chat] tool ${name} failed:`, msg);
    return JSON.stringify({ error: msg });
  }
}

const STATUS_VALUES = ["new", "contacted", "qualified", "disqualified", "converted"] as const;

const ATTACHMENT_RE = /<attachment name="?([^">]+)"?>([\s\S]*?)<\/attachment>/g;

function trimAttachmentHistory(uiMessages: unknown[]): unknown[] {
  type MsgPart = { type: string; text?: string };
  type Msg = { role: string; parts?: MsgPart[]; content?: MsgPart[] };
  const msgs = uiMessages as Msg[];

  // Find the index of the last user message — keep its attachments intact.
  let lastUserIdx = -1;
  for (let i = msgs.length - 1; i >= 0; i--) {
    if (msgs[i].role === "user") { lastUserIdx = i; break; }
  }

  return msgs.map((msg, idx) => {
    if (msg.role !== "user" || idx === lastUserIdx) return msg;
    const parts = (msg.parts ?? msg.content ?? []).map((part) => {
      if (part.type !== "text" || !part.text) return part;
      const stripped = part.text.replace(
        ATTACHMENT_RE,
        (_full, name: string) => `[attachment: ${name}]`,
      );
      return stripped === part.text ? part : { ...part, text: stripped };
    });
    return msg.parts
      ? { ...msg, parts }
      : { ...msg, content: parts };
  });
}

// CSV_RE matches a single CSV attachment (non-global).
const CSV_RE = /<attachment name="?([^">]+\.csv)"?>([\s\S]*?)<\/attachment>/i;

// Detects row-selection intent in the user's message (e.g. "first 10", "next 10", "rows 30-40", "last 5").
const ROW_HINT_RE = /\b(first|next|following|last|top|bottom|only)\s+\d+|\brows?\s+\d+|\d+\s*[-–]\s*\d+/i;

/**
 * Extracts any CSV attachment from the last user message and strips its body.
 *
 * Fast path (no row-selection hint): imports directly via MCP, returns importResult.
 * LLM path (row-selection hint present): returns extractedCsv so the LLM can
 * decide which rows to import via the import_csv tool.
 *
 * In both cases the raw CSV body is replaced with a lightweight placeholder
 * so the LLM never sees the full file content.
 */
async function preImportCSV(
  uiMessages: unknown[],
  campaignId: number,
): Promise<{ messages: unknown[]; importResult: string | null; extractedCsv: string | null }> {
  type MsgPart = { type: string; text?: string };
  type Msg = { role: string; parts?: MsgPart[]; content?: MsgPart[] };

  const msgs = uiMessages as Msg[];
  const lastIdx = msgs.length - 1;
  const lastMsg = msgs[lastIdx];
  if (!lastMsg || lastMsg.role !== "user") return { messages: uiMessages, importResult: null, extractedCsv: null };

  const parts = lastMsg.parts ?? lastMsg.content ?? [];
  let csvContent: string | null = null;
  let csvName = "";
  let userText = "";

  for (const part of parts) {
    if (part.type !== "text" || !part.text) continue;
    const m = part.text.match(CSV_RE);
    if (m) {
      csvName = m[1];
      csvContent = m[2].trim();
    } else {
      userText += part.text + " ";
    }
  }

  if (!csvContent) return { messages: uiMessages, importResult: null, extractedCsv: null };

  const rowCount = csvContent.split("\n").filter(Boolean).length - 1;

  // Replace the raw CSV body with a placeholder in the message sent to the LLM.
  const newParts = parts.map((part) => {
    if (part.type !== "text" || !part.text) return part;
    const stripped = part.text.replace(CSV_RE, `[CSV: ${csvName} — ${rowCount} rows]`);
    return stripped !== part.text ? { ...part, text: stripped } : part;
  });

  const newMsgs = [...msgs];
  newMsgs[lastIdx] = lastMsg.parts
    ? { ...lastMsg, parts: newParts }
    : { ...lastMsg, content: newParts };

  // If the user specified a row selection, let the LLM interpret it.
  if (ROW_HINT_RE.test(userText)) {
    return { messages: newMsgs, importResult: null, extractedCsv: csvContent };
  }

  // Fast path: import everything directly, no LLM round-trip.
  const importResult = await safeMcpCall("import_csv", { csv: csvContent }, campaignId);
  return { messages: newMsgs, importResult, extractedCsv: null };
}

/** Slice a CSV string to the specified rows. Accepts "N-M" (1-based, inclusive) or a plain count. */
function sliceCsv(csv: string, rows: string): string {
  const lines = csv.split("\n");
  const header = lines[0];
  const data = lines.slice(1).filter(Boolean);
  const rangeMatch = rows.match(/(\d+)\s*[-–]\s*(\d+)/);
  if (rangeMatch) {
    const start = parseInt(rangeMatch[1], 10) - 1;
    const end = parseInt(rangeMatch[2], 10);
    return [header, ...data.slice(start, end)].join("\n");
  }
  const countMatch = rows.match(/(\d+)/);
  if (countMatch) {
    const count = parseInt(countMatch[1], 10);
    return [header, ...data.slice(0, count)].join("\n");
  }
  return csv;
}

type ImportedRow = { id: number; name: string; company?: string; action: string };

/** Format an import result JSON string into a human-readable markdown summary. */
function formatImportSummary(resultJson: string): string {
  try {
    const r = JSON.parse(resultJson) as {
      companies?: number; leads?: number; skipped?: number; error?: string;
      rows?: ImportedRow[];
    };
    if (r.error) return `Import failed: ${r.error}`;
    const leads = r.leads ?? 0;
    const companies = r.companies ?? 0;
    const skipped = r.skipped ?? 0;
    const rows = r.rows ?? [];

    const lines: string[] = [];

    if (rows.length > 0) {
      const created = rows.filter(r => r.action === "created");
      const updated = rows.filter(r => r.action === "updated");
      if (created.length > 0) {
        lines.push(`**${created.length} new lead${created.length !== 1 ? "s" : ""} added:**`);
        for (const row of created) {
          lines.push(`- ${row.name}${row.company ? ` (${row.company})` : ""}`);
        }
      }
      if (updated.length > 0) {
        lines.push(`**${updated.length} existing lead${updated.length !== 1 ? "s" : ""} updated:**`);
        for (const row of updated) {
          lines.push(`- ${row.name}${row.company ? ` (${row.company})` : ""}`);
        }
      }
    } else {
      if (leads > 0) lines.push(`**${leads}** new lead${leads !== 1 ? "s" : ""} added`);
      if (companies > 0) lines.push(`**${companies}** compan${companies !== 1 ? "ies" : "y"} processed`);
    }

    if (skipped > 0) lines.push(`${skipped} row${skipped !== 1 ? "s" : ""} skipped (no email or LinkedIn URL)`);
    if (lines.length === 0) return "Import complete — all rows already existed (no changes made).";
    return lines.join("\n");
  } catch {
    return `Import result: ${resultJson}`;
  }
}

/**
 * Stream a plain text message using the AI SDK UI message stream protocol,
 * bypassing any LLM call. Compatible with useChat / AssistantChatTransport.
 */
function importResultResponse(text: string): Response {
  const stream = createUIMessageStream({
    execute({ writer }) {
      const id = "text-0";
      writer.write({ type: "text-start", id });
      writer.write({ type: "text-delta", id, delta: text });
      writer.write({ type: "text-end", id });
    },
  });
  return createUIMessageStreamResponse({ stream });
}

export async function POST(req: Request) {
  const session = await auth();
  if (!session?.userId) {
    return new Response("Unauthorized", { status: 401 });
  }

  const campaignId = session.campaignId ?? 0;

  let body: string;
  try {
    body = await req.text();
  } catch {
    return new Response(JSON.stringify({ error: "Request body too large" }), {
      status: 413,
      headers: { "Content-Type": "application/json" },
    });
  }

  let parsed: { messages?: unknown };
  try {
    parsed = JSON.parse(body);
  } catch {
    return new Response(JSON.stringify({ error: "Invalid JSON — request body may be too large" }), {
      status: 400,
      headers: { "Content-Type": "application/json" },
    });
  }

  const { messages: rawUiMessages } = parsed;

  // Strip <attachment> content from all messages except the last user message.
  const trimmed = trimAttachmentHistory(Array.isArray(rawUiMessages) ? rawUiMessages : []);

  // Pre-import any CSV in the current message directly (no LLM round trip).
  const { messages: uiMessages, importResult, extractedCsv } = await preImportCSV(trimmed, campaignId);

  // Limit history to prevent context overflow as conversations grow.
  const MAX_HISTORY = 20;
  const limitedMessages = uiMessages.length > MAX_HISTORY ? uiMessages.slice(-MAX_HISTORY) : uiMessages;

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const messages = await convertToModelMessages(limitedMessages as any);

  const allTools = {
    search_leads: {
      description: "Search leads by name, email, status, or company. All filters optional.",
      inputSchema: jsonSchema({
        type: "object",
        properties: {
          query: { type: "string", description: "Search by name or email" },
          status: { type: "string", enum: STATUS_VALUES, description: "Filter by status" },
          company: { type: "string", description: "Filter by company name" },
        },
      }),
      execute: async (args: { query?: string; status?: string; company?: string }) =>
        safeMcpCall("search_leads", args, campaignId),
    },

    get_lead: {
      description: "Get full details for a single lead, including notes.",
      inputSchema: jsonSchema({
        type: "object",
        properties: { id: { type: "number", description: "Lead ID" } },
        required: ["id"],
      }),
      execute: async (args: { id: number }) =>
        safeMcpCall("get_lead", args, campaignId),
    },

    update_lead_status: {
      description: "Update the status of a lead.",
      inputSchema: jsonSchema({
        type: "object",
        properties: {
          id: { type: "number", description: "Lead ID" },
          status: { type: "string", enum: STATUS_VALUES, description: "New status" },
        },
        required: ["id", "status"],
      }),
      execute: async (args: { id: number; status: string }) =>
        safeMcpCall("update_lead_status", args, campaignId),
    },

    create_followup_note: {
      description: "Append a follow-up note to a lead.",
      inputSchema: jsonSchema({
        type: "object",
        properties: {
          lead_id: { type: "number", description: "Lead ID" },
          content: { type: "string", description: "Note content" },
        },
        required: ["lead_id", "content"],
      }),
      execute: async (args: { lead_id: number; content: string }) =>
        safeMcpCall("create_followup_note", args, campaignId),
    },

    import_csv: {
      description: "Import CSV contact data. When the user attaches a CSV file it appears as [CSV: filename — N rows]. Call this tool with rows='1-N' or a plain count (e.g. '10') to import a subset, or omit rows to import everything. The csv parameter is ignored when a file was attached — pass any non-empty string.",
      inputSchema: jsonSchema({
        type: "object",
        properties: {
          csv: { type: "string", description: "Ignored when a CSV file was attached. Otherwise, raw CSV content." },
          rows: { type: "string", description: "Row selection: '1-10' (1-based inclusive range), '10' (first 10), or omit for all rows." },
          format: {
            type: "string",
            enum: ["gojiberry", "revli_startup_contacts", "revli_investor_contacts",
                   "revli_startup_companies", "revli_investor_companies"],
            description: "Source format — omit to auto-detect",
          },
        },
        required: ["csv"],
      }),
      execute: async (args: { csv: string; rows?: string; format?: string }) => {
        let csv = extractedCsv ?? args.csv;
        if (args.rows) csv = sliceCsv(csv, args.rows);
        return safeMcpCall("import_csv", { csv, format: args.format }, campaignId);
      },
    },

    search_companies: {
      description: "Search companies by name.",
      inputSchema: jsonSchema({
        type: "object",
        properties: { query: { type: "string", description: "Company name substring" } },
        required: ["query"],
      }),
      execute: async (args: { query: string }) =>
        safeMcpCall("search_companies", args, campaignId),
    },

    get_company: {
      description: "Get details and all leads for a company.",
      inputSchema: jsonSchema({
        type: "object",
        properties: { id: { type: "number", description: "Company ID" } },
        required: ["id"],
      }),
      execute: async (args: { id: number }) =>
        safeMcpCall("get_company", args, campaignId),
    },

  };

  if (importResult) {
    return importResultResponse(formatImportSummary(importResult));
  }

  const result = streamText({
    model: resolveChatModel(),
    system: `You are an outreach assistant. Help users manage leads, companies, and campaigns.
Current campaign ID: ${campaignId}.

After every tool call, always respond with a clear text summary of the results in markdown. Never end your turn without a response.

CSV files are shown as [CSV: filename — N rows]. Call import_csv immediately when you see one. Use the rows parameter to honour any row selection the user specified (e.g. "first 10" → rows="1-10", "rows 30-40" → rows="30-40"). Omit rows to import everything.`,
    messages,
    tools: allTools,
    stopWhen: stepCountIs(5),
  });

  return result.toUIMessageStreamResponse();
}
