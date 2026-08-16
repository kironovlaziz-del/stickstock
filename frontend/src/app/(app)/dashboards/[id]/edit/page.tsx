"use client";

import { useEffect, useState, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import GridLayout, { WidthProvider } from "react-grid-layout";
import "react-grid-layout/css/styles.css";
import "react-resizable/css/styles.css";
import { api } from "@/lib/api";
import type { DashboardDetail, SavedQuery, ChartType, LayoutItem } from "@/lib/types";
import { mergeLayout, GRID_COLS } from "@/lib/dashboardLayout";

const Grid = WidthProvider(GridLayout);

const CHART_TYPES: ChartType[] = [
  "line",
  "bar",
  "pie",
  "scatter",
  "table",
  "heatmap",
  "boxplot",
  "treemap",
];

export default function DashboardEditPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [dashboard, setDashboard] = useState<DashboardDetail | null>(null);
  const [layout, setLayout] = useState<LayoutItem[]>([]);
  const [queries, setQueries] = useState<SavedQuery[]>([]);
  const [savedQueryId, setSavedQueryId] = useState("");
  const [chartType, setChartType] = useState<ChartType>("bar");
  const [error, setError] = useState<string | null>(null);
  const [savingLayout, setSavingLayout] = useState(false);

  const load = useCallback(async () => {
    const d = await api.getDashboard(params.id);
    setDashboard(d);
    setLayout(mergeLayout(d.widgets, d.layout));
  }, [params.id]);

  useEffect(() => {
    load();
    api
      .listQueries()
      .then((qs) => {
        setQueries(qs);
        if (qs[0]) setSavedQueryId(qs[0].id);
      })
      .catch(() => {});
  }, [load]);

  // Only persists on drag/resize *stop* (not react-grid-layout's more
  // frequent onLayoutChange, which fires continuously mid-drag) — one API
  // call per completed gesture instead of one per pixel of movement.
  async function persistLayout(next: LayoutItem[]) {
    setLayout(next);
    setSavingLayout(true);
    try {
      await api.updateDashboard(params.id, { layout: next });
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSavingLayout(false);
    }
  }

  async function handleAdd(e: React.FormEvent) {
    e.preventDefault();
    if (!savedQueryId) {
      setError("Save a query first, then attach it here.");
      return;
    }
    try {
      await api.addWidget(params.id, { saved_query_id: savedQueryId, chart_type: chartType });
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function handleRemove(widgetId: string) {
    await api.deleteWidget(params.id, widgetId);
    await load();
  }

  if (!dashboard) return <p className="text-sm text-slate-500">Loading...</p>;

  return (
    <div className="mx-auto max-w-4xl">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-extrabold tracking-tight">Edit — {dashboard.name}</h1>
        <div className="flex items-center gap-3">
          {savingLayout && <span className="text-xs text-slate-500">Saving layout...</span>}
          <button
            onClick={() => router.push(`/dashboards/${dashboard.id}`)}
            className="focus-ring rounded-lg bg-accent-gradient px-4 py-2 text-sm font-semibold text-white"
          >
            Done
          </button>
        </div>
      </div>

      <form onSubmit={handleAdd} className="glass mb-6 space-y-3 rounded-2xl p-5">
        <p className="text-sm font-semibold">Add a widget</p>
        <select
          value={savedQueryId}
          onChange={(e) => setSavedQueryId(e.target.value)}
          className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
        >
          {queries.map((q) => (
            <option key={q.id} value={q.id}>
              {q.name}
            </option>
          ))}
        </select>
        <select
          value={chartType}
          onChange={(e) => setChartType(e.target.value as ChartType)}
          className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
        >
          {CHART_TYPES.map((c) => (
            <option key={c} value={c}>
              {c}
            </option>
          ))}
        </select>
        <button
          type="submit"
          className="focus-ring w-full rounded-lg border border-white/10 px-4 py-2 text-sm font-medium hover:bg-white/5"
        >
          Add widget
        </button>
      </form>

      {error && (
        <div className="mb-4 rounded-lg border border-down/30 bg-down/10 px-3 py-2 text-sm text-down">
          {error}
        </div>
      )}

      {dashboard.widgets.length > 0 && (
        <>
          <p className="mb-2 text-xs text-slate-500">
            Drag to reposition, resize from the bottom-right corner — saved automatically.
          </p>
          <Grid
            className="layout mb-6"
            layout={layout}
            cols={GRID_COLS}
            rowHeight={70}
            margin={[16, 16]}
            onDragStop={(l) => persistLayout(l as LayoutItem[])}
            onResizeStop={(l) => persistLayout(l as LayoutItem[])}
          >
            {dashboard.widgets.map((w) => (
              <div key={w.id} className="glass flex flex-col overflow-hidden rounded-2xl p-4">
                <div className="mb-2 flex items-center justify-between gap-2">
                  <p className="truncate text-xs font-medium text-slate-400">
                    {queries.find((q) => q.id === w.saved_query_id)?.name ?? w.saved_query_id} · {w.chart_type}
                  </p>
                  <button
                    onClick={() => handleRemove(w.id)}
                    className="focus-ring shrink-0 rounded-md border border-down/30 px-2 py-0.5 text-xs text-down hover:bg-down/10"
                  >
                    Remove
                  </button>
                </div>
              </div>
            ))}
          </Grid>
        </>
      )}
    </div>
  );
}
