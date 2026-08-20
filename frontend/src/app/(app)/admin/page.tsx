"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { AdminUser, AdminStats } from "@/lib/types";

export default function AdminPage() {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [stats, setStats] = useState<AdminStats | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  async function load() {
    setLoading(true);
    try {
      const [u, s] = await Promise.all([api.adminListUsers(), api.adminStats()]);
      setUsers(u);
      setStats(s);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function toggleBlocked(u: AdminUser) {
    try {
      await api.adminUpdateUser(u.id, { is_blocked: !u.is_blocked });
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function toggleAdmin(u: AdminUser) {
    try {
      await api.adminUpdateUser(u.id, { is_admin: !u.is_admin });
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  if (error) {
    return (
      <div className="mx-auto max-w-3xl">
        <div className="rounded-lg border border-down/30 bg-down/10 px-3 py-2 text-sm text-down">{error}</div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-3xl">
      <h1 className="mb-6 text-2xl font-extrabold tracking-tight">Admin</h1>

      {stats && (
        <div className="mb-6 grid grid-cols-2 gap-3 sm:grid-cols-4">
          {[
            { label: "Users", value: stats.users },
            { label: "Connections", value: stats.data_sources },
            { label: "Queries", value: stats.saved_queries },
            { label: "Dashboards", value: stats.dashboards },
          ].map((s) => (
            <div key={s.label} className="glass rounded-2xl p-4">
              <p className="text-2xl font-extrabold tracking-tight">{s.value}</p>
              <p className="text-xs text-slate-500">{s.label}</p>
            </div>
          ))}
        </div>
      )}

      <div className="glass rounded-2xl p-5">
        <p className="mb-4 text-sm font-semibold">Users</p>

        {loading ? (
          <p className="text-sm text-slate-500">Loading...</p>
        ) : (
          <div className="space-y-2">
            {users.map((u) => (
              <div
                key={u.id}
                className="flex items-center justify-between rounded-lg border border-white/10 px-3 py-2"
              >
                <div>
                  <p className="text-sm">{u.email}</p>
                  <p className="text-xs text-slate-500">
                    {u.locale} · joined {new Date(u.created_at).toLocaleDateString()}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  {u.is_admin && (
                    <span className="rounded-full border border-accent-blue/30 bg-accent-blue/10 px-2 py-0.5 text-xs text-accent-blue">
                      admin
                    </span>
                  )}
                  <button
                    onClick={() => toggleAdmin(u)}
                    className="focus-ring rounded-md border border-white/10 px-2 py-1 text-xs hover:bg-white/5"
                  >
                    {u.is_admin ? "Revoke admin" : "Make admin"}
                  </button>
                  <button
                    onClick={() => toggleBlocked(u)}
                    className={`focus-ring rounded-md border px-2 py-1 text-xs ${
                      u.is_blocked
                        ? "border-up/30 text-up hover:bg-up/10"
                        : "border-down/30 text-down hover:bg-down/10"
                    }`}
                  >
                    {u.is_blocked ? "Unblock" : "Block"}
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
