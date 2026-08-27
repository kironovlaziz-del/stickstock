"use client";

import {
  LineChart,
  Line,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  ScatterChart,
  Scatter,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import type { QueryResult, ChartType } from "@/lib/types";

const PALETTE = ["#4f7cff", "#9b5de5", "#22d3c5", "#5ebd8b", "#e0707a", "#f5b942"];
const TOOLTIP_STYLE = { background: "#161d2e", border: "1px solid rgba(255,255,255,0.1)" };

function toRows(result: QueryResult): Record<string, unknown>[] {
  return result.rows.map((row) => {
    const obj: Record<string, unknown> = {};
    result.columns.forEach((col, i) => {
      obj[col] = row[i];
    });
    return obj;
  });
}

export default function ChartWidget({
  result,
  chartType,
}: {
  result: QueryResult;
  chartType: ChartType;
}) {
  if (!result.rows.length) {
    return <p className="text-sm text-slate-500">No data for this query yet.</p>;
  }

  const data = toRows(result);
  const [xKey, ...restKeys] = result.columns;
  const yKeys = restKeys.length > 0 ? restKeys : [xKey];

  // ---------- KPI Card ----------
  if (chartType === "kpi") {
    const firstRow = result.rows[0];
    let value: number | null = null;
    if (firstRow) {
      for (const cell of firstRow) {
        const num = typeof cell === "number" ? cell : parseFloat(String(cell));
        if (!isNaN(num)) {
          value = num;
          break;
        }
      }
    }
    console.log("KPI result:", result);
    console.log("KPI value:", value);
    return (
      <div className="flex items-center justify-center h-full min-h-[120px] p-4 bg-base-800/30 rounded-xl border border-white/10">
        <div className="text-5xl font-bold text-white text-center">
          {value !== null ? value.toLocaleString() : "—"}
        </div>
      </div>
    );
  }

  // ---------- Table ----------
  if (chartType === "table") {
    return (
      <div className="overflow-auto">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-white/10 text-slate-400">
              {result.columns.map((c) => (
                <th key={c} className="whitespace-nowrap px-3 py-2 font-medium">
                  {c}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {data.slice(0, 200).map((row, i) => (
              <tr key={i} className="border-b border-white/5">
                {result.columns.map((c) => (
                  <td key={c} className="whitespace-nowrap px-3 py-2 text-slate-200">
                    {String(row[c] ?? "")}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  }

  // ---------- Pie ----------
  if (chartType === "pie") {
    const valueKey = yKeys[0];
    return (
      <ResponsiveContainer width="100%" height={260}>
        <PieChart>
          <Pie data={data} dataKey={valueKey} nameKey={xKey} outerRadius={90} innerRadius={50}>
            {data.map((_, i) => (
              <Cell key={i} fill={PALETTE[i % PALETTE.length]} />
            ))}
          </Pie>
          <Tooltip contentStyle={TOOLTIP_STYLE} />
        </PieChart>
      </ResponsiveContainer>
    );
  }

  // ---------- Bar ----------
  if (chartType === "bar") {
    return (
      <ResponsiveContainer width="100%" height={260}>
        <BarChart data={data}>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.06)" />
          <XAxis dataKey={xKey} stroke="#8b93a7" fontSize={12} />
          <YAxis stroke="#8b93a7" fontSize={12} />
          <Tooltip contentStyle={TOOLTIP_STYLE} />
          {yKeys.map((k, i) => (
            <Bar key={k} dataKey={k} fill={PALETTE[i % PALETTE.length]} radius={[4, 4, 0, 0]} />
          ))}
        </BarChart>
      </ResponsiveContainer>
    );
  }

  // ---------- Scatter ----------
  if (chartType === "scatter") {
    return (
      <ResponsiveContainer width="100%" height={260}>
        <ScatterChart>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.06)" />
          <XAxis dataKey={xKey} stroke="#8b93a7" fontSize={12} />
          <YAxis dataKey={yKeys[0]} stroke="#8b93a7" fontSize={12} />
          <Tooltip contentStyle={TOOLTIP_STYLE} />
          <Scatter data={data} fill={PALETTE[0]} />
        </ScatterChart>
      </ResponsiveContainer>
    );
  }

  // ---------- Forecast ----------
  if (chartType === "forecast") {
    return (
      <ResponsiveContainer width="100%" height={260}>
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.06)" />
          <XAxis dataKey={xKey} stroke="#8b93a7" fontSize={12} />
          <YAxis stroke="#8b93a7" fontSize={12} />
          <Tooltip contentStyle={TOOLTIP_STYLE} />
          {yKeys.map((k, i) => {
            const isForecast = k.toLowerCase().includes("forecast") || k.toLowerCase().includes("predicted");
            return (
              <Line
                key={k}
                type="monotone"
                dataKey={k}
                stroke={PALETTE[i % PALETTE.length]}
                strokeWidth={2}
                strokeDasharray={isForecast ? "5 5" : "0"}
                dot={false}
              />
            );
          })}
        </LineChart>
      </ResponsiveContainer>
    );
  }

  // ---------- Default: Line ----------
  return (
    <ResponsiveContainer width="100%" height={260}>
      <LineChart data={data}>
        <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.06)" />
        <XAxis dataKey={xKey} stroke="#8b93a7" fontSize={12} />
        <YAxis stroke="#8b93a7" fontSize={12} />
        <Tooltip contentStyle={TOOLTIP_STYLE} />
        {yKeys.map((k, i) => (
          <Line
            key={k}
            type="monotone"
            dataKey={k}
            stroke={PALETTE[i % PALETTE.length]}
            strokeWidth={2}
            dot={false}
          />
        ))}
      </LineChart>
    </ResponsiveContainer>
  );
}
