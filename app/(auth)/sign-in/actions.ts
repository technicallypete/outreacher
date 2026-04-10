"use server";

import { signIn } from "@/lib/auth";
import { redirect } from "next/navigation";

export async function sendMagicLink(formData: FormData) {
  const email = formData.get("email") as string;
  console.log("[auth] sending magic link to:", email);

  try {
    await signIn("resend", { email, redirectTo: "/chat" });
  } catch (err: unknown) {
    // Next.js throws a special redirect error on success — rethrow it
    if (
      err instanceof Error &&
      (err.message === "NEXT_REDIRECT" ||
        ("digest" in err && String((err as { digest?: string }).digest).startsWith("NEXT_REDIRECT")))
    ) {
      throw err;
    }
    console.error("[auth] signIn error:", err);
    redirect("/sign-in?error=send_failed");
  }
}
