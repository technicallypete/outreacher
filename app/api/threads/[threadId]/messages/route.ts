import { auth } from "@/lib/auth";
import pool from "@/lib/db";

type Params = { params: Promise<{ threadId: string }> };

// Messages are stored in "ai-sdk/v6" encoded format.
// The decode fn in aiSDKV6FormatAdapter reads: stored.id, stored.parent_id, stored.content
type StoredMessage = {
  id: string;
  parent_id: string | null;
  format: string;
  content: Record<string, unknown>;
};

// Body sent by the client: encoded message + metadata
type AppendBody = {
  id: string;
  parentId: string | null;
  content: Record<string, unknown>;
};

export async function GET(_req: Request, { params }: Params) {
  const session = await auth();
  if (!session?.userId) return new Response("Unauthorized", { status: 401 });

  const { threadId } = await params;

  const { rows: threadRows } = await pool.query<{ head_id: string | null }>(
    `SELECT head_id FROM app.chat_threads
     WHERE id = $1 AND campaign_id = $2 AND user_id = $3`,
    [threadId, session.campaignId!, session.userId]
  );
  if (!threadRows[0]) return new Response("Not found", { status: 404 });

  const { rows } = await pool.query<StoredMessage>(
    `SELECT id, parent_id, 'ai-sdk/v6' AS format, content
     FROM app.chat_messages
     WHERE thread_id = $1
     ORDER BY created_at ASC`,
    [threadId]
  );

  return Response.json({
    headId: threadRows[0].head_id,
    messages: rows,
  });
}

export async function POST(req: Request, { params }: Params) {
  const session = await auth();
  if (!session?.userId) return new Response("Unauthorized", { status: 401 });

  const { threadId } = await params;

  const { rows: threadRows } = await pool.query(
    `SELECT id FROM app.chat_threads
     WHERE id = $1 AND campaign_id = $2 AND user_id = $3`,
    [threadId, session.campaignId!, session.userId]
  );
  if (!threadRows[0]) return new Response("Not found", { status: 404 });

  const { id, parentId, content } = await req.json() as AppendBody;

  await pool.query(
    `INSERT INTO app.chat_messages (id, thread_id, parent_id, content)
     VALUES ($1, $2, $3, $4)
     ON CONFLICT (thread_id, id) DO UPDATE SET content = EXCLUDED.content`,
    [id, threadId, parentId ?? null, JSON.stringify(content)]
  );

  await pool.query(
    `UPDATE app.chat_threads SET head_id = $1, updated_at = NOW() WHERE id = $2`,
    [id, threadId]
  );

  return new Response(null, { status: 204 });
}
