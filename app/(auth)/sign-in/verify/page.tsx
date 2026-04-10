import Link from "next/link";

export default function VerifyPage() {
  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="max-w-md w-full mx-auto p-8 text-center">
        <h1 className="text-2xl font-semibold mb-2">Check your email</h1>
        <p className="text-gray-500 mb-6">
          A sign-in link is on its way. Click it to continue — it expires in 24 hours.
        </p>
        <Link href="/sign-in" className="text-sm text-blue-600 hover:underline">
          Use a different email
        </Link>
      </div>
    </div>
  );
}
