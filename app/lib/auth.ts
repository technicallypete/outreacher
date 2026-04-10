import NextAuth from "next-auth";
import PostgresAdapter from "@auth/pg-adapter";
import Resend from "next-auth/providers/resend";
import { authConfig } from "./auth.config";
import pool from "./db";

// Full auth config — Node.js only (uses pg adapter).
// Import this only in server components, API routes, and server actions.
export const { handlers, auth, signIn, signOut, unstable_update } = NextAuth({
  ...authConfig,
  providers: [
    Resend({
      apiKey: process.env.RESEND_API_KEY,
      from: process.env.RESEND_FROM ?? "Outreacher <onboarding@resend.dev>",
    }),
  ],
  adapter: PostgresAdapter(pool),
  session: { strategy: "jwt" },
  callbacks: {
    async jwt({ token, user, trigger, session }) {
      // On first sign-in, link or create the app user
      if (user?.email) {
        const { rows } = await pool.query<{
          id: number;
          nextauth_id: string | null;
        }>(`SELECT id, nextauth_id FROM app.users WHERE email = $1 LIMIT 1`, [
          user.email,
        ]);
        if (rows.length > 0) {
          token.userId = rows[0].id;
          if (!rows[0].nextauth_id && user.id) {
            await pool.query(
              `UPDATE app.users SET nextauth_id = $1 WHERE id = $2`,
              [user.id, rows[0].id]
            );
          }
        } else {
          // Create app user on first sign-in
          const slug = user.email.split("@")[0].replace(/[^a-z0-9]/gi, "_");
          const { rows: created } = await pool.query<{ id: number }>(
            `INSERT INTO app.users (name, email, slug, nextauth_id)
             VALUES ($1, $2, $3, $4)
             ON CONFLICT (email) DO UPDATE SET nextauth_id = EXCLUDED.nextauth_id
             RETURNING id`,
            [user.name ?? slug, user.email, slug, user.id]
          );
          token.userId = created[0].id;
        }
      }

      // Session update from client (after onboarding sets campaignId/orgId)
      if (trigger === "update" && session) {
        const upd = session as { campaignId?: number; orgId?: number };
        if (upd.campaignId != null) token.campaignId = upd.campaignId;
        if (upd.orgId != null) token.orgId = upd.orgId;
      }

      // Auto-populate campaignId/orgId from DB if missing (e.g. after onboarding)
      if (token.userId && !token.campaignId) {
        const { rows } = await pool.query<{ campaign_id: number; org_id: number }>(
          `SELECT c.id AS campaign_id, c.organization_id AS org_id
           FROM app.campaigns c
           JOIN app.campaign_memberships cm ON cm.campaign_id = c.id
           WHERE cm.user_id = $1 AND c.is_default = true
           LIMIT 1`,
          [token.userId]
        );
        if (rows.length > 0) {
          token.campaignId = rows[0].campaign_id;
          token.orgId = rows[0].org_id;
        }
      }

      return token;
    },
    async session({ session, token }) {
      return {
        ...session,
        userId: token.userId as number | undefined,
        campaignId: token.campaignId as number | undefined,
        orgId: token.orgId as number | undefined,
      };
    },
  },
});

declare module "next-auth" {
  interface Session {
    userId?: number;
    campaignId?: number;
    orgId?: number;
  }
}
