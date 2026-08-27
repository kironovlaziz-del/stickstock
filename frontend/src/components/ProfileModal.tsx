"use client";

import { useState } from "react";

interface ColumnProfile {
  name: string;
  data_type: string;
  null_count: number;
  null_percentage: number;
  unique_count: number;
  unique_percentage: number;
  min?: any;
  max?: any;
  mean?: number;
  std?: number;
  quantiles?: Record<string, number>;
  top_values?: Record<string, number>;
}

interface ProfileData {
  columns: ColumnProfile[];
  total_rows: number;
}

interface ProfileModalProps {
  isOpen: boolean;
  data: ProfileData | null;
  onClose: () => void;
  tableName: string;
}

export default function ProfileModal({ isOpen, data, onClose, tableName }: ProfileModalProps) {
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 20;

  if (!isOpen || !data) return null;

  const totalPages = Math.ceil(data.columns.length / pageSize);
  const paginatedColumns = data.columns.slice((currentPage - 1) * pageSize, currentPage * pageSize);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="glass w-full max-w-5xl max-h-[90vh] overflow-auto rounded-2xl p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-bold tracking-tight">
            Table Profile: <span className="text-accent-blue">{tableName}</span>
          </h2>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white transition text-2xl"
          >
            ✕
          </button>
        </div>

        <p className="text-sm text-slate-400 mb-4">
          Total rows: <span className="font-semibold text-white">{data.total_rows.toLocaleString()}</span>
          &nbsp;· Columns: <span className="font-semibold text-white">{data.columns.length}</span>
        </p>

        <div className="overflow-x-auto max-h-[500px] overflow-y-auto">
          <table className="w-full text-sm text-left border-collapse">
            <thead className="text-xs uppercase text-slate-400 bg-base-800/50 sticky top-0 z-10">
              <tr>
                <th className="px-3 py-2 sticky left-0 bg-base-800/90">Column</th>
                <th className="px-3 py-2">Type</th>
                <th className="px-3 py-2 text-right">Null %</th>
                <th className="px-3 py-2 text-right">Unique %</th>
                <th className="px-3 py-2 text-right">Min</th>
                <th className="px-3 py-2 text-right">Max</th>
                <th className="px-3 py-2 text-right">Mean</th>
                <th className="px-3 py-2 text-right">Std Dev</th>
                <th className="px-3 py-2">Quantiles</th>
                <th className="px-3 py-2">Top Values</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5">
              {paginatedColumns.map((col) => (
                <tr key={col.name} className="hover:bg-white/5 transition">
                  <td className="px-3 py-2 font-mono text-xs sticky left-0 bg-base-950/90">
                    {col.name}
                  </td>
                  <td className="px-3 py-2 text-xs text-slate-300">{col.data_type}</td>
                  <td className="px-3 py-2 text-right text-xs">
                    {col.null_percentage.toFixed(1)}%
                  </td>
                  <td className="px-3 py-2 text-right text-xs">
                    {col.unique_percentage.toFixed(1)}%
                  </td>
                  <td className="px-3 py-2 text-right text-xs">
                    {col.min !== undefined ? (typeof col.min === 'number' ? col.min.toFixed(2) : col.min) : '-'}
                  </td>
                  <td className="px-3 py-2 text-right text-xs">
                    {col.max !== undefined ? (typeof col.max === 'number' ? col.max.toFixed(2) : col.max) : '-'}
                  </td>
                  <td className="px-3 py-2 text-right text-xs">
                    {col.mean !== undefined ? col.mean.toFixed(2) : '-'}
                  </td>
                  <td className="px-3 py-2 text-right text-xs">
                    {col.std !== undefined ? col.std.toFixed(2) : '-'}
                  </td>
                  <td className="px-3 py-2 text-xs">
                    {col.quantiles ? (
                      <span className="text-slate-400">
                        {Object.entries(col.quantiles).map(([q, v]) => (
                          <span key={q} className="inline-block mr-1">
                            {q}: {v.toFixed(2)}
                          </span>
                        ))}
                      </span>
                    ) : '-'}
                  </td>
                  <td className="px-3 py-2 text-xs">
                    {col.top_values ? (
                      <span className="text-slate-400">
                        {Object.entries(col.top_values).map(([val, count]) => (
                          <span key={val} className="inline-block mr-2">
                            {val} ({count})
                          </span>
                        ))}
                      </span>
                    ) : '-'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {data.columns.length > pageSize && (
          <div className="mt-4 flex items-center justify-between">
            <span className="text-xs text-slate-400">
              {paginatedColumns.length} of {data.columns.length} columns
            </span>
            <div className="flex gap-2">
              <button
                disabled={currentPage <= 1}
                onClick={() => setCurrentPage(p => Math.max(p - 1, 1))}
                className="px-3 py-1 text-xs rounded border border-white/10 disabled:opacity-30 hover:bg-white/5"
              >
                Prev
              </button>
              <button
                disabled={currentPage >= totalPages}
                onClick={() => setCurrentPage(p => Math.min(p + 1, totalPages))}
                className="px-3 py-1 text-xs rounded border border-white/10 disabled:opacity-30 hover:bg-white/5"
              >
                Next
              </button>
            </div>
          </div>
        )}

        <div className="mt-4 flex justify-end">
          <button
            onClick={onClose}
            className="focus-ring rounded-lg bg-accent-gradient px-4 py-2 text-sm font-semibold text-white hover:opacity-80"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
