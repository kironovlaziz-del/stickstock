"use client";

import { Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import Link from "next/link";

function SharedAnalyticsContent() {
  const searchParams = useSearchParams();
  const [data, setData] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const encoded = searchParams.get("data");
    if (!encoded) {
      setError("No analysis data found in the link.");
      return;
    }
    try {
      const decoded = decodeURIComponent(encoded);
      const parsed = JSON.parse(decoded);
      setData(parsed);
    } catch (e) {
      setError("Invalid analysis data. The link may be corrupted.");
    }
  }, [searchParams]);

  if (error) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-base-950 px-4">
        <div className="glass max-w-md rounded-2xl p-6 text-center">
          <p className="text-down">{error}</p>
          <Link href="/" className="mt-4 inline-block text-accent-blue hover:underline">
            Go to Dashboard
          </Link>
        </div>
      </main>
    );
  }

  if (!data) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-base-950 text-slate-400">
        Loading...
      </main>
    );
  }

  const { type, result, timestamp } = data;

  const renderResult = () => {
    switch (type) {
      case "regression": {
        const { intercept, coefficients, r_squared, p_values } = result;
        return (
          <div className="space-y-2 text-sm">
            <p><span className="text-slate-400">Equation:</span> y = {intercept.toFixed(4)} + {coefficients.map((c: number, i: number) => `${c.toFixed(4)} * x${i+1}`).join(" + ")}</p>
            <p><span className="text-slate-400">R²:</span> {r_squared.toFixed(4)}</p>
            <p><span className="text-slate-400">p-values:</span> {p_values.map((p: number) => p.toFixed(4)).join(", ")}</p>
          </div>
        );
      }
      case "forecast": {
        const { forecast } = result;
        return (
          <div>
            <p className="text-sm text-slate-400 mb-2">Forecast (next {forecast.length} steps):</p>
            <div className="flex flex-wrap gap-2">
              {forecast.map((v: number, i: number) => (
                <span key={i} className="bg-base-800 px-3 py-1 rounded-lg text-sm font-mono">
                  {v.toFixed(2)}
                </span>
              ))}
            </div>
          </div>
        );
      }
      case "anomalies": {
        const { is_outlier, scores } = result;
        const outliers = is_outlier.map((o: boolean, i: number) => ({ index: i, is_outlier: o, score: scores[i] }));
        return (
          <div>
            <p className="text-sm text-slate-400 mb-2">Anomaly detection results:</p>
            <div className="max-h-60 overflow-y-auto space-y-1">
              {outliers.slice(0, 20).map((item: any) => (
                <div key={item.index} className="text-xs flex justify-between border-b border-white/5 py-1">
                  <span>Row {item.index + 1}</span>
                  <span className={item.is_outlier ? "text-down font-bold" : "text-slate-400"}>
                    {item.is_outlier ? "⚠️ Outlier" : "Normal"} (score: {item.score.toFixed(4)})
                  </span>
                </div>
              ))}
            </div>
          </div>
        );
      }
      case "ttest": {
        const { statistic, p_value, mean_a, mean_b } = result;
        return (
          <div className="space-y-2 text-sm">
            <p><span className="text-slate-400">Mean A:</span> {mean_a.toFixed(4)}</p>
            <p><span className="text-slate-400">Mean B:</span> {mean_b.toFixed(4)}</p>
            <p><span className="text-slate-400">t-statistic:</span> {statistic.toFixed(4)}</p>
            <p><span className="text-slate-400">p-value:</span> {p_value.toFixed(4)}</p>
          </div>
        );
      }
      default:
        return <pre className="font-mono text-xs text-slate-300 whitespace-pre-wrap">{JSON.stringify(result, null, 2)}</pre>;
    }
  };

  return (
    <main className="min-h-screen bg-base-950 p-6 text-slate-100">
      <div className="mx-auto max-w-3xl">
        <div className="mb-6 flex items-center justify-between">
          <h1 className="text-2xl font-extrabold tracking-tight">Shared Analysis</h1>
          <Link href="/" className="text-sm text-accent-blue hover:underline">
            Back to Dashboard
          </Link>
        </div>
        <div className="glass rounded-2xl p-6">
          <p className="text-xs text-slate-500 mb-4">
            Analysis type: <span className="font-mono uppercase">{type}</span>
            {timestamp && ` · ${new Date(timestamp).toLocaleString()}`}
          </p>
          {renderResult()}
        </div>
      </div>
    </main>
  );
}

export default function SharedAnalyticsPage() {
  return (
    <Suspense fallback={<div className="flex min-h-screen items-center justify-center bg-base-950 text-slate-400">Loading...</div>}>
      <SharedAnalyticsContent />
    </Suspense>
  );
}