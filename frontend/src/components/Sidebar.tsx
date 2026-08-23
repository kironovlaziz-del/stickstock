"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useI18n } from "@/lib/i18n";
import { api } from "@/lib/api";

const NAV = [
  { href: "/dashboards", key: "nav.dashboards", icon: "▦" },
  { href: "/queries", key: "nav.queries", icon: "⌁" },
  { href: "/connections", key: "nav.connections", icon: "◈" },
  { href: "/reports", key: "nav.reports", icon: "📅" },
];

export default function Sidebar() {
  const pathname = usePathname();
  const { t } = useI18n();
  const [user, setUser] = useState<{ email: string; first_name?: string; last_name?: string; avatar_url?: string } | null>(null);

  useEffect(() => {
    api.me()
      .then((profile) => setUser(profile))
      .catch(() => {});
  }, []);

  const displayName = user?.first_name || user?.last_name || user?.email || "User";

  return (
    <aside className="glass flex w-60 flex-col gap-1 border-r border-white/5 p-4">
      {/* Profile at the top */}
      <Link
        href="/profile"
        className="mb-4 flex items-center gap-3 rounded-lg px-2 py-2 hover:bg-white/5 transition-colors"
      >
        {user?.avatar_url ? (
          <img src={user.avatar_url} alt="" className="h-8 w-8 rounded-full object-cover" />
        ) : (
          <div className="h-8 w-8 rounded-full bg-accent-gradient flex items-center justify-center text-sm font-bold text-white">
            {displayName.charAt(0).toUpperCase()}
          </div>
        )}
        <div className="flex flex-col overflow-hidden">
          <span className="truncate text-sm font-medium">{displayName}</span>
          <span className="truncate text-xs text-slate-400">{user?.email || ""}</span>
        </div>
      </Link>

      <div className="h-px bg-white/5 mb-2" />

      {NAV.map((item) => {
        const active = pathname?.startsWith(item.href);
        return (
          <Link
            key={item.href}
            href={item.href}
            className={`focus-ring flex items-center gap-3 rounded-lg border-l-2 px-3 py-2 text-sm transition-colors ${
              active
                ? "border-accent-blue bg-white/5 font-semibold text-white"
                : "border-transparent text-slate-400 hover:bg-white/5 hover:text-slate-200"
            }`}
          >
            <span aria-hidden>{item.icon}</span>
            {t(item.key)}
          </Link>
        );
      })}
    </aside>
  );
}