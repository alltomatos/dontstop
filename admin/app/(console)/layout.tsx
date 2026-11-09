import { redirect } from "next/navigation";
import { Sidebar } from "@/components/console/sidebar";
import { Topbar } from "@/components/console/topbar";
import { consoleSessionState } from "@/lib/admin-auth";

export default async function ConsoleLayout({ children }: { children: React.ReactNode }) {
  const session = await consoleSessionState();
  if (session === "unauthenticated") redirect("/login");
  return (
    <div className="app">
      <Sidebar />
      <div className="main">
        <Topbar />
        <main className="content">{children}</main>
      </div>
    </div>
  );
}
