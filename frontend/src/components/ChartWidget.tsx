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

  // line / heatmap / boxplot / treemap: a line chart is a reasonable
  // default render until those get dedicated implementations — see
  // README roadmap. Better to show *something* correct than nothing.
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
