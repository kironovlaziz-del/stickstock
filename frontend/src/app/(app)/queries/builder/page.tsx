"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import type { DataSource, TableInfo, QueryResult } from "@/lib/types";

interface FilterRow {
  column: string;
  op: string;
  value: string;
}

interface AggRow {
  column: string;
  func: string;
  alias: string;
}

const OPS = ["=", "!=", ">", ">=", "<", "<=", "LIKE"];
const FUNCS = ["SUM", "AVG", "COUNT", "MIN", "MAX"];

function buildSQL(
  table: string,
  columns: string[],
  filters: FilterRow[],
  groupBy: string[],
  aggregations: AggRow[],
  orderBy: string,
  orderDir: string,
  limit: number
): { sql: string; params: Record<string, unknown> } {
  const params: Record<string, unknown> = {};
  const selectParts: string[] = [];

  if (groupBy.length > 0 || aggregations.length > 0) {
    selectParts.push(...groupBy);
    aggregations.forEach((a) => {
      if (!a.column || !a.func) return;
      selectParts.push(`${a.func}(${a.column})${a.alias ? ` AS ${a.alias}` : ""}`);
    });
  } else {
    selectParts.push(...(columns.length > 0 ? columns : ["*"]));
  }
  if (selectParts.length === 0) selectParts.push("*");

  let sql = `SELECT ${selectParts.join(", ")} FROM ${table || "?"}`;

  const validFilters = filters.filter((f) => f.column && f.value !== "");
  if (validFilters.length > 0) {
    const conditions = validFilters.map((f, i) => {
      const paramName = `filter_${i}`;
      params[paramName] = f.value;
      return `${f.column} ${f.op} :${paramName}`;
    });
    sql += ` WHERE ${conditions.join(" AND ")}`;
  }

  if (groupBy.length > 0) {
    sql += ` GROUP BY ${groupBy.join(", ")}`;
  }
  if (orderBy) {
    sql += ` ORDER BY ${orderBy} ${orderDir}`;
  }
  if (limit > 0) {
    sql += ` LIMIT ${limit}`;
  }

  return { sql, params };
}

export default function QueryBuilderPage() {
  const [sources, setSources] = useState<DataSource[]>([]);
  const [dataSourceId, setDataSourceId] = useState("");
  const [tables, setTables] = useState<TableInfo[]>([]);
  const [tableName, setTableName] = useState("");
  const [selectedColumns, setSelectedColumns] = useState<string[]>([]);
  const [filters, setFilters] = useState<FilterRow[]>([]);
  const [groupBy, setGroupBy] = useState<string[]>([]);
  const [aggregations, setAggregations] = useState<AggRow[]>([]);
  const [orderBy, setOrderBy] = useState("");
  const [orderDir, setOrderDir] = useState("ASC");
  const [limit, setLimit] = useState(100);
  const [result, setResult] = useState<QueryResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loadingSchema, setLoadingSchema] = useState(false);
  const [running, setRunning] = useState(false);
  const [saveName, setSaveName] = useState("");

  // Пагинация
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 20;
  const totalPages = result ? Math.ceil(result.rows.length / pageSize) : 0;
  const paginatedRows = result ? result.rows.slice((currentPage - 1) * pageSize, currentPage * pageSize) : [];

  useEffect(() => {
    api
      .listDataSources()
      .then((ds) => {
        setSources(ds);
        if (ds[0]) setDataSourceId(ds[0].id);
      })
      .catch(() => {});
  }, []);

  useEffect(() => {
    if (!dataSourceId) return;
    setLoadingSchema(true);
    setError(null);
    api
      .getSchema(dataSourceId)
      .then((res) => {
        setTables(res.tables ?? []);
        setTableName("");
        setSelectedColumns([]);
      })
      .catch((e) => setError(e instanceof Error ? e.message : String(e)))
      .finally(() => setLoadingSchema(false));
  }, [dataSourceId]);

  const currentTable = tables.find((tb) => tb.name === tableName);
  const columnNames = currentTable?.columns.map((c) => c.name) ?? [];
  const qualifiedTable =
    currentTable && currentTable.schema && currentTable.schema !== "public"
      ? `${currentTable.schema}.${currentTable.name}`
      : tableName;

  const { sql, params } = buildSQL(
    qualifiedTable,
    selectedColumns,
    filters,
    groupBy,
    aggregations,
    orderBy,
    orderDir,
    limit
  );

  function toggleColumn(col: string) {
    setSelectedColumns((prev) => (prev.includes(col) ? prev.filter((c) => c !== col) : [...prev, col]));
  }

  function toggleGroupBy(col: string) {
    setGroupBy((prev) => (prev.includes(col) ? prev.filter((c) => c !== col) : [...prev, col]));
  }

  async function handleRun() {
    if (!tableName) {
      setError("Pick a table first.");
      return;
    }
    if (!dataSourceId) {
      setError("Pick a data source.");
      return;
    }
    setRunning(true);
    setError(null);
    try {
      const response = await api.runAdHoc({ data_source_id: dataSourceId, sql, params });
      setResult(response);
      setCurrentPage(1);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setRunning(false);
    }
  }

  function buildStandaloneSQL(): string {
    const validFilters = filters.filter((f) => f.column && f.value !== "");
    const selectParts: string[] = [];
    if (groupBy.length > 0 || aggregations.length > 0) {
      selectParts.push(...groupBy);
      aggregations.forEach((a) => {
        if (!a.column || !a.func) return;
        selectParts.push(`${a.func}(${a.column})${a.alias ? ` AS ${a.alias}` : ""}`);
      });
    } else {
      selectParts.push(...(selectedColumns.length > 0 ? selectedColumns : ["*"]));
    }
    if (selectParts.length === 0) selectParts.push("*");

    let out = `SELECT ${selectParts.join(", ")} FROM ${qualifiedTable}`;
    if (validFilters.length > 0) {
      const conditions = validFilters.map((f) => {
        const escaped = f.value.replace(/'/g, "''");
        return `${f.column} ${f.op} '${escaped}'`;
      });
      out += ` WHERE ${conditions.join(" AND ")}`;
    }
    if (groupBy.length > 0) out += ` GROUP BY ${groupBy.join(", ")}`;
    if (orderBy) out += ` ORDER BY ${orderBy} ${orderDir}`;
    if (limit > 0) out += ` LIMIT ${limit}`;
    return out;
  }

  async function handleSave() {
    if (!saveName || !tableName) return;
    try {
      const standalone = buildStandaloneSQL();
      await api.createQuery({ data_source_id: dataSourceId, name: saveName, sql_text: standalone });
      setSaveName("");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div className="mx-auto max-w-5xl">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-extrabold tracking-tight">Visual query builder</h1>
        <Link
          href="/queries"
          className="focus-ring rounded-lg border border-white/10 px-4 py-2 text-sm font-medium hover:bg-white/5"
        >
          Back to Queries
        </Link>
      </div>

      <div className="glass mb-4 rounded-2xl p-5">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-400">Connection</label>
            <select
              value={dataSourceId}
              onChange={(e) => setDataSourceId(e.target.value)}
              className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
            >
              {sources.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-400">Table</label>
            <select
              value={tableName}
              onChange={(e) => {
                setTableName(e.target.value);
                setSelectedColumns([]);
                setGroupBy([]);
              }}
              disabled={loadingSchema}
              className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
            >
              <option value="">{loadingSchema ? "Loading..." : "Select a table..."}</option>
              {tables.map((tb) => (
                <option key={`${tb.schema}.${tb.name}`} value={tb.name}>
                  {tb.schema && tb.schema !== "public" ? `${tb.schema}.${tb.name}` : tb.name}
                </option>
              ))}
            </select>
          </div>
        </div>
      </div>

      {tableName && (
        <>
          <div className="glass mb-4 rounded-2xl p-5">
            <p className="mb-3 text-sm font-semibold">Columns</p>
            <div className="flex flex-wrap gap-2">
              {columnNames.map((col) => (
                <button
                  key={col}
                  onClick={() => toggleColumn(col)}
                  className={`focus-ring rounded-full border px-3 py-1 text-xs font-medium ${
                    selectedColumns.includes(col)
                      ? "border-accent-blue bg-accent-blue/20 text-white"
                      : "border-white/10 text-slate-400 hover:bg-white/5"
                  }`}
                >
                  {col}
                </button>
              ))}
            </div>
            <p className="mt-2 text-xs text-slate-500">None selected = SELECT *</p>
          </div>

          <div className="glass mb-4 rounded-2xl p-5">
            <div className="mb-3 flex items-center justify-between">
              <p className="text-sm font-semibold">Filters</p>
              <button
                onClick={() => setFilters((f) => [...f, { column: columnNames[0] ?? "", op: "=", value: "" }])}
                className="focus-ring rounded-md border border-white/10 px-2 py-1 text-xs hover:bg-white/5"
              >
                + Add filter
              </button>
            </div>
            <div className="space-y-2">
              {filters.map((f, i) => (
                <div key={i} className="flex items-center gap-2">
                  <select
                    value={f.column}
                    onChange={(e) =>
                      setFilters((prev) => prev.map((r, idx) => (idx === i ? { ...r, column: e.target.value } : r)))
                    }
                    className="focus-ring rounded-lg border border-white/10 bg-base-900 px-2 py-1.5 text-xs outline-none"
                  >
                    {columnNames.map((col) => (
                      <option key={col} value={col}>
                        {col}
                      </option>
                    ))}
                  </select>
                  <select
                    value={f.op}
                    onChange={(e) =>
                      setFilters((prev) => prev.map((r, idx) => (idx === i ? { ...r, op: e.target.value } : r)))
                    }
                    className="focus-ring rounded-lg border border-white/10 bg-base-900 px-2 py-1.5 text-xs outline-none"
                  >
                    {OPS.map((op) => (
                      <option key={op} value={op}>
                        {op}
                      </option>
                    ))}
                  </select>
                  <input
                    value={f.value}
                    onChange={(e) =>
                      setFilters((prev) => prev.map((r, idx) => (idx === i ? { ...r, value: e.target.value } : r)))
                    }
                    placeholder="value"
                    className="focus-ring flex-1 rounded-lg border border-white/10 bg-base-900 px-2 py-1.5 text-xs outline-none"
                  />
                  <button
                    onClick={() => setFilters((prev) => prev.filter((_, idx) => idx !== i))}
                    className="focus-ring rounded-md border border-down/30 px-2 py-1 text-xs text-down hover:bg-down/10"
                  >
                    ✕
                  </button>
                </div>
              ))}
              {filters.length === 0 && <p className="text-xs text-slate-500">No filters — returns all rows.</p>}
            </div>
          </div>

          <div className="glass mb-4 grid grid-cols-1 gap-4 rounded-2xl p-5 sm:grid-cols-2">
            <div>
              <p className="mb-2 text-sm font-semibold">Group by</p>
              <div className="flex flex-wrap gap-2">
                {columnNames.map((col) => (
                  <button
                    key={col}
                    onClick={() => toggleGroupBy(col)}
                    className={`focus-ring rounded-full border px-3 py-1 text-xs font-medium ${
                      groupBy.includes(col)
                        ? "border-accent-purple bg-accent-purple/20 text-white"
                        : "border-white/10 text-slate-400 hover:bg-white/5"
                    }`}
                  >
                    {col}
                  </button>
                ))}
              </div>
            </div>
            <div>
              <div className="mb-2 flex items-center justify-between">
                <p className="text-sm font-semibold">Aggregations</p>
                <button
                  onClick={() => setAggregations((a) => [...a, { column: columnNames[0] ?? "", func: "SUM", alias: "" }])}
                  className="focus-ring rounded-md border border-white/10 px-2 py-1 text-xs hover:bg-white/5"
                >
                  + Add
                </button>
              </div>
              <div className="space-y-2">
                {aggregations.map((a, i) => (
                  <div key={i} className="flex items-center gap-2">
                    <select
                      value={a.func}
                      onChange={(e) =>
                        setAggregations((prev) => prev.map((r, idx) => (idx === i ? { ...r, func: e.target.value } : r)))
                      }
                      className="focus-ring rounded-lg border border-white/10 bg-base-900 px-2 py-1.5 text-xs outline-none"
                    >
                      {FUNCS.map((fn) => (
                        <option key={fn} value={fn}>
                          {fn}
                        </option>
                      ))}
                    </select>
                    <select
                      value={a.column}
                      onChange={(e) =>
                        setAggregations((prev) => prev.map((r, idx) => (idx === i ? { ...r, column: e.target.value } : r)))
                      }
                      className="focus-ring rounded-lg border border-white/10 bg-base-900 px-2 py-1.5 text-xs outline-none"
                    >
                      {columnNames.map((col) => (
                        <option key={col} value={col}>
                          {col}
                        </option>
                      ))}
                    </select>
                    <input
                      value={a.alias}
                      onChange={(e) =>
                        setAggregations((prev) => prev.map((r, idx) => (idx === i ? { ...r, alias: e.target.value } : r)))
                      }
                      placeholder="alias"
                      className="focus-ring w-20 rounded-lg border border-white/10 bg-base-900 px-2 py-1.5 text-xs outline-none"
                    />
                    <button
                      onClick={() => setAggregations((prev) => prev.filter((_, idx) => idx !== i))}
                      className="focus-ring rounded-md border border-down/30 px-2 py-1 text-xs text-down hover:bg-down/10"
                    >
                      ✕
                    </button>
                  </div>
                ))}
              </div>
            </div>
          </div>

          <div className="glass mb-4 grid grid-cols-1 gap-4 rounded-2xl p-5 sm:grid-cols-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-slate-400">Order by</label>
              <select
                value={orderBy}
                onChange={(e) => setOrderBy(e.target.value)}
                className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-2 py-1.5 text-xs outline-none"
              >
                <option value="">none</option>
                {columnNames.map((col) => (
                  <option key={col} value={col}>
                    {col}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-slate-400">Direction</label>
              <select
                value={orderDir}
                onChange={(e) => setOrderDir(e.target.value)}
                className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-2 py-1.5 text-xs outline-none"
              >
                <option value="ASC">ASC</option>
                <option value="DESC">DESC</option>
              </select>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-slate-400">Limit</label>
              <input
                type="number"
                value={limit}
                onChange={(e) => setLimit(Number(e.target.value))}
                className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-2 py-1.5 text-xs outline-none"
              />
            </div>
          </div>

          <div className="glass mb-4 rounded-2xl p-5">
            <p className="mb-2 text-xs font-medium uppercase tracking-wide text-slate-400">Generated SQL</p>
            <pre className="overflow-x-auto rounded-lg bg-base-900 p-3 font-mono text-xs text-slate-300">{sql}</pre>
            <div className="mt-3 flex items-center gap-2">
              <button
                onClick={handleRun}
                disabled={running}
                className="focus-ring rounded-lg bg-accent-gradient px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
              >
                {running ? "..." : "Run"}
              </button>
              <input
                value={saveName}
                onChange={(e) => setSaveName(e.target.value)}
                placeholder="New query"
                className="focus-ring flex-1 rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
              />
              <button
                onClick={handleSave}
                className="focus-ring rounded-lg border border-white/10 px-4 py-2 text-sm font-medium hover:bg-white/5"
              >
                Save
              </button>
            </div>
          </div>
        </>
      )}

      {error && (
        <div className="mb-4 rounded-lg border border-down/30 bg-down/10 px-3 py-2 text-sm text-down">{error}</div>
      )}

      {result && (
        <div className="glass rounded-2xl p-5">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-xs text-slate-400">
              {result.rows.length} rows · Page {currentPage} of {totalPages}
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
          <div className="overflow-x-auto max-h-[400px] overflow-y-auto">
            <table className="w-full text-left text-sm border-collapse">
              <thead className="sticky top-0 bg-base-900 z-10">
                <tr className="border-b border-white/10 text-slate-400">
                  {result.columns.map((c) => (
                    <th key={c} className="whitespace-nowrap px-3 py-2 font-medium border-r border-white/5 last:border-r-0">
                      {c}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {paginatedRows.map((row, i) => (
                  <tr key={i} className="border-b border-white/5 hover:bg-white/5 transition">
                    {row.map((cell, j) => (
                      <td key={j} className="whitespace-nowrap px-3 py-2 text-slate-200 border-r border-white/5 last:border-r-0">
                        {String(cell ?? "")}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {result.truncated && (
            <p className="mt-2 text-xs text-slate-500">Results truncated to the first rows.</p>
          )}
        </div>
      )}
    </div>
  );
}
