"use client";

import { useRouter } from "next/navigation";
import { createClient } from "@/lib/supabase/client";
import { useI18n } from "@/lib/i18n";

const LOCALES = [
  { code: "en", label: "EN" },
  { code: "ru", label: "RU" },
  { code: "uz", label: "UZ" },
  { code: "kk", label: "KK" },
  { code: "tg", label: "TG" },
];

export default function TopBar({ userEmail }: { userEmail: string }) {
  const router = useRouter();
  const { locale, setLocale } = useI18n();

  async function handleLogout() {
    const supabase = createClient();
    await supabase.auth.signOut();
    router.push("/login");
    router.refresh();
  }

  return (
    <header className="glass flex items-center justify-between border-b border-white/5 px-6 py-3">
      <div />
      <div className="flex items-center gap-4">
        <select
          value={locale}
          onChange={(e) => setLocale(e.target.value)}
          className="focus-ring rounded-md border border-white/10 bg-base-900 px-2 py-1 text-xs"
          aria-label="Language"
        >
          {LOCALES.map((l) => (
            <option key={l.code} value={l.code}>
              {l.label}
            </option>
          ))}
        </select>
        <span className="text-sm text-slate-400">{userEmail}</span>
        <button
          onClick={handleLogout}
          className="focus-ring rounded-md border border-white/10 px-3 py-1.5 text-xs font-medium text-slate-300 hover:bg-white/5"
        >
          Log out
        </button>
      </div>
    </header>
  );
}
