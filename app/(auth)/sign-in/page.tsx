import { sendMagicLink } from "./actions";

export default function SignInPage() {
  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="max-w-md w-full mx-auto p-8">
        <h1 className="text-2xl font-semibold mb-1">Sign in to Outreacher</h1>
        <p className="text-gray-500 mb-6 text-sm">
          We&apos;ll send you a magic link — no password needed.
        </p>

        <form action={sendMagicLink} className="space-y-4">
          <div>
            <label htmlFor="email" className="block text-sm font-medium mb-1">
              Email address
            </label>
            <input
              id="email"
              name="email"
              type="email"
              required
              placeholder="you@example.com"
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <button
            type="submit"
            className="w-full bg-blue-600 text-white rounded-md px-4 py-2 text-sm font-medium hover:bg-blue-700 transition-colors"
          >
            Send sign-in link
          </button>
        </form>
      </div>
    </div>
  );
}
