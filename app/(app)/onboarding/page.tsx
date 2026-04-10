import { redirect } from "next/navigation";
import { auth } from "@/lib/auth";
import OnboardingForm from "./OnboardingForm";

export default async function OnboardingPage() {
  const session = await auth();

  // auth() runs the full JWT callback which auto-looks up campaignId from DB.
  // If found, the user already onboarded — skip to chat.
  if (session?.campaignId) {
    redirect("/chat");
  }

  return (
    <div className="flex items-center justify-center flex-1">
      <OnboardingForm />
    </div>
  );
}
