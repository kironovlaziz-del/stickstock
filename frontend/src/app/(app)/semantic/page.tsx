"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";

interface Metric {
  id: string;
  name: string;
  description?: string;
  expression: string;
  data_source_id: string;
  table: string;
  table_name?: string; // добавим опционально
  created_at: string;
}

interface Dataset {
  id: string;
  name: string;
  description?: string;
  columns: string[];
  data_source_id: string;
  table: string;
  table_name?: string;
  filters?: any;
  created_at: string;
}

export default function SemanticPage() {
  const [activeTab, setActiveTab] = useState<"metrics" | "datasets">("metrics");
  const [metrics, setMetrics] = useState<Metric[]>([]);
  const [datasets, setDatasets] = useState<Dataset[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [formMode, setFormMode] = useState<"create" | "edit">("create");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState<any>({});

  async function loadMetrics() {
    try {
      const data = await api.semantic.metrics.list();
      setMetrics(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function loadDatasets() {
    try {
      const data = await api.semantic.datasets.list();
      setDatasets(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function loadAll() {
    setLoading(true);
    setError(null);
    await Promise.all([loadMetrics(), loadDatasets()]);
    setLoading(false);
  }

  useEffect(() => {
    loadAll();
  }, []);

  const handleMetricSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      if (formMode === "create") {
        await api.semantic.metrics.create(formData);
      } else {
        await api.semantic.metrics.create(formData);
      }
      setShowForm(false);
      setFormData({});
      await loadMetrics();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const handleMetricDelete = async (id: string) => {
    if (!confirm("Delete this metric?")) return;
    try {
      await api.semantic.metrics.delete(id);
      await loadMetrics();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const handleDatasetSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await api.semantic.datasets.create(formData);
      setShowForm(false);
      setFormData({});
      await loadDatasets();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const handleDatasetDelete = async (id: string) => {
    if (!confirm("Delete this dataset?")) return;
    try {
      await api.semantic.datasets.delete(id);
      await loadDatasets();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const renderForm = () => {
    const isMetric = activeTab === "metrics";
    const onSubmit = isMetric ? handleMetricSubmit : handleDatasetSubmit;
    const fields = isMetric ? [
      { name: "name", label: "Name", required: true, hint: "A short name for the metric/dataset." },
      { name: "description", label: "Description", required: false, hint: "Optional description." },
      { name: "expression", label: "Expression (SQL)", required: true, hint: "SQL expression, e.g. SUM(price * quantity).", placeholder: "SUM(total_amount)" },
      { name: "data_source_id", label: "Data Source ID", required: true, hint: "UUID of the data source from Connections.", placeholder: "UUID" },
      { name: "table", label: "Table", required: true, hint: "Table name including schema, e.g. demo.orders.", placeholder: "demo.orders" },
    ] : [
      { name: "name", label: "Name", required: true, hint: "A short name for the dataset." },
      { name: "description", label: "Description", required: false, hint: "Optional description." },
      { name: "columns", label: "Columns (comma separated)", required: true, hint: "Comma-separated list of column names.", placeholder: "order_id, customer_id" },
      { name: "data_source_id", label: "Data Source ID", required: true, hint: "UUID of the data source from Connections.", placeholder: "UUID" },
      { name: "table", label: "Table", required: true, hint: "Table name including schema, e.g. demo.orders.", placeholder: "demo.orders" },
    ];

    return (
      <form onSubmit={onSubmit} className="space-y-4">
        {fields.map((field) => (
          <div key={field.name}>
            <label className="block text-xs font-medium text-slate-400">
              {field.label} {field.required && "*"}
            </label>
            <input
              required={field.required}
              value={formData[field.name] || ""}
              onChange={(e) => {
                if (field.name === "columns") {
                  setFormData({ ...formData, columns: e.target.value.split(",").map(s => s.trim()) });
                } else {
                  setFormData({ ...formData, [field.name]: e.target.value });
                }
              }}
              placeholder={field.placeholder || ""}
              className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
            />
            <p className="text-xs text-slate-500 mt-1">{field.hint}</p>
          </div>
        ))}
        <div className="flex justify-end gap-2">
          <button
            type="button"
            onClick={() => setShowForm(false)}
            className="px-4 py-2 text-sm text-slate-400 hover:text-white"
          >
            Cancel
          </button>
          <button
            type="submit"
            className="focus-ring rounded-lg bg-accent-gradient px-4 py-2 text-sm font-semibold text-white"
          >
            Save
          </button>
        </div>
      </form>
    );
  };

  if (loading) return <p className="text-sm text-slate-500">Loading...</p>;

  return (
    <div className="mx-auto max-w-5xl">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-extrabold tracking-tight">Semantic Layer</h1>
        <button
          onClick={() => {
            setFormMode("create");
            setFormData({});
            setShowForm(true);
          }}
          className="focus-ring rounded-lg bg-accent-gradient px-4 py-2 text-sm font-semibold text-white"
        >
          + New {activeTab === "metrics" ? "Metric" : "Dataset"}
        </button>
      </div>

      <div className="flex gap-4 border-b border-white/10 mb-6">
        <button
          className={`pb-2 text-sm font-medium transition ${activeTab === "metrics" ? "border-b-2 border-accent-blue text-white" : "text-slate-400 hover:text-white"}`}
          onClick={() => setActiveTab("metrics")}
        >
          Metrics
        </button>
        <button
          className={`pb-2 text-sm font-medium transition ${activeTab === "datasets" ? "border-b-2 border-accent-blue text-white" : "text-slate-400 hover:text-white"}`}
          onClick={() => setActiveTab("datasets")}
        >
          Datasets
        </button>
      </div>

      {error && (
        <div className="mb-4 rounded-lg border border-down/30 bg-down/10 px-3 py-2 text-sm text-down">
          {error}
        </div>
      )}

      {showForm && (
        <div className="glass mb-6 rounded-2xl p-5">
          <h2 className="text-lg font-bold mb-4">
            {formMode === "create" ? "Create" : "Edit"} {activeTab === "metrics" ? "Metric" : "Dataset"}
          </h2>
          {renderForm()}
        </div>
      )}

      {activeTab === "metrics" ? (
        <div className="glass rounded-2xl p-5">
          {metrics.length === 0 ? (
            <p className="text-sm text-slate-500">No metrics defined yet.</p>
          ) : (
            <div className="space-y-4">
              {metrics.map((m) => (
                <div key={m.id} className="flex items-center justify-between border-b border-white/5 py-2">
                  <div>
                    <p className="font-medium">{m.name}</p>
                    <p className="text-xs text-slate-400">{m.expression}</p>
                    <p className="text-xs text-slate-500">Table: {m.table_name}</p>
                  </div>
                  <button
                    onClick={() => handleMetricDelete(m.id)}
                    className="text-sm font-medium text-down hover:text-down/80 transition"
                  >
                    Delete
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      ) : (
        <div className="glass rounded-2xl p-5">
          {datasets.length === 0 ? (
            <p className="text-sm text-slate-500">No datasets defined yet.</p>
          ) : (
            <div className="space-y-4">
              {datasets.map((d) => (
                <div key={d.id} className="flex items-center justify-between border-b border-white/5 py-2">
                  <div>
                    <p className="font-medium">{d.name}</p>
                    <p className="text-xs text-slate-400">Columns: {d.columns.join(", ")}</p>
                    <p className="text-xs text-slate-500">Table: {d.table_name}</p>
                  </div>
                  <button
                    onClick={() => handleDatasetDelete(d.id)}
                    className="text-sm font-medium text-down hover:text-down/80 transition"
                  >
                    Delete
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
