"use client";
import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation"; // добавьте импорт useRouter

export default function LoginPage() {
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
      const res = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
        // credentials: "include", // не нужно для Bearer-токена
      });
      const data = await res.json();
      if (!res.ok) { 
        setError(data.error || "Login failed");
        setLoading(false);
        return;
      }
      // Сохраняем токен
      if (data.token) {
        localStorage.setItem("ss_token", data.token);
      }
      const params = new URLSearchParams(window.location.search);
      const redirect = params.get("redirectedFrom") ?? "/dashboards";
      // используем router.push для программного перехода
      router.push(redirect);
    } catch (err) {
      console.error("Login error:", err);
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
        <h1 className="mb-1 text-2xl font-extrabold tracking-tight">Log in</h1>
        <p className="mb-6 text-sm text-slate-400">Your data, queried and visualized.</p>
        {error && <div className="mb-4 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">{error}</div>}
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-400">Username</label>
            <input type="text" required value={username} onChange={e => setUsername(e.target.value)}
              className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none" />
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-400">Password</label>
            <input type="password" required value={password} onChange={e => setPassword(e.target.value)}
              className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none" />
          </div>
          <button type="submit" disabled={loading}
            className="focus-ring w-full rounded-lg bg-accent-gradient px-3 py-2 text-sm font-semibold text-white disabled:opacity-50">
            {loading ? "Logging in..." : "Log in"}
          </button>
        </form>
        <p className="mt-4 text-center text-sm text-slate-400">
          No account? <Link href="/signup" className="text-accent-blue hover:underline">Sign up</Link>
        </p>
        <p className="mt-2 text-center text-sm text-slate-400">
        </p>
      </div>
    </main>
  );
}