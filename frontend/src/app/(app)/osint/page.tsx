"use client";

import { useState } from "react";
import { api } from "@/lib/api";

type LookupType = "whois" | "dns" | "breach";

const TYPE_LABELS: Record<LookupType, string> = {
  whois: "WHOIS (domain)",
  dns: "DNS records (domain)",
  breach: "Breach check (email)",
};

export default function OSINTPage() {
  const [type, setType] = useState<LookupType>("whois");
  const [target, setTarget] = useState("");
  const [result, setResult] = useState<Record<string, unknown> | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleLookup(e: React.FormEvent) {
    e.preventDefault();
    if (!target) return;
    setLoading(true);
    setError(null);
    setResult(null);
    try {
      setResult(await api.osintLookup({ type, target }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="mx-auto max-w-3xl">
      <h1 className="mb-2 text-2xl font-extrabold tracking-tight">OSINT lookup</h1>
      <p className="mb-6 text-sm text-slate-400">
        WHOIS and DNS are public registry data. Breach check uses HaveIBeenPwned — for checking
        your own exposure, not anyone else&apos;s data.
      </p>

      <form onSubmit={handleLookup} className="glass mb-6 space-y-4 rounded-2xl p-5">
        <div className="flex gap-2">
          {(Object.keys(TYPE_LABELS) as LookupType[]).map((t) => (
            <button
              key={t}
              type="button"
              onClick={() => setType(t)}
              className={`focus-ring rounded-lg border px-3 py-2 text-xs font-medium ${
                type === t
                  ? "border-accent-blue bg-accent-blue/20 text-white"
                  : "border-white/10 text-slate-400 hover:bg-white/5"
              }`}
            >
              {TYPE_LABELS[t]}
            </button>
          ))}
        </div>

        <input
          value={target}
          onChange={(e) => setTarget(e.target.value)}
          placeholder={type === "breach" ? "email@example.com" : "example.com"}
          className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
        />

        <button
          type="submit"
          disabled={loading}
          className="focus-ring rounded-lg bg-accent-gradient px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
        >
          {loading ? "Looking up..." : "Look up"}
        </button>
      </form>

      {error && (
        <div className="mb-4 rounded-lg border border-down/30 bg-down/10 px-3 py-2 text-sm text-down">{error}</div>
      )}

      {result && (
        <div className="glass rounded-2xl p-5">
          {typeof result.raw === "string" ? (
            <pre className="max-h-96 overflow-auto whitespace-pre-wrap font-mono text-xs text-slate-300">
              {result.raw}
            </pre>
          ) : (
            <pre className="max-h-96 overflow-auto whitespace-pre-wrap font-mono text-xs text-slate-300">
              {JSON.stringify(result, null, 2)}
            </pre>
          )}
        </div>
      )}
    </div>
  );
}
