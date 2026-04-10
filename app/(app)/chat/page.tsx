import { redirect } from "next/navigation";
import { auth } from "@/lib/auth";
import ChatPanel from "@/components/ChatPanel";

export default async function ChatPage() {
  const session = await auth();

  if (!session?.campaignId) {
    redirect("/onboarding");
  }

  return (
    <div className="h-[calc(100vh-3.5rem)] flex justify-end overflow-hidden">
      <ChatPanel />
    </div>
  );
}
