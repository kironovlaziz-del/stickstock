import { cookies } from "next/headers";
import Sidebar from "@/components/Sidebar";
import TopBar from "@/components/TopBar";

export default async function AppLayout({ children }: { children: React.ReactNode }) {
  // Проверяем наличие токена в куке (если он там есть) или просто пропускаем
  // Вся защита теперь на бекенде, поэтому просто рендерим страницу
  // Если хотите защитить страницу от неавторизованных, можно сделать запрос к /api/me,
  // но для простоты пропускаем.
  return (
    <div className="flex min-h-screen">
      <Sidebar />
      <div className="flex flex-1 flex-col">
        <TopBar userEmail="user@example.com" />
        <main className="flex-1 overflow-y-auto p-6">{children}</main>
      </div>
    </div>
  );
}
