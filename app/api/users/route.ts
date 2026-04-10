import pool from "@/lib/db";
import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";

const CreateUserSchema = z.object({
  name: z.string().min(1),
  slug: z.string().min(1),
  email: z.string().email().optional(),
});

export async function POST(req: NextRequest) {
  const parsed = CreateUserSchema.safeParse(await req.json());
  if (!parsed.success) {
    return NextResponse.json({ error: parsed.error.issues }, { status: 400 });
  }

  const { name, slug, email } = parsed.data;

  const { rows: [user] } = await pool.query(
    `INSERT INTO app.users (name, slug, email, is_system) VALUES ($1, $2, $3, false) RETURNING id, name, slug, email, is_system`,
    [name, slug, email ?? null]
  );

  return NextResponse.json(
    { content: [{ type: "text", text: JSON.stringify(user) }] },
    { status: 201 }
  );
}
