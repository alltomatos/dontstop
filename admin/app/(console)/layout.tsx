import { Sidebar } from "@/components/console/sidebar";
import { Topbar } from "@/components/console/topbar";

export default function ConsoleLayout({ children }: { children: React.ReactNode }) {
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
