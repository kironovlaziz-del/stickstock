export interface Profile {
  id: string;
  locale: string;
  created_at: string;
}

export interface DataSource {
  id: string;
  name: string;
  kind: string;
}

export interface SavedQuery {
  id: string;
  data_source_id: string;
  name: string;
  sql_text?: string;
  version: number;
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
  | "treemap";

export interface LayoutItem {
  i: string; // widget id
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface Widget {
  id: string;
  dashboard_id: string;
  saved_query_id: string;
  chart_type: ChartType;
  config: Record<string, unknown>;
}

export interface DashboardSummary {
  id: string;
  name: string;
  created_at: string;
}

export interface DashboardDetail {
  id: string;
  name: string;
  layout: LayoutItem[];
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
