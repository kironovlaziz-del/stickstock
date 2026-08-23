"use client";

import { useEffect, useState, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { DashboardDetail, Collaborator } from "@/lib/types";

export default function DashboardSharePage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [dashboard, setDashboard] = useState<DashboardDetail | null>(null);
  const [shareUrl, setShareUrl] = useState<string | null>(null);
  const [collaborators, setCollaborators] = useState<Collaborator[]>([]);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<"editor" | "viewer">("viewer");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [copied, setCopied] = useState(false);

  const load = useCallback(async () => {
    const d = await api.getDashboard(params.id);
    setDashboard(d);
    setCollaborators(await api.listCollaborators(params.id));
  }, [params.id]);

  useEffect(() => {
    load().catch((e) => setError(e instanceof Error ? e.message : String(e)));
  }, [load]);

  async function handleCreateLink() {
    setBusy(true);
    setError(null);
    try {
      const { share_token } = await api.createShareLink(params.id);
      setShareUrl(`${window.location.origin}/shared/${share_token}`);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function handleRevokeLink() {
    setBusy(true);
    setError(null);
    try {
      await api.revokeShareLink(params.id);
      setShareUrl(null);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  function handleCopy() {
    if (!shareUrl) return;
    navigator.clipboard.writeText(shareUrl).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  async function handleAddCollaborator(e: React.FormEvent) {
    e.preventDefault();
    if (!email) return;
    setError(null);
    try {
      await api.addCollaborator(params.id, { email, role });
      setEmail("");
      setCollaborators(await api.listCollaborators(params.id));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function handleRemoveCollaborator(userId: string) {
    try {
      await api.removeCollaborator(params.id, userId);
      setCollaborators(await api.listCollaborators(params.id));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  if (!dashboard) return <p className="text-sm text-slate-500">Loading...</p>;

  if (dashboard.role !== "owner") {
    return (
      <div className="mx-auto max-w-2xl">
        <div className="rounded-lg border border-down/30 bg-down/10 px-3 py-2 text-sm text-down">
          Only the owner can manage sharing for this dashboard.
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-extrabold tracking-tight">Share — {dashboard.name}</h1>
        <button
          onClick={() => router.push(`/dashboards/${dashboard.id}`)}
          className="focus-ring rounded-lg border border-white/10 px-4 py-2 text-sm font-medium hover:bg-white/5"
        >
          Done
        </button>
      </div>

      {error && (
        <div className="mb-4 rounded-lg border border-down/30 bg-down/10 px-3 py-2 text-sm text-down">{error}</div>
      )}

      <div className="glass mb-6 rounded-2xl p-5">
        <p className="mb-1 text-sm font-semibold">Public link</p>
        <p className="mb-4 text-xs text-slate-500">
          Anyone with this link can view the dashboard — no account needed. Query results are fetched
          live server-side; the underlying connection credentials are never exposed.
        </p>

        {dashboard.share_enabled && !shareUrl && (
          <p className="mb-3 text-xs text-up">Sharing is currently on for this dashboard.</p>
        )}

        {shareUrl && (
          <div className="mb-3 flex items-center gap-2">
            <input
              readOnly
              value={shareUrl}
              className="flex-1 rounded-lg border border-white/10 bg-base-900 px-3 py-2 font-mono text-xs outline-none"
            />
            <button
              onClick={handleCopy}
              className="focus-ring rounded-lg border border-white/10 px-3 py-2 text-xs font-medium hover:bg-white/5"
            >
              {copied ? "Copied!" : "Copy"}
            </button>
          </div>
        )}

        <div className="flex gap-2">
          <button
            onClick={handleCreateLink}
            disabled={busy}
            className="focus-ring rounded-lg bg-accent-gradient px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
          >
            {dashboard.share_enabled ? "Regenerate link" : "Create share link"}
          </button>
          {dashboard.share_enabled && (
            <button
              onClick={handleRevokeLink}
              disabled={busy}
              className="focus-ring rounded-lg border border-down/30 px-4 py-2 text-sm font-medium text-down hover:bg-down/10 disabled:opacity-50"
            >
              Revoke
            </button>
          )}
        </div>
      </div>

      <div className="glass rounded-2xl p-5">
        <p className="mb-1 text-sm font-semibold">Collaborators</p>
        <p className="mb-4 text-xs text-slate-500">
          They need an existing account on this app — invite by the email they signed up with.
        </p>

        <form onSubmit={handleAddCollaborator} className="mb-4 flex items-center gap-2">
          <input
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="teammate@example.com"
            className="focus-ring flex-1 rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
          />
          <select
            value={role}
            onChange={(e) => setRole(e.target.value as "editor" | "viewer")}
            className="focus-ring rounded-lg border border-white/10 bg-base-900 px-2 py-2 text-sm outline-none"
          >
            <option value="viewer">Viewer</option>
            <option value="editor">Editor</option>
          </select>
          <button
            type="submit"
            className="focus-ring rounded-lg border border-white/10 px-3 py-2 text-sm font-medium hover:bg-white/5"
          >
            Add
          </button>
        </form>

        <div className="space-y-2">
          {collaborators.map((c) => (
            <div key={c.id} className="flex items-center justify-between rounded-lg border border-white/10 px-3 py-2">
              <div>
                <p className="text-sm">{c.email}</p>
                <p className="text-xs text-slate-500">{c.role}</p>
              </div>
              <button
                onClick={() => handleRemoveCollaborator(c.id)}
                className="focus-ring rounded-md border border-down/30 px-2 py-1 text-xs text-down hover:bg-down/10"
              >
                Remove
              </button>
            </div>
          ))}
          {collaborators.length === 0 && <p className="text-xs text-slate-500">No collaborators yet.</p>}
        </div>
      </div>
    </div>
  );
}
