"use client";

import { LineChart, Line, ResponsiveContainer } from "recharts";

interface KpiCardProps {
  label: string;
  value: string;
  changePct?: number;
  sparkline?: number[];
}

export default function KpiCard({ label, value, changePct, sparkline }: KpiCardProps) {
  const isUp = (changePct ?? 0) >= 0;
  const data = (sparkline ?? []).map((v, i) => ({ i, v }));

  return (
    <div className="glass relative overflow-hidden rounded-2xl p-5">
      {data.length > 1 && (
        <div className="pointer-events-none absolute inset-x-4 bottom-2 h-12 opacity-30 blur-xl" aria-hidden>
          <div className="h-full w-full bg-accent-gradient" />
        </div>
      )}

      <p className="mb-2 text-xs font-medium uppercase tracking-wide text-slate-400">{label}</p>
      <div className="flex items-end justify-between">
        <span className="text-3xl font-extrabold tracking-tight">{value}</span>
        {changePct !== undefined && (
          <span className={`text-sm font-semibold ${isUp ? "text-up" : "text-down"}`}>
            {isUp ? "↑" : "↓"} {Math.abs(changePct).toFixed(1)}%
          </span>
        )}
      </div>

      {data.length > 1 && (
        <div className="relative z-10 mt-3 h-10">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={data}>
              <Line
                type="monotone"
                dataKey="v"
                stroke={isUp ? "#5ebd8b" : "#e0707a"}
                strokeWidth={2}
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
