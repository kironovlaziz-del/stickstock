"use client";

import { useState } from "react";
import { api } from "@/lib/api";

interface TemplateEditorProps {
  dataSourceId?: string;
  initialTemplate?: string;
  onRender?: (sql: string, params: Record<string, unknown>) => void;
}

export default function TemplateEditor({
  dataSourceId,
  initialTemplate = "",
  onRender
}: TemplateEditorProps) {
  const [template, setTemplate] = useState(initialTemplate);
  const [context, setContext] = useState<Record<string, unknown>>({});
  const [renderedSQL, setRenderedSQL] = useState("");
  const [params, setParams] = useState<Record<string, unknown>>({});
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [showVariables, setShowVariables] = useState(false);

  const findVariables = (text: string): string[] => {
    const matches = text.match(/\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}/g);
    if (!matches) return [];
    return matches.map(m => {
      const match = m.match(/\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}/);
      return match ? match[1] : "";
    }).filter(Boolean);
  };

  const variables = findVariables(template);

    const handleRender = async () => {
    console.log("Template:", template);
    console.log("Context:", context);
    // Ensure all variables have values
    const vars = findVariables(template);
    const missing = vars.filter(v => !(v in context) || context[v] === "");
    if (missing.length > 0) {
      setError("Missing values for variables: " + missing.join(", "));
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const response = await api.renderTemplate(template, context);
      const sql = response.rendered_sql || "";
      setRenderedSQL(sql);
      setParams(response.params);
      if (onRender && sql.trim() !== "") {
        onRender(sql, response.params);
      } else if (onRender && sql.trim() === "") {
        setError("Rendered SQL is empty. Please check your template and variables.");
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-4">
      <div>
        <label className="block text-xs font-medium text-slate-400 mb-1">
          SQL Template (Jinja2 syntax)
        </label>
        <textarea
          value={template}
          onChange={(e) => setTemplate(e.target.value)}
          rows={8}
          className="w-full rounded-lg border border-white/10 bg-base-900 p-3 font-mono text-sm outline-none focus:ring-2 focus:ring-accent-blue"
          placeholder="SELECT * FROM users WHERE age > {{ min_age }} {% if status %} AND status = '{{ status }}' {% endif %}"
        />
        <div className="mt-1 flex flex-wrap gap-2 text-xs text-slate-500">
          <span>Variables: {'{{ variable }}'}</span>
          <span>Conditions: {'{% if ... %}'}</span>
          <span>Filters: {'{{ value | quote }}'}</span>
        </div>
      </div>

      <div>
        <button
          onClick={() => setShowVariables(!showVariables)}
          className="text-xs text-accent-blue hover:underline"
        >
          {showVariables ? "Hide" : "Show"} variables
        </button>
        {showVariables && (
          <div className="mt-2 glass rounded-lg p-4 space-y-2">
            <p className="text-xs text-slate-400 font-medium">Variables found:</p>
            {variables.length === 0 ? (
              <p className="text-xs text-slate-500">No variables found</p>
            ) : (
              variables.map((v) => (
                <div key={v} className="flex items-center gap-2">
                  <label className="text-xs font-mono text-slate-300 w-24">{v}</label>
                  <input
                    type="text"
                    value={String(context[v] || "")}
                    onChange={(e) => setContext({ ...context, [v]: e.target.value })}
                    placeholder={"Value for " + v}
                    className="flex-1 rounded border border-white/10 bg-base-800 px-2 py-1 text-xs outline-none focus:ring-1 focus:ring-accent-blue"
                  />
                </div>
              ))
            )}
          </div>
        )}
      </div>

      <div className="flex gap-2">
        <button
          onClick={handleRender}
          disabled={loading || !template}
          className="focus-ring rounded-lg bg-accent-gradient px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
        >
          {loading ? "..." : "Render SQL"}
        </button>
      </div>

      {error && (
        <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">
          {error}
        </div>
      )}

      {renderedSQL && (
        <div className="glass rounded-lg p-4">
          <div className="flex items-center justify-between mb-2">
            <p className="text-xs font-medium text-slate-400">Rendered SQL</p>
            <button
              onClick={() => navigator.clipboard.writeText(renderedSQL)}
              className="text-xs text-accent-blue hover:underline"
            >
              Copy
            </button>
          </div>
          <pre className="overflow-x-auto rounded bg-base-800 p-3 font-mono text-xs text-slate-300">
            {renderedSQL}
          </pre>
          {Object.keys(params).length > 0 && (
            <div className="mt-2">
              <p className="text-xs text-slate-400">Parameters:</p>
              <pre className="overflow-x-auto rounded bg-base-800 p-2 font-mono text-xs text-slate-500">
                {JSON.stringify(params, null, 2)}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}