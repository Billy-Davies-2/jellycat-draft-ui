"use client";

import { useEffect, useState } from "react";

export default function LoginPage({ searchParams }: { searchParams?: { [k: string]: string | string[] | undefined } }) {
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const callbackURLParam = typeof searchParams?.callbackURL === "string" ? searchParams?.callbackURL : undefined;
    const callbackURL = callbackURLParam || "/";

    async function start() {
      try {
        const res = await fetch("/api/auth/sign-in/social", {
          method: "POST",
          headers: { "content-type": "application/json" },
          body: JSON.stringify({
            provider: "authentik",
            callbackURL,
          }),
        });
        if (!res.ok) {
          const text = await res.text();
          throw new Error(text || `Sign-in failed (${res.status})`);
        }
        const data = await res.json();
        if (data.redirect && data.url) {
          window.location.href = data.url as string;
          return;
        }
        if (data.token) {
          // already signed-in via token flow; go back
          window.location.href = callbackURL;
          return;
        }
        throw new Error("Unexpected sign-in response");
      } catch (e: any) {
        setError(e?.message || "Unable to start login");
      }
    }

    start();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="min-h-screen flex items-center justify-center p-6">
      <div className="w-full max-w-md rounded-lg border p-6 bg-white">
        <h1 className="text-xl font-semibold mb-2">Redirecting to Authentik…</h1>
        <p className="text-sm text-gray-600">Starting OAuth flow with your identity provider.</p>
        {error && (
          <p className="mt-3 text-sm text-red-600">{error}</p>
        )}
      </div>
    </div>
  );
}
