import type { NextAuthConfig } from "next-auth";

// Edge-compatible auth config — no providers, no DB adapter, no Node.js-only imports.
// Used by middleware solely to verify JWTs without touching postgres.
export const authConfig: NextAuthConfig = {
  providers: [],
  pages: {
    signIn: "/sign-in",
    verifyRequest: "/sign-in/verify",
  },
  callbacks: {
    authorized({ auth }) {
      return !!auth;
    },
  },
};
