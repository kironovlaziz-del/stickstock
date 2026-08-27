"use client";

import Image from "next/image";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useI18n } from "@/lib/i18n";

const NAV = [
  { href: "/semantic", label: "Semantic", icon: "🧠" },
  { href: "/lineage", label: "Lineage", icon: "🔗" },
  { href: "/dashboards", label: "Dashboards", icon: "▦" },
  { href: "/queries", label: "Queries", icon: "⌁" },
  { href: "/connections", label: "Connections", icon: "◈" },
  { href: "/reports", label: "Reports", icon: "📅" },
];

export default function Sidebar() {
  const pathname = usePathname();
  const { t } = useI18n();

  return (
    <aside className="glass flex w-60 flex-col gap-1 border-r border-white/5 p-4">
      <div className="mb-6 px-2">
        <Image
          src="logotip.png"
          alt="Stickstock"
          width={150}
          height={40}
          priority
          unoptimized
          style={{ objectFit: "contain" }}
        />
      </div>

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
            {item.label}
          </Link>
        );
      })}
    </aside>
  );
}