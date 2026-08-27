"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { DataSource } from "@/lib/types";
import { SkeletonTableRow } from "@/components/Skeleton";
import ProfileModal from "@/components/ProfileModal";

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

  // SSH fields
  const [sshHost, setSshHost] = useState("");
  const [sshPort, setSshPort] = useState(22);
  const [sshUser, setSshUser] = useState("");
  const [sshPassword, setSshPassword] = useState("");
  const [sshPrivateKey, setSshPrivateKey] = useState("");

  // Profile modal
  const [profileData, setProfileData] = useState<any>(null);
  const [profileTable, setProfileTable] = useState("");
  const [showProfile, setShowProfile] = useState(false);

  // Copy feedback
  const [copiedId, setCopiedId] = useState<string | null>(null);

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
      await api.createDataSource({
        name,
        kind,
        dsn,
        ssh_host: sshHost || undefined,
        ssh_port: sshPort || 22,
        ssh_user: sshUser || undefined,
        ssh_password: sshPassword || undefined,
        ssh_private_key: sshPrivateKey || undefined,
      });
      setName("");
      setDsn("");
      setSshHost("");
      setSshPort(22);
      setSshUser("");
      setSshPassword("");
      setSshPrivateKey("");
      setShowForm(false);
      await load();
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
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function handleProfile(id: string, name: string) {
    const table = prompt(`Enter table name to profile for "${name}":`, "");
    if (!table) return;
    try {
      const result = await api.profileDataSource(id, table);
      setProfileData(result);
      setProfileTable(table);
      setShowProfile(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  const handleCopy = (id: string) => {
    navigator.clipboard.writeText(id);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  };

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

          {/* SSH Section */}
          <details className="border-t border-white/10 pt-4">
            <summary className="cursor-pointer text-sm font-medium text-slate-400 hover:text-white">
              Advanced: SSH Tunnel
            </summary>
            <div className="mt-3 space-y-3">
              <div>
                <label className="mb-1 block text-xs font-medium text-slate-400">SSH Host</label>
                <input
                  value={sshHost}
                  onChange={(e) => setSshHost(e.target.value)}
                  placeholder="ssh.example.com"
                  className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-slate-400">SSH Port</label>
                <input
                  type="number"
                  value={sshPort}
                  onChange={(e) => setSshPort(Number(e.target.value))}
                  placeholder="22"
                  className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-slate-400">SSH User</label>
                <input
                  value={sshUser}
                  onChange={(e) => setSshUser(e.target.value)}
                  placeholder="root"
                  className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-slate-400">SSH Password (optional)</label>
                <input
                  type="password"
                  value={sshPassword}
                  onChange={(e) => setSshPassword(e.target.value)}
                  placeholder="password"
                  className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-slate-400">SSH Private Key (optional)</label>
                <textarea
                  value={sshPrivateKey}
                  onChange={(e) => setSshPrivateKey(e.target.value)}
                  placeholder="-----BEGIN RSA PRIVATE KEY-----..."
                  rows={3}
                  className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 font-mono text-sm outline-none"
                />
              </div>
            </div>
          </details>

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
                <p className="text-xs text-slate-500">
                  {s.kind}
                  {s.ssh_host && (
                    <span className="ml-2 text-accent-blue">🔒 SSH: {s.ssh_user}@{s.ssh_host}</span>
                  )}
                </p>
                <p className="text-xs text-slate-500 mt-1 font-mono">
                  ID: <span className="text-slate-400">{s.id}</span>
                  <button
                    onClick={() => handleCopy(s.id)}
                    className="ml-2 text-accent-blue hover:text-accent-blue/80 transition"
                    title="Copy ID"
                  >
                    {copiedId === s.id ? "✅" : "📋"}
                  </button>
                </p>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => handleDelete(s.id)}
                  className="text-sm font-medium text-down hover:text-down/80 transition"
                >
                  Delete
                </button>
                <button
                  onClick={() => handleProfile(s.id, s.name)}
                  className="text-sm font-medium text-accent-blue hover:text-accent-blue/80 transition"
                >
                  Profile
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      <ProfileModal
        isOpen={showProfile}
        data={profileData}
        onClose={() => setShowProfile(false)}
        tableName={profileTable}
      />
    </div>
  );
}
