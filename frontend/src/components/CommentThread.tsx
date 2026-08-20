"use client";

import { useState } from "react";
import { api } from "@/lib/api";
import type { Comment } from "@/lib/types";

export default function CommentThread({ dashboardId, widgetId }: { dashboardId: string; widgetId: string }) {
  const [open, setOpen] = useState(false);
  const [comments, setComments] = useState<Comment[] | null>(null);
  const [body, setBody] = useState("");
  const [busy, setBusy] = useState(false);

  async function load() {
    setComments(await api.listComments(dashboardId, widgetId));
  }

  async function toggle() {
    const next = !open;
    setOpen(next);
    if (next && comments === null) {
      await load();
    }
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

  return (
    <div className="mt-3 border-t border-white/5 pt-2">
      <button
        onClick={toggle}
        className="focus-ring text-xs font-medium text-slate-400 hover:text-slate-200"
      >
        💬 {comments ? `${comments.length} comment${comments.length === 1 ? "" : "s"}` : "Comments"}
      </button>

      {open && (
        <div className="mt-2 space-y-2">
          {comments === null ? (
            <p className="text-xs text-slate-500">Loading...</p>
          ) : (
            <>
              {comments.map((c) => (
                <div key={c.id} className="flex items-start justify-between gap-2 rounded-lg bg-base-900 px-3 py-2">
                  <p className="text-xs text-slate-300">{c.body}</p>
                  <button
                    onClick={() => handleDelete(c.id)}
                    className="focus-ring shrink-0 text-xs text-slate-600 hover:text-down"
                    title="Delete (only your own comments)"
                  >
                    ✕
                  </button>
                </div>
              ))}
              {comments.length === 0 && <p className="text-xs text-slate-500">No comments yet.</p>}
            </>
          )}

          <form onSubmit={handleSubmit} className="flex gap-2">
            <input
              value={body}
              onChange={(e) => setBody(e.target.value)}
              placeholder="Add a comment..."
              className="focus-ring flex-1 rounded-lg border border-white/10 bg-base-900 px-2 py-1.5 text-xs outline-none"
            />
            <button
              type="submit"
              disabled={busy}
              className="focus-ring rounded-lg border border-white/10 px-3 py-1.5 text-xs font-medium hover:bg-white/5 disabled:opacity-50"
            >
              Post
            </button>
          </form>
        </div>
      )}
    </div>
  );
}
