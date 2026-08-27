"use client";
import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import Image from "next/image";

export default function SignupPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSuccess(false);
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
      if (data.token) {
        localStorage.setItem("ss_token", data.token);
        setSuccess(true);
        // Ждём 1.5 секунды, показывая сообщение, затем редирект
        setTimeout(() => {
          router.push("/dashboards");
        }, 1500);
      } else {
        setError("Unexpected response from server.");
        setLoading(false);
      }
    } catch (err) {
      console.error("Signup error:", err);
      setError("Network error. Please try again.");
      setLoading(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center px-4">
      <div className="glass w-full max-w-sm rounded-2xl p-8">
        {/* Logo */}
        <div className="mb-6 flex items-center justify-center">
          <Image
            src="/logotip.png"
            alt="StickStock"
            width={150}
            height={40}
            priority
            style={{ objectFit: "contain" }}
          />
        </div>
        <h1 className="mb-1 text-2xl font-extrabold tracking-tight text-center">Create account</h1>
        <p className="mb-6 text-sm text-slate-400 text-center">
          Get started with StickStock. All fields are required.
        </p>
        {error && (
          <div className="mb-4 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">
            {error}
          </div>
        )}
        {success && (
          <div className="mb-4 rounded-lg border border-green-500/30 bg-green-500/10 px-3 py-2 text-sm text-green-400">
            ✅ Account created! Redirecting to dashboard...
          </div>
        )}
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-400">Username</label>
            <input
              type="text"
              required
              minLength={3}
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="e.g., myuser"
              className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
            />
            <p className="text-xs text-slate-500 mt-1">At least 3 characters.</p>
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-400">Password</label>
            <input
              type="password"
              required
              minLength={6}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
            />
            <p className="text-xs text-slate-500 mt-1">Minimum 6 characters.</p>
          </div>
          <button
            type="submit"
            disabled={loading || success}
            className="focus-ring w-full rounded-lg bg-accent-gradient px-3 py-2 text-sm font-semibold text-white disabled:opacity-50"
          >
            {loading ? "Creating..." : success ? "Done!" : "Sign up"}
          </button>
        </form>
        <p className="mt-4 text-center text-sm text-slate-400">
          Have an account?{" "}
          <Link href="/login" className="text-accent-blue hover:underline">
            Log in
          </Link>
        </p>
      </div>
    </main>
  );
}
