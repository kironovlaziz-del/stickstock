"use client";

import { useState, useEffect } from "react";
import { api } from "@/lib/api";
import type { QueryResult } from "@/lib/types";

type AnalysisType = "regression" | "forecast" | "anomalies" | "ttest";

interface AnalyticsPanelProps {
  result: QueryResult;
  queryId?: string;
}

export default function AnalyticsPanel({ result, queryId }: AnalyticsPanelProps) {
  const [analysisType, setAnalysisType] = useState<AnalysisType>("regression");
  const [loading, setLoading] = useState(false);
  const [output, setOutput] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    if (!queryId) return;
    const key = `analytics_${queryId}_${analysisType}`;
    const stored = localStorage.getItem(key);
    if (stored) {
      try {
        const parsed = JSON.parse(stored);
        setOutput(parsed);
        setSaved(true);
      } catch (e) {
        console.error("Failed to restore analytics result", e);
      }
    }
  }, [queryId, analysisType]);

  if (!result || !result.rows || !result.columns || result.rows.length === 0) {
    return <p className="text-sm text-slate-500">No data available for analytics.</p>;
  }

  const numericColumns = result.columns.filter((col, idx) => {
    return result.rows.some((row) => typeof row[idx] === "number");
  });

  const firstNumericCol = numericColumns[0] || "";
  const secondNumericCol = numericColumns[1] || "";

  const handleRun = async () => {
    setLoading(true);
    setError(null);
    setOutput(null);
    setSaved(false);

    try {
      let response: any;

      switch (analysisType) {
        case "regression": {
          if (numericColumns.length < 2) throw new Error("Need two numeric columns for regression.");
          const y: number[] = [];
          const x: number[][] = [];
          const yIdx = result.columns.indexOf(firstNumericCol);
          const xIdx = result.columns.indexOf(secondNumericCol);

          result.rows.forEach((row) => {
            const yVal = row[yIdx];
            const xVal = row[xIdx];
            if (typeof yVal === "number" && typeof xVal === "number") {
              y.push(yVal);
              x.push([xVal]);
            }
          });

          if (y.length < 3) throw new Error("Need at least 3 data points for regression.");
          response = await api.regression({ y, x });
          break;
        }

        case "forecast": {
          if (numericColumns.length < 1) throw new Error("Need at least one numeric column for forecast.");
          const series: number[] = [];
          const idx = result.columns.indexOf(firstNumericCol);
          result.rows.forEach((row) => {
            const val = row[idx];
            if (typeof val === "number") series.push(val);
          });
          if (series.length < 10) throw new Error("Need at least 10 data points for forecast.");
          response = await api.statsForecast({ series, periods: 5 });
          break;
        }

        case "anomalies": {
          if (numericColumns.length < 1) throw new Error("Need at least one numeric column for anomaly detection.");
          const rows: number[][] = [];
          const cols = numericColumns.slice(0, 2);
          const indices = cols.map((c) => result.columns.indexOf(c));
          result.rows.forEach((row) => {
            const vals = indices.map((i) => {
              const v = row[i];
              return typeof v === "number" ? v : 0;
            });
            rows.push(vals);
          });
          if (rows.length < 10) throw new Error("Need at least 10 rows for anomaly detection.");
          response = await api.anomalies({ columns: cols, rows, contamination: 0.05 });
          break;
        }

        case "ttest": {
          if (numericColumns.length < 2) throw new Error("Need two numeric columns for t-test.");
          const idxA = result.columns.indexOf(firstNumericCol);
          const idxB = result.columns.indexOf(secondNumericCol);
          const sampleA: number[] = [];
          const sampleB: number[] = [];
          result.rows.forEach((row) => {
            const a = row[idxA];
            const b = row[idxB];
            if (typeof a === "number") sampleA.push(a);
            if (typeof b === "number") sampleB.push(b);
          });
          if (sampleA.length < 2 || sampleB.length < 2) throw new Error("Each sample needs at least 2 values.");
          response = await api.ttest({ sample_a: sampleA, sample_b: sampleB });
          break;
        }

        default:
          throw new Error("Unsupported analysis type.");
      }

      setOutput(response);
      if (queryId) {
        const key = `analytics_${queryId}_${analysisType}`;
        localStorage.setItem(key, JSON.stringify(response));
        setSaved(true);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  };

  const handleCopyLink = () => {
    if (!output) return;
    const data = {
      type: analysisType,
      result: output,
      timestamp: new Date().toISOString(),
    };
    const encoded = encodeURIComponent(JSON.stringify(data));
    const url = `${window.location.origin}/shared/analytics?data=${encoded}`;
    navigator.clipboard.writeText(url);
    alert("Link copied to clipboard! You can share this analysis result.");
  };

  const handleClearSaved = () => {
    if (!queryId) return;
    const key = `analytics_${queryId}_${analysisType}`;
    localStorage.removeItem(key);
    setOutput(null);
    setSaved(false);
  };

  const renderResult = () => {
    if (!output) return null;

    try {
      switch (analysisType) {
        case "regression": {
          const { intercept, coefficients, r_squared, p_values } = output;
          if (intercept === undefined || !coefficients) return <p className="text-sm text-slate-500">Invalid regression result.</p>;
          return (
            <div className="space-y-1 text-sm">
              <p><span className="text-slate-400">Equation:</span> y = {intercept.toFixed(4)} + {coefficients.map((c: number, i: number) => `${c.toFixed(4)} * x${i+1}`).join(" + ")}</p>
              <p><span className="text-slate-400">R²:</span> {r_squared != null ? r_squared.toFixed(4) : "—"}</p>
              <p><span className="text-slate-400">p-values:</span> {p_values ? p_values.map((p: number) => p.toFixed(4)).join(", ") : "—"}</p>
            </div>
          );
        }
        case "forecast": {
          const { forecast } = output;
          if (!forecast || !Array.isArray(forecast)) return <p className="text-sm text-slate-500">Invalid forecast result.</p>;
          return (
            <div>
              <p className="text-sm text-slate-400 mb-1">Forecast (next {forecast.length} steps):</p>
              <div className="flex flex-wrap gap-2">
                {forecast.map((v: number, i: number) => (
                  <span key={i} className="bg-base-800 px-3 py-1 rounded-lg text-sm font-mono">
                    {typeof v === "number" ? v.toFixed(2) : "—"}
                  </span>
                ))}
              </div>
            </div>
          );
        }
        case "anomalies": {
          const { is_outlier, scores } = output;
          if (!is_outlier || !scores || !Array.isArray(is_outlier)) return <p className="text-sm text-slate-500">Invalid anomaly result.</p>;
          const outliers = is_outlier.map((o: boolean, i: number) => ({ index: i, is_outlier: o, score: scores[i] }));
          return (
            <div>
              <p className="text-sm text-slate-400 mb-1">Anomaly detection results:</p>
              <div className="max-h-40 overflow-y-auto space-y-1">
                {outliers.slice(0, 20).map((item: any) => (
                  <div key={item.index} className="text-xs flex justify-between border-b border-white/5 py-1">
                    <span>Row {item.index + 1}</span>
                    <span className={item.is_outlier ? "text-down font-bold" : "text-slate-400"}>
                      {item.is_outlier ? "⚠️ Outlier" : "Normal"} (score: {typeof item.score === "number" ? item.score.toFixed(4) : "—"})
                    </span>
                  </div>
                ))}
              </div>
            </div>
          );
        }
        case "ttest": {
          const { statistic, p_value, mean_a, mean_b } = output;
          if (statistic === undefined || p_value === undefined) return <p className="text-sm text-slate-500">Invalid t-test result.</p>;
          return (
            <div className="space-y-1 text-sm">
              <p><span className="text-slate-400">Mean A:</span> {mean_a != null ? mean_a.toFixed(4) : "—"}</p>
              <p><span className="text-slate-400">Mean B:</span> {mean_b != null ? mean_b.toFixed(4) : "—"}</p>
              <p><span className="text-slate-400">t-statistic:</span> {statistic.toFixed(4)}</p>
              <p><span className="text-slate-400">p-value:</span> {p_value.toFixed(4)}</p>
            </div>
          );
        }
        default:
          return <pre className="font-mono text-xs text-slate-300 whitespace-pre-wrap">{JSON.stringify(output, null, 2)}</pre>;
      }
    } catch (e) {
      return <p className="text-sm text-down">Error rendering results: {e instanceof Error ? e.message : String(e)}</p>;
    }
  };

  return (
    <div className="mt-6 border-t border-white/10 pt-4">
      <h3 className="mb-2 text-sm font-semibold uppercase tracking-wide text-slate-400">Analytics</h3>

      <div className="flex flex-wrap items-center gap-2">
        <select
          value={analysisType}
          onChange={(e) => setAnalysisType(e.target.value as AnalysisType)}
          className="focus-ring rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
        >
          <option value="regression">Regression</option>
          <option value="forecast">Forecast (ARIMA)</option>
          <option value="anomalies">Anomaly Detection</option>
          <option value="ttest">T-test</option>
        </select>

        <button
          onClick={handleRun}
          disabled={loading}
          className="focus-ring rounded-lg bg-accent-gradient px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
        >
          {loading ? "Running..." : "Run Analysis"}
        </button>

        {output && (
          <button
            onClick={handleCopyLink}
            className="focus-ring rounded-lg border border-white/10 px-4 py-2 text-sm font-medium hover:bg-white/5"
          >
            🔗 Share
          </button>
        )}

        {saved && (
          <button
            onClick={handleClearSaved}
            className="focus-ring rounded-lg border border-down/30 px-4 py-2 text-sm font-medium text-down hover:bg-down/10"
          >
            ✕ Clear saved
          </button>
        )}
      </div>

      {error && (
        <div className="mt-3 rounded-lg border border-down/30 bg-down/10 px-3 py-2 text-sm text-down">
          {error}
        </div>
      )}

      {output && (
        <div className="mt-3 overflow-auto rounded-lg bg-base-900 p-4">
          {renderResult()}
        </div>
      )}
    </div>
  );
}