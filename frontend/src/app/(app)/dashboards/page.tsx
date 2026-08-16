"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { DashboardSummary } from "@/lib/types";
import { useI18n } from "@/lib/i18n";

export default function DashboardsPage() {
  const { t } = useI18n();
  const router = useRouter();
  const [dashboards, setDashboards] = useState<DashboardSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState("");

  useEffect(() => {
    api
      .listDashboards()
      .then(setDashboards)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!name) return;
    const dashboard = await api.createDashboard({ name });
    router.push(`/dashboards/${dashboard.id}/edit`);
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
        <p className="text-sm text-slate-500">{t("common.loading")}</p>
      ) : dashboards.length === 0 ? (
        <p className="text-sm text-slate-500">No dashboards yet.</p>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {dashboards.map((d) => (
            <Link
              key={d.id}
              href={`/dashboards/${d.id}`}
              className="glass focus-ring block rounded-2xl p-5 hover:bg-white/5"
            >
              <p className="font-semibold">{d.name}</p>
              <p className="mt-1 text-xs text-slate-500">{new Date(d.created_at).toLocaleDateString()}</p>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
