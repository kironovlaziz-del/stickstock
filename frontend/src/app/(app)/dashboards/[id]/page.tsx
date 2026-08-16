"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import GridLayout, { WidthProvider } from "react-grid-layout";
import "react-grid-layout/css/styles.css";
import "react-resizable/css/styles.css";
import { api } from "@/lib/api";
import type { DashboardDetail, QueryResult } from "@/lib/types";
import { mergeLayout, GRID_COLS } from "@/lib/dashboardLayout";
import ChartWidget from "@/components/ChartWidget";

const Grid = WidthProvider(GridLayout);

export default function DashboardViewPage() {
  const params = useParams<{ id: string }>();
  const [dashboard, setDashboard] = useState<DashboardDetail | null>(null);
  const [results, setResults] = useState<Record<string, QueryResult>>({});
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const d = await api.getDashboard(params.id);
      setDashboard(d);
      const entries = await Promise.all(
        d.widgets.map(async (w) => [w.id, await api.runSaved(w.saved_query_id)] as const)
      );
      setResults(Object.fromEntries(entries));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [params.id]);

  useEffect(() => {
    load();
  }, [load]);

  if (error) {
    return (
      <div className="rounded-lg border border-down/30 bg-down/10 px-3 py-2 text-sm text-down">
        {error}
      </div>
    );
  }
  if (!dashboard) return <p className="text-sm text-slate-500">Loading...</p>;

  const layout = mergeLayout(dashboard.widgets, dashboard.layout);

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-extrabold tracking-tight">{dashboard.name}</h1>
        <Link
          href={`/dashboards/${dashboard.id}/edit`}
          className="focus-ring rounded-lg border border-white/10 px-4 py-2 text-sm font-medium hover:bg-white/5"
        >
          Edit widgets
        </Link>
      </div>

      {dashboard.widgets.length === 0 ? (
        <p className="text-sm text-slate-500">No widgets yet — add one from the edit page.</p>
      ) : (
        <Grid
          className="layout"
          layout={layout}
          cols={GRID_COLS}
          rowHeight={70}
          margin={[16, 16]}
          isDraggable={false}
          isResizable={false}
        >
          {dashboard.widgets.map((w) => (
            <div key={w.id} className="glass overflow-hidden rounded-2xl p-5">
              <p className="mb-3 text-xs font-medium uppercase tracking-wide text-slate-400">{w.chart_type}</p>
              {results[w.id] ? (
                <ChartWidget result={results[w.id]} chartType={w.chart_type} />
              ) : (
                <p className="text-sm text-slate-500">Loading...</p>
              )}
            </div>
          ))}
        </Grid>
      )}
    </div>
  );
}
