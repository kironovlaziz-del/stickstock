"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import type { DataSource, SavedQuery, QueryResult } from "@/lib/types";
import TemplateEditor from "@/components/TemplateEditor";

export default function QueriesPage() {
  const [queries, setQueries] = useState<SavedQuery[]>([]);
  const [sources, setSources] = useState<DataSource[]>([]);
  const [dataSourceId, setDataSourceId] = useState("");
  const [sql, setSql] = useState("SELECT * FROM ");
  const [result, setResult] = useState<QueryResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [running, setRunning] = useState(false);
  const [saveName, setSaveName] = useState("");
  const [mode, setMode] = useState<"sql" | "template">("sql");

  useEffect(() => {
    api.listQueries().then(setQueries).catch(() => {});
    api.listDataSources().then((ds) => {
      setSources(ds);
      if (ds.length > 0) {
        const savedId = localStorage.getItem("selectedDataSourceId");
        if (savedId && ds.some(d => d.id === savedId)) {
          setDataSourceId(savedId);
        } else {
          setDataSourceId(ds[0].id);
          localStorage.setItem("selectedDataSourceId", ds[0].id);
        }
      }
    }).catch(() => {});
  }, []);

  async function handleRunQuery(query: string, params?: Record<string, unknown>) {
    let selectedId = dataSourceId;
    if (!selectedId && sources.length > 0) {
      selectedId = sources[0].id;
      setDataSourceId(selectedId);
      localStorage.setItem("selectedDataSourceId", selectedId);
    }
    
    if (!selectedId) {
      setError("Please add a data source in Connections first.");
      return;
    }
    
    setRunning(true);
    setError(null);
    try {
      console.log("Running query with:", { data_source_id: selectedId, sql: query, params });
      const res = await api.runAdHoc({ data_source_id: selectedId, sql: query, params: params || {} });
      setResult(res);
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
        <Link href="/queries/builder" className="focus-ring mb-4 flex items-center gap-2 rounded-lg bg-accent-gradient px-3 py-2 text-sm font-semibold text-white">
          ✨ Visual query builder
        </Link>
        <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-slate-400">Saved Queries</h2>
        <div className="space-y-1">
          {queries.map((q) => (
            <Link key={q.id} href={"/queries/" + q.id} className="focus-ring block rounded-lg px-3 py-2 text-sm text-slate-300 hover:bg-white/5">
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
              onChange={(e) => {
                setDataSourceId(e.target.value);
                localStorage.setItem("selectedDataSourceId", e.target.value);
              }}
              className="focus-ring rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
            >
              {sources.map((s) => (<option key={s.id} value={s.id}>{s.name}</option>))}
            </select>
            {sources.length === 0 && <span className="text-sm text-slate-500">No data sources. Add one in Connections.</span>}
          </div>

          <div className="flex gap-2 mb-3">
            <button onClick={() => setMode("sql")} className={"px-3 py-1 text-sm rounded " + (mode === "sql" ? "bg-accent-blue/20 border border-accent-blue text-white" : "border border-white/10 text-slate-400 hover:bg-white/5")}>
              SQL Editor
            </button>
            <button onClick={() => setMode("template")} className={"px-3 py-1 text-sm rounded " + (mode === "template" ? "bg-accent-blue/20 border border-accent-blue text-white" : "border border-white/10 text-slate-400 hover:bg-white/5")}>
              Jinja2 Template
            </button>
          </div>

          {mode === "sql" ? (
            <div>
              <textarea value={sql} onChange={(e) => setSql(e.target.value)} rows={8} spellCheck={false} placeholder="SELECT * FROM ... (:param for variables)" className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 p-3 font-mono text-sm outline-none" />
              <div className="mt-3 flex items-center gap-2">
                <button onClick={() => handleRunQuery(sql)} disabled={running} className="focus-ring rounded-lg bg-accent-gradient px-4 py-2 text-sm font-semibold text-white disabled:opacity-50">
                  {running ? "..." : "Run"}
                </button>
                <input value={saveName} onChange={(e) => setSaveName(e.target.value)} placeholder="New query" className="focus-ring flex-1 rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none" />
                <button onClick={handleSave} className="focus-ring rounded-lg border border-white/10 px-4 py-2 text-sm font-medium hover:bg-white/5">
                  Save
                </button>
              </div>
            </div>
          ) : (
            <div>
              <TemplateEditor 
                dataSourceId={dataSourceId} 
                onRender={(renderedSQL, params) => {
                  setSql(renderedSQL);
                  handleRunQuery(renderedSQL, params);
                }} 
              />
              <div className="mt-3 flex items-center gap-2">
                <input value={saveName} onChange={(e) => setSaveName(e.target.value)} placeholder="Save template as query" className="focus-ring flex-1 rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none" />
                <button onClick={handleSave} className="focus-ring rounded-lg border border-white/10 px-4 py-2 text-sm font-medium hover:bg-white/5">
                  Save
                </button>
              </div>
            </div>
          )}
        </div>

        {error && (
          <div className="mt-4 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400 flex items-center justify-between">
            <span>{error}</span>
            <button onClick={() => setError(null)} className="text-red-400 hover:text-white transition">✕</button>
          </div>
        )}

        {result && (
          <div className="glass mt-4 overflow-auto rounded-2xl p-5">
            <div className="mb-2 flex items-center justify-between">
              <span className="text-xs text-slate-400">{result.rows.length} rows</span>
            </div>
            <div className="overflow-x-auto max-h-[400px] overflow-y-auto">
              <table className="w-full text-left text-sm border-collapse">
                <thead className="sticky top-0 bg-base-900 z-10">
                  <tr className="border-b border-white/10 text-slate-400">
                    {result.columns.map((c) => (<th key={c} className="whitespace-nowrap px-3 py-2 font-medium border-r border-white/5 last:border-r-0">{c}</th>))}
                  </tr>
                </thead>
                <tbody>
                  {result.rows.map((row, i) => (<tr key={i} className="border-b border-white/5 hover:bg-white/5 transition">
                    {row.map((cell, j) => (<td key={j} className="whitespace-nowrap px-3 py-2 text-slate-200 border-r border-white/5 last:border-r-0">{String(cell ?? "")}</td>))}
                  </tr>))}
                </tbody>
              </table>
            </div>
            {result.truncated && (<p className="mt-2 text-xs text-slate-500">Results truncated to the first rows.</p>)}
          </div>
        )}
      </section>
    </div>
  );
}