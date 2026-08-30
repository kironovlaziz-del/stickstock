"use client";

import { useState, useEffect, useRef } from "react";
import { api } from "@/lib/api";
import type { Comment } from "@/lib/types";

export default function CommentThread({
  dashboardId,
  widgetId,
  readOnly = false,
}: {
  dashboardId: string;
  widgetId: string;
  readOnly?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const [comments, setComments] = useState<Comment[] | null>(null);
  const [body, setBody] = useState("");
  const [busy, setBusy] = useState(false);
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (readOnly) {
      api.listComments(dashboardId, widgetId).then(setComments);
    }
  }, [readOnly, dashboardId, widgetId]);

  // Закрывать при клике вне панели
  useEffect(() => {
    if (!open) return;
    function handleClick(e: MouseEvent) {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [open]);

  async function load() {
    setComments(await api.listComments(dashboardId, widgetId));
  }

  async function toggle() {
    const next = !open;
    setOpen(next);
    if (next && comments === null) await load();
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!body.trim()) return;
    setBusy(true);
    try {
      await api.createComment(dashboardId, widgetId, body.trim());
      setBody("");
      await load();
    } finally {
      setBusy(false);
    }
  }

  async function handleDelete(commentId: string) {
    await api.deleteComment(commentId);
    await load();
  }

  // ── Read-only: просто показываем комментарии без кнопок ───
  if (readOnly) {
    if (!comments || comments.length === 0) return null;
    return (
      <div className="mt-2 border-t border-white/5 pt-2 space-y-1">
        {comments.map((c) => (
          <div key={c.id} className="rounded-lg bg-base-900/60 px-3 py-2">
            <p className="text-xs text-slate-300">{c.body}</p>
          </div>
        ))}
      </div>
    );
  }

  // ── Обычный режим: кнопка + всплывающая панель ────────────
  const count = comments?.length ?? 0;

  return (
    <div className="relative" ref={panelRef}>
      {/* Кнопка — не занимает места внутри виджета */}
      <button
        onClick={toggle}
        className="mt-2 flex items-center gap-1 text-xs text-slate-500 hover:text-slate-300 transition focus:outline-none"
      >
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <path d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z"/>
        </svg>
        {comments !== null ? `${count} comment${count === 1 ? "" : "s"}` : "Comments"}
      </button>

      {/* Панель — абсолютная, не сжимает виджет */}
      {open && (
        <div className="absolute bottom-full left-0 right-0 z-50 mb-1 rounded-xl border border-white/10 bg-base-900 shadow-2xl p-3 space-y-2"
          style={{ maxHeight: 260, overflowY: "auto" }}>

          {comments === null ? (
            <p className="text-xs text-slate-500">Loading...</p>
          ) : (
            <>
              {comments.length === 0 && (
                <p className="text-xs text-slate-500">No comments yet.</p>
              )}
              {comments.map((c) => (
                <div key={c.id} className="flex items-start justify-between gap-2 rounded-lg bg-base-800 px-3 py-2">
                  <p className="text-xs text-slate-300 break-words flex-1">{c.body}</p>
                  <button
                    onClick={() => handleDelete(c.id)}
                    className="shrink-0 text-xs text-slate-600 hover:text-red-400 transition"
                    title="Delete"
                  >✕</button>
                </div>
              ))}
            </>
          )}

          <form onSubmit={handleSubmit} className="flex gap-2 pt-1 border-t border-white/5">
            <input
              value={body}
              onChange={(e) => setBody(e.target.value)}
              placeholder="Add a comment..."
              className="flex-1 rounded-lg border border-white/10 bg-base-800 px-2 py-1.5 text-xs outline-none focus:border-accent-blue"
            />
            <button
              type="submit"
              disabled={busy || !body.trim()}
              className="rounded-lg border border-white/10 px-3 py-1.5 text-xs font-medium hover:bg-white/5 disabled:opacity-40 transition"
            >
              Post
            </button>
          </form>
        </div>
      )}
    </div>
  );
}
