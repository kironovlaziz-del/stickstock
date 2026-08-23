"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";

export default function ProfilePage() {
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [title, setTitle] = useState("");
  const [email, setEmail] = useState("");
  const [avatarUrl, setAvatarUrl] = useState("");

  useEffect(() => {
    api.me()
      .then((profile) => {
        setFirstName(profile.first_name || "");
        setLastName(profile.last_name || "");
        setTitle(profile.title || "");
        setEmail(profile.email || "");
        setAvatarUrl(profile.avatar_url || "");
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError(null);
    setSuccess(false);
    try {
      await api.updateProfile({
        first_name: firstName,
        last_name: lastName,
        title: title,
        avatar_url: avatarUrl,
      });
      setSuccess(true);
      router.refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <p className="text-sm text-slate-500">Loading...</p>;

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="text-2xl font-extrabold tracking-tight mb-6">My Profile</h1>

      <form onSubmit={handleSubmit} className="glass space-y-4 rounded-2xl p-6">
        {error && (
          <div className="rounded-lg border border-down/30 bg-down/10 px-3 py-2 text-sm text-down">
            {error}
          </div>
        )}
        {success && (
          <div className="rounded-lg border border-up/30 bg-up/10 px-3 py-2 text-sm text-up">
            Profile updated successfully!
          </div>
        )}

        <div>
          <label className="mb-1 block text-xs font-medium text-slate-400">Email</label>
          <input
            value={email}
            disabled
            className="w-full rounded-lg border border-white/10 bg-base-800 px-3 py-2 text-sm text-slate-400 outline-none cursor-not-allowed"
          />
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-400">First Name</label>
            <input
              value={firstName}
              onChange={(e) => setFirstName(e.target.value)}
              placeholder="John"
              className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
            />
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-400">Last Name</label>
            <input
              value={lastName}
              onChange={(e) => setLastName(e.target.value)}
              placeholder="Doe"
              className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
            />
          </div>
        </div>

        <div>
          <label className="mb-1 block text-xs font-medium text-slate-400">Job Title / Role</label>
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Data Analyst"
            className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
          />
        </div>

        <div>
          <label className="mb-1 block text-xs font-medium text-slate-400">Avatar URL</label>
          <input
            value={avatarUrl}
            onChange={(e) => setAvatarUrl(e.target.value)}
            placeholder="https://example.com/avatar.jpg"
            className="focus-ring w-full rounded-lg border border-white/10 bg-base-900 px-3 py-2 text-sm outline-none"
          />
          {avatarUrl && (
            <img src={avatarUrl} alt="Avatar preview" className="mt-2 h-16 w-16 rounded-full object-cover" />
          )}
        </div>

        <button
          type="submit"
          disabled={saving}
          className="focus-ring rounded-lg bg-accent-gradient px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
        >
          {saving ? "Saving..." : "Save Profile"}
        </button>
      </form>
    </div>
  );
}
