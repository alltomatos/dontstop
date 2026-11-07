import { Sidebar } from "@/components/console/sidebar";
import { Topbar } from "@/components/console/topbar";
import { Onboarding } from "@/components/console/onboarding";

export default function ConsoleLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="app">
      <Sidebar />
      <div className="main">
        <Topbar />
        <main className="content">{children}</main>
      </div>
      <Onboarding />
    </div>
  );
}
