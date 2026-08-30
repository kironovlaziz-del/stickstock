"use client";

import { useEffect, useState, useCallback } from "react";
import { useParams } from "next/navigation";
import GridLayout, { WidthProvider } from "react-grid-layout";
import "react-grid-layout/css/styles.css";
import "react-resizable/css/styles.css";
import type { QueryResult, Widget, LayoutItem } from "@/lib/types";
import { mergeLayout, GRID_COLS } from "@/lib/dashboardLayout";
import ChartWidget from "@/components/ChartWidget";
import CommentThread from "@/components/CommentThread";

const Grid = WidthProvider(GridLayout);
const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL ?? "/api";

interface PublicDashboard {
  id: string;
  name: string;
  layout: LayoutItem[];
  widgets: Widget[];
}

export default function SharedDashboardPage() {
  const params = useParams<{ token: string }>();
  const [dashboard, setDashboard] = useState<PublicDashboard | null>(null);
  const [results, setResults] = useState<Record<string, QueryResult>>({});
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/public/dashboards/${params.token}`);
      if (!res.ok) {
        const body = await res.json().catch(() => null);
        throw new Error(body?.error ?? "This share link is invalid or has been revoked.");
      }
      const d: PublicDashboard = await res.json();
      setDashboard(d);

      const entries = await Promise.all(
        d.widgets.map(async (w) => {
          const r = await fetch(`${API_BASE}/public/dashboards/${params.token}/widgets/${w.id}/run`, {
            method: "POST",
          });
          const result = r.ok ? ((await r.json()) as QueryResult) : { columns: [], rows: [], truncated: false };
          return [w.id, result] as const;
        })
      );
      setResults(Object.fromEntries(entries));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [params.token]);

  useEffect(() => {
    load();
  }, [load]);

  if (error) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-base-950 px-4 text-slate-100">
        <div className="glass max-w-sm rounded-2xl p-6 text-center">
          <p className="text-sm text-down">{error}</p>
        </div>
      </main>
    );
  }
  if (!dashboard) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-base-950 text-slate-400">
        Loading...
      </main>
    );
  }

  const layout = mergeLayout(dashboard.widgets, dashboard.layout);

  return (
    <main className="min-h-screen bg-base-950 p-6 text-slate-100">
      <div className="mx-auto max-w-6xl">
        <div className="mb-6 flex items-center gap-2">
          <img src="/logotip.png" alt="Stickstock" style={{height:"32px",objectFit:"contain"}} />
<span className="text-xs font-medium uppercase tracking-wide text-slate-500">Shared — read only</span>
        </div>
        <h1 className="mb-6 text-2xl font-extrabold tracking-tight">{dashboard.name}</h1>

        {dashboard.widgets.length === 0 ? (
          <p className="text-sm text-slate-500">This dashboard has no widgets yet.</p>
        ) : (
          <Grid
            className="layout"
            layout={layout}
            cols={GRID_COLS}
            rowHeight={80}
            margin={[16, 16]}
            isDraggable={false}
            isResizable={false}
          >
            {dashboard.widgets.map((w) => (
              <div key={w.id} className="glass rounded-2xl p-5 flex flex-col">
                <p className="mb-3 text-xs font-medium uppercase tracking-wide text-slate-400">{w.chart_type}</p>
                {results[w.id] ? (
                  <ChartWidget result={results[w.id]} chartType={w.chart_type} />
                ) : (
                  <p className="text-sm text-slate-500">Loading...</p>
                )}
                <CommentThread dashboardId={dashboard.id} widgetId={w.id} readOnly />
              </div>
            ))}
          </Grid>
        )}
      </div>
    </main>
  );
}
