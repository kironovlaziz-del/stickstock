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
  const [isAdmin, setIsAdmin] = useState(false);

  useEffect(() => {
    api
      .me()
      .then((profile) => setIsAdmin(profile.is_admin))
      .catch(() => {});
  }, []);

  const items = isAdmin ? [...NAV, { href: "/admin", key: "nav.admin", icon: "🛡️" }] : NAV;

  return (
    <aside className="glass flex w-60 flex-col gap-1 border-r border-white/5 p-4">
      <div className="mb-6 flex items-center gap-2 px-2">
        <div className="h-7 w-7 rounded-lg bg-accent-gradient" />
        <span className="text-base font-bold tracking-tight">{t("app.name")}</span>
      </div>

      {items.map((item) => {
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
