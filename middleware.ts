import NextAuth from "next-auth";
import { authConfig } from "@/lib/auth.config";

const { auth } = NextAuth(authConfig);

export default auth((req) => {
  const { pathname } = req.nextUrl;

  // Static assets, auth routes, and the homepage are always public.
  if (
    pathname.startsWith("/api/") ||
    pathname.startsWith("/sign-in") ||
    pathname === "/"
  ) {
    return;
  }

  // Not logged in → sign in
  if (!req.auth) {
    return Response.redirect(new URL("/sign-in", req.url));
  }
});

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
