"use client";

import { useEffect, useState, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import GridLayout, { WidthProvider } from "react-grid-layout";
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
  "kpi",
  "forecast",
];

export default function DashboardEditPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [dashboard, setDashboard] = useState<DashboardDetail | null>(null);
  const [layout, setLayout] = useState<LayoutItem[]>([]);
  const [queries, setQueries] = useState<SavedQuery[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [savingLayout, setSavingLayout] = useState(false);
  const [editingWidget, setEditingWidget] = useState<any | null>(null);
  const [widgetConfig, setWidgetConfig] = useState({
    saved_query_id: "",
    chart_type: "bar" as ChartType,
    x_field: "",
    y_field: "",
    horizon: 10,
  });

  const load = useCallback(async () => {
    try {
      const d = await api.getDashboard(params.id);
      setDashboard(d);
      setLayout(mergeLayout(d.widgets, d.layout));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [params.id]);

  useEffect(() => {
    load();
    api
      .listQueries()
      .then((qs) => {
        setQueries(qs);
      })
      .catch(() => {});
  }, [load]);

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

  const handleSaveWidget = async () => {
    if (!editingWidget) return;
    try {
      const payload = {
        saved_query_id: widgetConfig.saved_query_id,
        chart_type: widgetConfig.chart_type,
        config: {
          x_field: widgetConfig.x_field,
          y_field: widgetConfig.y_field,
          horizon: widgetConfig.horizon,
        },
      };
      if (editingWidget.id) {
        await api.updateWidget(params.id, editingWidget.id, payload);
      } else {
        await api.addWidget(params.id, payload);
      }
      setEditingWidget(null);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const handleRemoveWidget = async (widgetId: string) => {
    if (!confirm("Delete this widget?")) return;
    try {
      await api.deleteWidget(params.id, widgetId);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  if (!dashboard) return <p className="text-sm text-slate-500">Loading...</p>;

  return (
    <div className="mx-auto max-w-4xl">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-extrabold tracking-tight">Edit — {dashboard.name}</h1>
        <div className="flex items-center gap-3">
          {savingLayout && <span className="text-xs text-slate-500">Saving layout...</span>}
          <button
            onClick={() => {
              setEditingWidget({ id: null, saved_query_id: "", chart_type: "bar", config: {} });
              setWidgetConfig({
                saved_query_id: queries[0]?.id || "",
                chart_type: "bar",
                x_field: "",
                y_field: "",
                horizon: 10,
              });
            }}
            className="focus-ring rounded-lg bg-accent-gradient px-4 py-2 text-sm font-semibold text-white"
          >
            Add Widget
          </button>
          <button
            onClick={() => router.push(`/dashboards/${dashboard.id}`)}
            className="focus-ring rounded-lg border border-white/10 px-4 py-2 text-sm font-medium hover:bg-white/5"
          >
            Done
          </button>
        </div>
      </div>

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
              <div key={w.id} className="glass flex flex-col overflow-hidden rounded-2xl p-4 relative">
                <div className="mb-2 flex items-center justify-between gap-2">
                  <p className="truncate text-xs font-medium text-slate-400">
                    {queries.find((q) => q.id === w.saved_query_id)?.name || w.saved_query_id} · {w.chart_type}
                  </p>
                  <div className="flex items-center gap-1" onMouseDown={(e) => e.stopPropagation()}>
                    <button
                      onClick={() => {
                        setEditingWidget(w);
                        setWidgetConfig({
                          saved_query_id: w.saved_query_id,
                          chart_type: w.chart_type,
                          x_field: (w.config as any)?.x_field || "",
                          y_field: (w.config as any)?.y_field || "",
                          horizon: (w.config as any)?.horizon || 10,
                        });
                      }}
                      className="p-1 text-slate-500 hover:text-white transition text-sm"
                    >
                      ⚙️
                    </button>
                    <button
                      onClick={() => handleRemoveWidget(w.id)}
                      className="p-1 text-slate-500 hover:text-red-400 transition text-sm"
                    >
                      ✕
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </Grid>
        </>
      )}

      {dashboard.widgets.length === 0 && (
        <p className="text-sm text-slate-500">No widgets yet. Click "Add Widget" to start.</p>
      )}

      {editingWidget && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-base-900 p-6 rounded-2xl max-w-md w-full border border-white/10">
            <h2 className="text-xl font-bold mb-4">
              {editingWidget.id ? "Edit Widget" : "Add Widget"}
            </h2>
            <div className="space-y-3">
              <div>
                <label className="text-xs text-slate-400 block mb-1">Saved Query</label>
                <select
                  value={widgetConfig.saved_query_id}
                  onChange={(e) => setWidgetConfig({...widgetConfig, saved_query_id: e.target.value})}
                  className="w-full p-2 border border-white/10 rounded bg-base-800 text-white text-sm"
                >
                  <option value="">Select a query</option>
                  {queries.map((q) => (
                    <option key={q.id} value={q.id}>{q.name}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="text-xs text-slate-400 block mb-1">Chart Type</label>
                <select
                  value={widgetConfig.chart_type}
                  onChange={(e) => setWidgetConfig({...widgetConfig, chart_type: e.target.value as ChartType})}
                  className="w-full p-2 border border-white/10 rounded bg-base-800 text-white text-sm"
                >
                  {CHART_TYPES.map((c) => (
                    <option key={c} value={c}>{c}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="text-xs text-slate-400 block mb-1">X‑axis Field</label>
                <input
                  placeholder="e.g. product"
                  value={widgetConfig.x_field}
                  onChange={(e) => setWidgetConfig({...widgetConfig, x_field: e.target.value})}
                  className="w-full p-2 border border-white/10 rounded bg-base-800 text-white text-sm"
                />
              </div>
              <div>
                <label className="text-xs text-slate-400 block mb-1">Y‑axis Field</label>
                <input
                  placeholder="e.g. sales_count"
                  value={widgetConfig.y_field}
                  onChange={(e) => setWidgetConfig({...widgetConfig, y_field: e.target.value})}
                  className="w-full p-2 border border-white/10 rounded bg-base-800 text-white text-sm"
                />
              </div>
              <div>
                <label className="text-xs text-slate-400 block mb-1">Forecast Horizon (steps)</label>
                <input
                  type="number"
                  placeholder="10"
                  value={widgetConfig.horizon || 10}
                  onChange={(e) =>
                    setWidgetConfig({
                      ...widgetConfig,
                      horizon: parseInt(e.target.value) || 10,
                    })
                  }
                  className="w-full p-2 border border-white/10 rounded bg-base-800 text-white text-sm"
                />
              </div>
            </div>
            <div className="flex justify-end gap-2 mt-4">
              <button
                onClick={() => setEditingWidget(null)}
                className="px-4 py-2 text-sm text-slate-400 hover:text-white"
              >
                Cancel
              </button>
              <button
                onClick={handleSaveWidget}
                className="bg-accent-gradient px-4 py-2 rounded text-white text-sm font-semibold"
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}