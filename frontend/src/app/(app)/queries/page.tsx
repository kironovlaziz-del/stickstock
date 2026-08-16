"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import type { DataSource, SavedQuery, QueryResult } from "@/lib/types";
import { useI18n } from "@/lib/i18n";

export default function QueriesPage() {
  const { t } = useI18n();
  const [queries, setQueries] = useState<SavedQuery[]>([]);
  const [sources, setSources] = useState<DataSource[]>([]);
  const [dataSourceId, setDataSourceId] = useState("");
  const [sql, setSql] = useState("SELECT * FROM ");
  const [result, setResult] = useState<QueryResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [running, setRunning] = useState(false);
  const [saveName, setSaveName] = useState("");

  useEffect(() => {
    api.listQueries().then(setQueries).catch(() => {});
    api
      .listDataSources()
      .then((ds) => {
        setSources(ds);
        if (ds[0]) setDataSourceId(ds[0].id);
      })
      .catch(() => {});
  }, []);

  async function handleRun() {
    if (!dataSourceId) {
      setError("Add a connection first.");
      return;
    }
    setRunning(true);
    setError(null);
    try {
      setResult(await api.runAdHoc({ data_source_id: dataSourceId, sql }));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setRunning(false);
    }
  }

  async function handleSave() {
    if (!saveName || !dataSourceId) return;
    try {
      await api.createQuery({ data_source_id: dataSourceId, name: saveName, sql_text: sql });
      setSaveName("");
      setQueries(await api.listQueries());
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-[280px_1fr]">
      <aside>
        <Link
          href="/queries/builder"
          className="focus-ring mb-4 flex items-center gap-2 rounded-lg bg-accent-gradient px-3 py-2 text-sm font-semibold text-white"
        >
          ✨ Visual query builder
        </Link>

        <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-slate-400">
          {t("nav.queries")}
        </h2>
        <div className="space-y-1">
          {queries.map((q) => (
            <Link
              key={q.id}
              href={`/queries/${q.id}`}
              className="focus-ring block rounded-lg px-3 py-2 text-sm text-slate-300 hover:bg-white/5"
            >
              {q.name}
            </Link>
          ))}
          {queries.length === 0 && <p className="px-3 text-sm text-slate-500">No saved queries yet.</p>}
        </div>
      </aside>

      <section>
        <div className="glass rounded-2xl p-5">
          <div className="mb-3 flex items-center gap-3">
            <select
              value={dataSourceId}
              onChange={(e) => setDataSourceId(e.target.value)}
              className="focus-ring rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
            >
              {sources.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
            <button
              onClick={handleRun}
              disabled={running}
              className="focus-ring rounded-lg bg-accent-gradient px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
            >
              {running ? "..." : t("query.run")}
            </button>
          </div>

          <textarea
            value={sql}
            onChange={(e) => setSql(e.target.value)}
            rows={8}
            spellCheck={false}
            className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 p-3 font-mono text-sm outline-none"
          />

          <div className="mt-3 flex items-center gap-2">
            <input
              value={saveName}
              onChange={(e) => setSaveName(e.target.value)}
              placeholder={t("query.new")}
              className="focus-ring flex-1 rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
            />
            <button
              onClick={handleSave}
              className="focus-ring rounded-lg border border-white/10 px-4 py-2 text-sm font-medium hover:bg-white/5"
            >
              {t("query.save")}
            </button>
          </div>
        </div>

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
            {result.truncated && (
              <p className="mt-2 text-xs text-slate-500">Results truncated to the first rows.</p>
            )}
          </div>
        )}
      </section>
    </div>
  );
}
