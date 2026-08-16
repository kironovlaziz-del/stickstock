"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { api, type ExportFormat } from "@/lib/api";
import type { SavedQuery, QueryResult, QueryVersion } from "@/lib/types";
import { useI18n } from "@/lib/i18n";

export default function QueryDetailPage() {
  const { t } = useI18n();
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [query, setQuery] = useState<SavedQuery | null>(null);
  const [sql, setSql] = useState("");
  const [result, setResult] = useState<QueryResult | null>(null);
  const [exportFormat, setExportFormat] = useState<ExportFormat>("csv");
  const [versions, setVersions] = useState<QueryVersion[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api
      .getQuery(params.id)
      .then((q) => {
        setQuery(q);
        setSql(q.sql_text ?? "");
      })
      .catch((e) => setError(e instanceof Error ? e.message : String(e)));
    api.queryVersions(params.id).then(setVersions).catch(() => {});
  }, [params.id]);

  async function handleRun() {
    setBusy(true);
    setError(null);
    try {
      setResult(await api.runSaved(params.id));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function handleSave() {
    setBusy(true);
    setError(null);
    try {
      await api.updateQuery(params.id, { sql_text: sql });
      setVersions(await api.queryVersions(params.id));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function handleDelete() {
    if (!confirm("Delete this query?")) return;
    await api.deleteQuery(params.id);
    router.push("/queries");
  }

  async function handleExport() {
    setError(null);
    try {
      await api.exportSavedQuery(params.id, exportFormat);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  if (!query) return <p className="text-sm text-slate-500">{t("common.loading")}</p>;

  return (
    <div className="mx-auto grid max-w-5xl grid-cols-1 gap-6 lg:grid-cols-[1fr_240px]">
      <section>
        <div className="mb-4 flex items-center justify-between">
          <h1 className="text-2xl font-extrabold tracking-tight">{query.name}</h1>
          <div className="flex items-center gap-2">
            <button
              onClick={handleRun}
              disabled={busy}
              className="focus-ring rounded-lg bg-accent-gradient px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
            >
              {t("query.run")}
            </button>
            <button
              onClick={handleSave}
              disabled={busy}
              className="focus-ring rounded-lg border border-white/10 px-4 py-2 text-sm font-medium hover:bg-white/5"
            >
              {t("common.save")}
            </button>
            <select
              value={exportFormat}
              onChange={(e) => setExportFormat(e.target.value as ExportFormat)}
              className="focus-ring rounded-lg border border-white/10 bg-base-900 px-2 py-2 text-xs outline-none"
              aria-label="Export format"
            >
              <option value="csv">CSV</option>
              <option value="xlsx">Excel</option>
              <option value="pdf">PDF</option>
            </select>
            <button
              onClick={handleExport}
              className="focus-ring rounded-lg border border-white/10 px-4 py-2 text-sm font-medium hover:bg-white/5"
            >
              {t("common.export")}
            </button>
            <button
              onClick={handleDelete}
              className="focus-ring rounded-lg border border-down/30 px-4 py-2 text-sm font-medium text-down hover:bg-down/10"
            >
              {t("common.delete")}
            </button>
          </div>
        </div>

        <textarea
          value={sql}
          onChange={(e) => setSql(e.target.value)}
          rows={10}
          spellCheck={false}
          className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 p-3 font-mono text-sm outline-none"
        />

        {error && (
          <div className="mt-4 rounded-lg border border-down/30 bg-down/10 px-3 py-2 text-sm text-down">
            {error}
          </div>
        )}

        {result && (
          <div className="glass mt-4 overflow-auto rounded-2xl p-5">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-white/10 text-slate-400">
                  {result.columns.map((c) => (
                    <th key={c} className="whitespace-nowrap px-3 py-2 font-medium">
                      {c}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {result.rows.map((row, i) => (
                  <tr key={i} className="border-b border-white/5">
                    {row.map((cell, j) => (
                      <td key={j} className="whitespace-nowrap px-3 py-2 text-slate-200">
                        {String(cell ?? "")}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <aside>
        <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-slate-400">History</h2>
        <div className="space-y-2">
          {versions.map((v) => (
            <button
              key={v.version}
              onClick={() => setSql(v.sql_text)}
              className="focus-ring block w-full rounded-lg border border-white/10 px-3 py-2 text-left text-xs hover:bg-white/5"
            >
              <span className="font-semibold">v{v.version}</span>
              <span className="ml-2 text-slate-500">{new Date(v.created_at).toLocaleString()}</span>
            </button>
          ))}
        </div>
      </aside>
    </div>
  );
}
