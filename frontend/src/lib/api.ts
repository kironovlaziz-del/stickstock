import { createClient } from "@/lib/supabase/client";
import type {
  Profile,
  DataSource,
  SavedQuery,
  QueryVersion,
  QueryResult,
  DashboardSummary,
  DashboardDetail,
  Widget,
  TableInfo,
  LayoutItem,
} from "@/lib/types";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL ?? "/api";

async function authHeader(): Promise<Record<string, string>> {
  const supabase = createClient();
  const {
    data: { session },
  } = await supabase.auth.getSession();
  return session ? { Authorization: `Bearer ${session.access_token}` } : {};
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = {
    "Content-Type": "application/json",
    ...(await authHeader()),
    ...(init.headers ?? {}),
  };

  const res = await fetch(`${API_BASE}${path}`, { ...init, headers });

  if (!res.ok) {
    let message = `request failed with status ${res.status}`;
    try {
      const body = await res.json();
      if (body?.error) message = body.error;
    } catch {
      // response wasn't JSON — keep the generic message
    }
    throw new Error(message);
  }

  if (res.status === 204) {
    return undefined as T;
  }
  return res.json() as Promise<T>;
}

// downloadBlob triggers a browser download for endpoints that return a
// file rather than JSON (CSV/XLSX/PDF export) — these need the auth
// header attached manually since a plain <a href> download can't carry
// an Authorization header, so the file has to be fetched as a blob first.
async function downloadBlob(path: string, init: RequestInit, filenameFallback: string): Promise<void> {
  const headers = {
    ...(await authHeader()),
    ...(init.headers ?? {}),
  };

  const res = await fetch(`${API_BASE}${path}`, { ...init, headers });
  if (!res.ok) {
    let message = `export failed with status ${res.status}`;
    try {
      const body = await res.json();
      if (body?.error) message = body.error;
    } catch {
      // response wasn't JSON — keep the generic message
    }
    throw new Error(message);
  }

  const blob = await res.blob();
  const disposition = res.headers.get("Content-Disposition") ?? "";
  const match = disposition.match(/filename="([^"]+)"/);
  const filename = match ? match[1] : filenameFallback;

  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

export type ExportFormat = "csv" | "xlsx" | "pdf";

export const api = {
  me: () => request<Profile>("/me"),
  updateLocale: (locale: string) =>
    request<void>("/me", { method: "PUT", body: JSON.stringify({ locale }) }),

  listDataSources: () => request<DataSource[]>("/datasources"),
  createDataSource: (body: { name: string; kind: string; dsn: string }) =>
    request<DataSource>("/datasources", { method: "POST", body: JSON.stringify(body) }),
  getSchema: (dataSourceId: string) =>
    request<{ tables: TableInfo[] }>(`/datasources/${dataSourceId}/schema`),

  listQueries: () => request<SavedQuery[]>("/queries"),
  getQuery: (id: string) => request<SavedQuery>(`/queries/${id}`),
  createQuery: (body: { data_source_id: string; name: string; sql_text: string }) =>
    request<SavedQuery>("/queries", { method: "POST", body: JSON.stringify(body) }),
  updateQuery: (id: string, body: { name?: string; sql_text: string }) =>
    request<{ id: string; version: number }>(`/queries/${id}`, {
      method: "PUT",
      body: JSON.stringify(body),
    }),
  deleteQuery: (id: string) => request<void>(`/queries/${id}`, { method: "DELETE" }),
  queryVersions: (id: string) => request<QueryVersion[]>(`/queries/${id}/versions`),
  runAdHoc: (body: { data_source_id: string; sql: string; params?: Record<string, unknown> }) =>
    request<QueryResult>("/queries/run", { method: "POST", body: JSON.stringify(body) }),
  runSaved: (id: string, params?: Record<string, unknown>) =>
    request<QueryResult>(`/queries/${id}/run`, {
      method: "POST",
      body: JSON.stringify({ params: params ?? {} }),
    }),
  exportSavedQuery: (id: string, format: ExportFormat) =>
    downloadBlob(`/queries/${id}/export?format=${format}`, { method: "GET" }, `export.${format}`),
  exportAdHoc: (body: { data_source_id: string; sql: string; format: ExportFormat; title?: string }) =>
    downloadBlob(
      "/export/query",
      { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) },
      `export.${body.format}`
    ),

  listDashboards: () => request<DashboardSummary[]>("/dashboards"),
  getDashboard: (id: string) => request<DashboardDetail>(`/dashboards/${id}`),
  createDashboard: (body: { name: string; layout?: LayoutItem[] }) =>
    request<DashboardDetail>("/dashboards", { method: "POST", body: JSON.stringify(body) }),
  updateDashboard: (id: string, body: { name?: string; layout?: LayoutItem[] }) =>
    request<void>(`/dashboards/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteDashboard: (id: string) => request<void>(`/dashboards/${id}`, { method: "DELETE" }),
  addWidget: (
    dashboardId: string,
    body: { saved_query_id: string; chart_type: string; config?: Record<string, unknown> }
  ) =>
    request<Widget>(`/dashboards/${dashboardId}/widgets`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  deleteWidget: (dashboardId: string, widgetId: string) =>
    request<void>(`/dashboards/${dashboardId}/widgets/${widgetId}`, { method: "DELETE" }),

  osintLookup: (body: { type: "whois" | "dns" | "breach"; target: string }) =>
    request<Record<string, unknown>>("/osint/lookup", { method: "POST", body: JSON.stringify(body) }),
};
