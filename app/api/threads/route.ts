import { auth } from "@/lib/auth";
import pool from "@/lib/db";

export async function GET() {
  const session = await auth();
  if (!session?.userId) return new Response("Unauthorized", { status: 401 });

  const { rows } = await pool.query<{
    id: string;
    title: string | null;
    status: string;
    created_at: string;
  }>(
    `SELECT id, title, status, created_at
     FROM app.chat_threads
     WHERE campaign_id = $1 AND user_id = $2
     ORDER BY updated_at DESC`,
    [session.campaignId, session.userId]
  );

  return Response.json({ threads: rows });
}

export async function POST() {
  const session = await auth();
  if (!session?.userId) return new Response("Unauthorized", { status: 401 });

  const { rows } = await pool.query<{ id: string }>(
    `INSERT INTO app.chat_threads (campaign_id, user_id)
     VALUES ($1, $2)
     RETURNING id`,
    [session.campaignId, session.userId]
  );

  return Response.json({ id: rows[0].id, status: "regular", title: null }, { status: 201 });
}
