"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { DashboardSummary } from "@/lib/types";
import { useI18n } from "@/lib/i18n";
import { SkeletonCard } from "@/components/Skeleton";

export default function DashboardsPage() {
  const { t } = useI18n();
  const router = useRouter();
  const [dashboards, setDashboards] = useState<DashboardSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState("");
  const [error, setError] = useState<string | null>(null);

  async function loadDashboards() {
    setLoading(true);
    setError(null);
    try {
      const data = await api.listDashboards();
      setDashboards(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadDashboards();
  }, []);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!name) return;
    try {
      const dashboard = await api.createDashboard({ name });
      router.push(`/dashboards/${dashboard.id}/edit`);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function handleDelete(id: string) {
    if (!confirm("Delete this dashboard? All associated widgets will be removed.")) return;
    try {
      await api.deleteDashboard(id);
      await loadDashboards();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div className="mx-auto max-w-4xl">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-extrabold tracking-tight">{t("nav.dashboards")}</h1>
        <button
          onClick={() => setCreating((c) => !c)}
          className="focus-ring rounded-lg bg-accent-gradient px-4 py-2 text-sm font-semibold text-white"
        >
          {t("dashboard.create")}
        </button>
      </div>

      {error && (
        <div className="mb-4 rounded-lg border border-down/30 bg-down/10 px-3 py-2 text-sm text-down">
          {error}
        </div>
      )}

      {creating && (
        <form onSubmit={handleCreate} className="glass mb-6 flex gap-2 rounded-2xl p-4">
          <input
            autoFocus
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Dashboard name"
            className="focus-ring flex-1 rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
          />
          <button
            type="submit"
            className="focus-ring rounded-lg bg-accent-gradient px-4 py-2 text-sm font-semibold text-white"
          >
            {t("common.save")}
          </button>
        </form>
      )}

      {loading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <SkeletonCard />
          <SkeletonCard />
          <SkeletonCard />
          <SkeletonCard />
          <SkeletonCard />
          <SkeletonCard />
        </div>
      ) : dashboards.length === 0 ? (
        <p className="text-sm text-slate-500">No dashboards yet.</p>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {dashboards.map((d) => (
            <div
              key={d.id}
              className="glass focus-ring block rounded-2xl p-5 hover:bg-white/5"
            >
              <div className="flex items-center justify-between">
                <Link href={`/dashboards/${d.id}`} className="flex-1">
                  <p className="font-semibold">{d.name}</p>
                  <p className="mt-1 text-xs text-slate-500">{new Date(d.created_at).toLocaleDateString()}</p>
                </Link>
                <button
                  onClick={(e) => {
                    e.preventDefault();
                    handleDelete(d.id);
                  }}
                  className="text-sm font-medium text-down hover:text-down/80 transition ml-2"
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