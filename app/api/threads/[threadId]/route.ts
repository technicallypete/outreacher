import { auth } from "@/lib/auth";
import pool from "@/lib/db";

type Params = { params: Promise<{ threadId: string }> };

async function getThread(threadId: string, campaignId: number, userId: number) {
  const { rows } = await pool.query<{
    id: string;
    title: string | null;
    status: string;
    head_id: string | null;
  }>(
    `SELECT id, title, status, head_id
     FROM app.chat_threads
     WHERE id = $1 AND campaign_id = $2 AND user_id = $3`,
    [threadId, campaignId, userId]
  );
  return rows[0] ?? null;
}

export async function GET(_req: Request, { params }: Params) {
  const session = await auth();
  if (!session?.userId) return new Response("Unauthorized", { status: 401 });

  const { threadId } = await params;
  const thread = await getThread(threadId, session.campaignId!, session.userId);
  if (!thread) return new Response("Not found", { status: 404 });

  return Response.json({ id: thread.id, title: thread.title, status: thread.status });
}

export async function PATCH(req: Request, { params }: Params) {
  const session = await auth();
  if (!session?.userId) return new Response("Unauthorized", { status: 401 });

  const { threadId } = await params;
  const body = await req.json() as { title?: string; status?: string };

  const updates: string[] = ["updated_at = NOW()"];
  const values: unknown[] = [threadId, session.campaignId!, session.userId];

  if (body.title !== undefined) {
    values.push(body.title);
    updates.push(`title = $${values.length}`);
  }
  if (body.status !== undefined) {
    values.push(body.status);
    updates.push(`status = $${values.length}`);
  }

  await pool.query(
    `UPDATE app.chat_threads
     SET ${updates.join(", ")}
     WHERE id = $1 AND campaign_id = $2 AND user_id = $3`,
    values
  );

  return new Response(null, { status: 204 });
}

export async function DELETE(_req: Request, { params }: Params) {
  const session = await auth();
  if (!session?.userId) return new Response("Unauthorized", { status: 401 });

  const { threadId } = await params;
  await pool.query(
    `DELETE FROM app.chat_threads WHERE id = $1 AND campaign_id = $2 AND user_id = $3`,
    [threadId, session.campaignId!, session.userId]
  );

  return new Response(null, { status: 204 });
}
