"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { DataSource } from "@/lib/types";
import { SkeletonTableRow } from "@/components/Skeleton";

const KINDS = ["postgres", "mysql", "mongodb", "rest"];

const DSN_PLACEHOLDERS: Record<string, string> = {
  postgres: "postgres://user:pass@host:5432/dbname",
  mysql: "user:pass@tcp(host:3306)/dbname",
  mongodb: "mongodb://user:pass@host:27017/dbname",
  rest: 'http://demo-rest:80',
};

export default function ConnectionsPage() {
  const [sources, setSources] = useState<DataSource[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState("");
  const [kind, setKind] = useState(KINDS[0]);
  const [dsn, setDsn] = useState("");
  const [saving, setSaving] = useState(false);

  async function load() {
    setLoading(true);
    try {
      const data = await api.listDataSources();
      setSources(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError(null);
    try {
      await api.createDataSource({ name, kind, dsn });
      setName("");
      setDsn("");
      setShowForm(false);
      await load(); // обновляем список после создания
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: string) {
    if (!confirm("Delete this connection? All related queries will become non-functional.")) return;
    try {
      await api.deleteDataSource(id);
      await load(); // обновляем список после удаления
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div className="mx-auto max-w-3xl">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-extrabold tracking-tight">Connections</h1>
        <button
          onClick={() => setShowForm((s) => !s)}
          className="focus-ring rounded-lg bg-accent-gradient px-4 py-2 text-sm font-semibold text-white"
        >
          Add Connection
        </button>
      </div>

      {error && (
        <div className="mb-4 rounded-lg border border-down/30 bg-down/10 px-3 py-2 text-sm text-down">
          {error}
        </div>
      )}

      {showForm && (
        <form onSubmit={handleCreate} className="glass mb-6 space-y-4 rounded-2xl p-5">
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-400">Name</label>
            <input
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Production Postgres"
              className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
            />
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-400">Kind</label>
            <select
              value={kind}
              onChange={(e) => setKind(e.target.value)}
              className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
            >
              {KINDS.map((k) => (
                <option key={k} value={k}>
                  {k}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-400">
              {kind === "rest" ? "Base URL (or JSON with headers)" : "Connection string"}
            </label>
            <input
              required
              value={dsn}
              onChange={(e) => setDsn(e.target.value)}
              placeholder={DSN_PLACEHOLDERS[kind] ?? ""}
              className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 font-mono text-sm outline-none"
            />
          </div>
          <button
            type="submit"
            disabled={saving}
            className="focus-ring rounded-lg bg-accent-gradient px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
          >
            {saving ? "..." : "Save"}
          </button>
        </form>
      )}

      {loading ? (
        <div className="space-y-2">
          <SkeletonTableRow />
          <SkeletonTableRow />
          <SkeletonTableRow />
          <SkeletonTableRow />
        </div>
      ) : sources.length === 0 ? (
        <p className="text-sm text-slate-500">No connections yet. Add one to get started.</p>
      ) : (
        <div className="space-y-2">
          {sources.map((s) => (
            <div key={s.id} className="glass flex items-center justify-between rounded-xl px-4 py-3">
              <div>
                <p className="font-medium">{s.name}</p>
                <p className="text-xs text-slate-500">{s.kind}</p>
              </div>
              <button
                onClick={() => handleDelete(s.id)}
                className="text-sm font-medium text-down hover:text-down/80 transition"
              >
                Delete
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
