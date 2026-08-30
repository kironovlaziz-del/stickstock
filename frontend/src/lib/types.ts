export type DashboardRole = "owner" | "editor" | "viewer";

export interface Profile {
  id: string;
  email: string;
  locale?: string;
  is_admin?: boolean;
  is_blocked?: boolean;
  created_at: string;
}

export interface DataSource {
  id: string;
  name: string;
  kind: string;
  ssh_host?: string;
  ssh_port?: number;
  ssh_user?: string;
}

export interface SavedQuery {
  id: string;
  data_source_id: string;
  name: string;
  sql_text?: string;
  version: number;
  last_run_at?: string;
  last_row_count?: number;
}

export interface QueryVersion {
  version: number;
  sql_text: string;
  created_at: string;
}

export interface QueryResult {
  columns: string[];
  rows: unknown[][];
  truncated: boolean;
}

export type ChartType =
  | "line"
  | "bar"
  | "pie"
  | "heatmap"
  | "table"
  | "boxplot"
  | "scatter"
  | "treemap"
  | "kpi"
  | "forecast"
  | "combo"
  | "radar"
  | "funnel"
  | "waterfall"
  | "bubble"
  | "stackedbar"
  | "area";

export interface LayoutItem {
  i: string;
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface Widget {
  id: string;
  dashboard_id: string;
  saved_query_id?: string;
  metric_id?: string;
  chart_type: ChartType;
  config: Record<string, unknown>;
}

export interface DashboardSummary {
  id: string;
  name: string;
  role: DashboardRole;
  created_at: string;
}

export interface DashboardDetail {
  id: string;
  name: string;
  layout: LayoutItem[];
  role: DashboardRole;
  share_enabled: boolean;
  created_at: string;
  widgets: Widget[];
}

export interface ColumnInfo {
  name: string;
  data_type: string;
  nullable: boolean;
}

export interface TableInfo {
  schema: string;
  name: string;
  columns: ColumnInfo[];
}

export interface Collaborator {
  id: string;
  user_id: string;
  email: string;
  role: "editor" | "viewer";
}

export interface Comment {
  id: string;
  widget_id: string;
  author_id: string;
  body: string;
  created_at: string;
}

export interface ScheduledReport {
  id: string;
  saved_query_id: string;
  cron_expr: string;
  delivery_kind: "email" | "telegram";
  delivery_target: string;
  is_active: boolean;
  last_run_at?: string;
  created_at: string;
}

export interface AdminUser {
  id: string;
  email: string;
  locale: string;
  is_admin: boolean;
  is_blocked: boolean;
  created_at: string;
}

export interface AdminStats {
  users: number;
  data_sources: number;
  saved_queries: number;
  dashboards: number;
}
