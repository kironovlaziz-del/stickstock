"use client";
import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";

export default function SignupPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const res = await fetch("/api/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      });
      const data = await res.json();
      if (!res.ok) {
        setError(data.error || "Registration failed");
        setLoading(false);
        return;
      }
      // ✅ Сохраняем токен
      if (data.token) {
        localStorage.setItem("ss_token", data.token);
      }
      router.push("/dashboards");
    } catch (err) {
      console.error("Signup error:", err);
      setError("Network error");
      setLoading(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center px-4">
      <div className="glass w-full max-w-sm rounded-2xl p-8">
        <div className="mb-8 flex items-center gap-2">
          <div className="h-8 w-8 rounded-lg bg-accent-gradient" />
          <span className="text-lg font-bold tracking-tight">StickStock</span>
        </div>
        <h1 className="mb-1 text-2xl font-extrabold tracking-tight">Create account</h1>
        <p className="mb-6 text-sm text-slate-400">No email required. Just username and password.</p>
        {error && <div className="mb-4 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">{error}</div>}
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-400">Username</label>
            <input type="text" required value={username} onChange={e => setUsername(e.target.value)}
              className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none" />
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-400">Password (min 6 chars)</label>
            <input type="password" required minLength={6} value={password} onChange={e => setPassword(e.target.value)}
              className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none" />
          </div>
          <button type="submit" disabled={loading}
            className="focus-ring w-full rounded-lg bg-accent-gradient px-3 py-2 text-sm font-semibold text-white disabled:opacity-50">
            {loading ? "Creating..." : "Sign up"}
          </button>
        </form>
        <p className="mt-4 text-center text-sm text-slate-400">
          Have account? <Link href="/login" className="text-accent-blue hover:underline">Log in</Link>
        </p>
      </div>
    </main>
  );
}