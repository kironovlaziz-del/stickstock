"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { createClient } from "@/lib/supabase/client";

const SUPPORTED_LOCALES = ["en"];

export default function SignupPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);

    const browserLocale = typeof navigator !== "undefined" ? navigator.language.slice(0, 2) : "en";
    const locale = SUPPORTED_LOCALES.includes(browserLocale) ? browserLocale : "en";

    const supabase = createClient();
    const { data, error } = await supabase.auth.signUp({
      email,
      password,
      options: {
        data: { locale },
        emailRedirectTo: `${window.location.origin}/auth/callback`,
      },
    });
    setLoading(false);
    if (error) {
      setError(error.message);
      return;
    }
    if (data.session) {
      router.push("/dashboards");
      router.refresh();
    } else {
      setNotice("Check your email to confirm your account before logging in.");
    }
  }

  async function handleOAuth(provider: "google" | "github") {
    const supabase = createClient();
    await supabase.auth.signInWithOAuth({
      provider,
      options: { redirectTo: `${window.location.origin}/auth/callback` },
    });
  }

  return (
    <main className="flex min-h-screen items-center justify-center px-4">
      <div className="glass w-full max-w-sm rounded-2xl p-8">
        <div className="mb-8 flex items-center gap-2">
          <div className="h-8 w-8 rounded-lg bg-accent-gradient" />
          <span className="text-lg font-bold tracking-tight">StickStock</span>
        </div>

        <h1 className="mb-1 text-2xl font-extrabold tracking-tight">Create an account</h1>
        <p className="mb-6 text-sm text-slate-400">Free, self-hosted, yours.</p>

        {error && (
          <div className="mb-4 rounded-lg border border-down/30 bg-down/10 px-3 py-2 text-sm text-down">
            {error}
          </div>
        )}
        {notice && (
          <div className="mb-4 rounded-lg border border-up/30 bg-up/10 px-3 py-2 text-sm text-up">
            {notice}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="email" className="mb-1 block text-xs font-medium text-slate-400">
              Email
            </label>
            <input
              id="email"
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
            />
          </div>
          <div>
            <label htmlFor="password" className="mb-1 block text-xs font-medium text-slate-400">
              Password
            </label>
            <input
              id="password"
              type="password"
              required
              minLength={6}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="focus-ring w-full rounded-lg bg-accent-gradient px-3 py-2 text-sm font-semibold text-white disabled:opacity-50"
          >
            {loading ? "Creating account..." : "Sign up"}
          </button>
        </form>

        <div className="my-6 flex items-center gap-3 text-xs text-slate-500">
          <div className="h-px flex-1 bg-white/10" />
          or
          <div className="h-px flex-1 bg-white/10" />
        </div>

        <div className="space-y-2">
          <button
            onClick={() => handleOAuth("google")}
            className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm font-medium hover:bg-base-800"
          >
            Continue with Google
          </button>
          <button
            onClick={() => handleOAuth("github")}
            className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm font-medium hover:bg-base-800"
          >
            Continue with GitHub
          </button>
        </div>

        <p className="mt-6 text-center text-sm text-slate-400">
          Already have an account?{" "}
          <Link href="/login" className="text-accent-blue hover:underline">
            Log in
          </Link>
        </p>
      </div>
    </main>
  );
}
