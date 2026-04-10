"use client";

import { useState } from "react";

export default function OnboardingForm() {
  const [orgName, setOrgName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError("");

    const res = await fetch("/api/onboarding", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ orgName }),
    });

    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      setError(data.error ?? "Something went wrong. Please try again.");
      setLoading(false);
      return;
    }

    // Full navigation to /chat. The chat page calls auth() server-side,
    // which auto-looks up campaignId from DB and redirects to onboarding if missing.
    window.location.href = "/chat";
  }

  return (
    <div className="max-w-md w-full mx-auto p-8">
      <h1 className="text-2xl font-semibold mb-1">Set up your workspace</h1>
      <p className="text-gray-500 mb-6 text-sm">
        Give your organization a name. You can always change it later.
      </p>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="orgName" className="block text-sm font-medium mb-1">
            Organization name
          </label>
          <input
            id="orgName"
            type="text"
            required
            value={orgName}
            onChange={(e) => setOrgName(e.target.value)}
            placeholder="Acme Corp"
            className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        {error && <p className="text-sm text-red-600">{error}</p>}

        <button
          type="submit"
          disabled={loading || !orgName.trim()}
          className="w-full bg-blue-600 text-white rounded-md px-4 py-2 text-sm font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
        >
          {loading ? "Creating…" : "Create workspace"}
        </button>
      </form>
    </div>
  );
}
