import Link from "next/link";
import { auth } from "@/lib/auth";

export default async function Home() {
  const session = await auth();

  return (
    <div className="min-h-screen flex flex-col">
      <header className="border-b border-gray-200 bg-white">
        <div className="max-w-5xl mx-auto px-6 h-14 flex items-center justify-between">
          <span className="font-semibold tracking-tight">Outreacher</span>
          <nav>
            {session ? (
              <Link
                href="/chat"
                className="text-sm font-medium bg-blue-600 text-white px-4 py-1.5 rounded-md hover:bg-blue-700 transition-colors"
              >
                Go to app
              </Link>
            ) : (
              <Link
                href="/sign-in"
                className="text-sm font-medium bg-blue-600 text-white px-4 py-1.5 rounded-md hover:bg-blue-700 transition-colors"
              >
                Sign in
              </Link>
            )}
          </nav>
        </div>
      </header>

      <main className="flex-1 flex items-center justify-center">
        <div className="text-center max-w-md px-6">
          <h1 className="text-3xl font-semibold mb-3">AI-powered outreach</h1>
          <p className="text-gray-500 mb-6">
            Manage leads, import contacts, and run outreach campaigns — all through a natural language interface.
          </p>
          {!session && (
            <Link
              href="/sign-in"
              className="inline-block bg-blue-600 text-white text-sm font-medium px-5 py-2 rounded-md hover:bg-blue-700 transition-colors"
            >
              Get started
            </Link>
          )}
        </div>
      </main>
    </div>
  );
}
