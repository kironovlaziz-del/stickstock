"use client";
import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import Sidebar from "@/components/Sidebar";
import TopBar from "@/components/TopBar";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [userEmail, setUserEmail] = useState("");
  const [checked, setChecked] = useState(false);

  useEffect(() => {
    fetch("/api/me", { credentials: "include" })
      .then(async r => {
        if (!r.ok) {
          router.replace("/login?redirectedFrom=" + encodeURIComponent(pathname));
          return;
        }
        const d = await r.json();
        setUserEmail(d.username || d.email || "");
        setChecked(true);
      })
      .catch(() => router.replace("/login"));
  }, [pathname]);

  if (!checked) return null;

  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <TopBar userEmail={userEmail} />
        <main className="flex-1 overflow-auto p-6">{children}</main>
      </div>
    </div>
  );
}
