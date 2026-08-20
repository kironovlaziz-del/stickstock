"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { ScheduledReport, SavedQuery } from "@/lib/types";

const CRON_PRESETS: { label: string; value: string }[] = [
  { label: "Every day at 9:00", value: "0 9 * * *" },
  { label: "Every Monday at 9:00", value: "0 9 * * 1" },
  { label: "1st of the month at 9:00", value: "0 9 1 * *" },
  { label: "Every hour", value: "0 * * * *" },
  { label: "Custom...", value: "" },
];

export default function ReportsPage() {
  const [reports, setReports] = useState<ScheduledReport[]>([]);
  const [queries, setQueries] = useState<SavedQuery[]>([]);
  const [savedQueryId, setSavedQueryId] = useState("");
  const [cronPreset, setCronPreset] = useState(CRON_PRESETS[0].value);
  const [cronCustom, setCronCustom] = useState("0 9 * * *");
  const [deliveryKind, setDeliveryKind] = useState<"email" | "telegram">("email");
  const [deliveryTarget, setDeliveryTarget] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);

  async function load() {
    setLoading(true);
    try {
      setReports(await api.listReports());
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
    api
      .listQueries()
      .then((qs) => {
        setQueries(qs);
        if (qs[0]) setSavedQueryId(qs[0].id);
      })
      .catch(() => {});
  }, []);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!savedQueryId || !deliveryTarget) return;
    const cronExpr = cronPreset || cronCustom;
    setCreating(true);
    setError(null);
    try {
      await api.createReport({
        saved_query_id: savedQueryId,
        cron_expr: cronExpr,
        delivery_kind: deliveryKind,
        delivery_target: deliveryTarget,
      });
      setDeliveryTarget("");
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setCreating(false);
    }
  }

  async function handleToggle(report: ScheduledReport) {
    try {
      await api.updateReport(report.id, { is_active: !report.is_active });
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function handleDelete(id: string) {
    if (!confirm("Delete this scheduled report?")) return;
    await api.deleteReport(id);
    await load();
  }

  function queryName(id: string) {
    return queries.find((q) => q.id === id)?.name ?? id;
  }

  return (
    <div className="mx-auto max-w-3xl">
      <h1 className="mb-6 text-2xl font-extrabold tracking-tight">Scheduled reports</h1>
      <p className="mb-6 text-sm text-slate-400">
        Runs a saved query on a schedule and delivers the result by email or Telegram. Needs SMTP or a
        Telegram bot token configured on the server — ask whoever deployed this if delivery fails.
      </p>

      <form onSubmit={handleCreate} className="glass mb-6 space-y-4 rounded-2xl p-5">
        <div>
          <label className="mb-1 block text-xs font-medium text-slate-400">Query</label>
          <select
            value={savedQueryId}
            onChange={(e) => setSavedQueryId(e.target.value)}
            className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
          >
            {queries.map((q) => (
              <option key={q.id} value={q.id}>
                {q.name}
              </option>
            ))}
          </select>
          {queries.length === 0 && (
            <p className="mt-1 text-xs text-slate-500">Save a query first — nothing to schedule yet.</p>
          )}
        </div>

        <div>
          <label className="mb-1 block text-xs font-medium text-slate-400">Schedule</label>
          <select
            value={cronPreset}
            onChange={(e) => setCronPreset(e.target.value)}
            className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
          >
            {CRON_PRESETS.map((p) => (
              <option key={p.label} value={p.value}>
                {p.label}
              </option>
            ))}
          </select>
          {cronPreset === "" && (
            <input
              value={cronCustom}
              onChange={(e) => setCronCustom(e.target.value)}
              placeholder="0 9 * * *  (minute hour day month weekday)"
              className="focus-ring mt-2 w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 font-mono text-xs outline-none"
            />
          )}
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-[140px_1fr]">
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-400">Deliver via</label>
            <select
              value={deliveryKind}
              onChange={(e) => setDeliveryKind(e.target.value as "email" | "telegram")}
              className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
            >
              <option value="email">Email</option>
              <option value="telegram">Telegram</option>
            </select>
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-400">
              {deliveryKind === "email" ? "Email address" : "Telegram chat ID"}
            </label>
            <input
              value={deliveryTarget}
              onChange={(e) => setDeliveryTarget(e.target.value)}
              placeholder={deliveryKind === "email" ? "you@example.com" : "123456789"}
              className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
            />
          </div>
        </div>

        <button
          type="submit"
          disabled={creating || queries.length === 0}
          className="focus-ring rounded-lg bg-accent-gradient px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
        >
          {creating ? "..." : "Schedule report"}
        </button>
      </form>

      {error && (
        <div className="mb-4 rounded-lg border border-down/30 bg-down/10 px-3 py-2 text-sm text-down">{error}</div>
      )}

      {loading ? (
        <p className="text-sm text-slate-500">Loading...</p>
      ) : reports.length === 0 ? (
        <p className="text-sm text-slate-500">No scheduled reports yet.</p>
      ) : (
        <div className="space-y-2">
          {reports.map((r) => (
            <div key={r.id} className="glass flex items-center justify-between rounded-xl px-4 py-3">
              <div>
                <p className="text-sm font-medium">{queryName(r.saved_query_id)}</p>
                <p className="text-xs text-slate-500">
                  <span className="font-mono">{r.cron_expr}</span> · {r.delivery_kind} → {r.delivery_target}
                  {r.last_run_at && ` · last ran ${new Date(r.last_run_at).toLocaleString()}`}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => handleToggle(r)}
                  className={`focus-ring rounded-md border px-3 py-1 text-xs font-medium ${
                    r.is_active
                      ? "border-up/30 text-up hover:bg-up/10"
                      : "border-white/10 text-slate-500 hover:bg-white/5"
                  }`}
                >
                  {r.is_active ? "Active" : "Paused"}
                </button>
                <button
                  onClick={() => handleDelete(r.id)}
                  className="focus-ring rounded-md border border-down/30 px-3 py-1 text-xs text-down hover:bg-down/10"
                >
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
