"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Activity,
  CircleDollarSign,
  Database,
  KeyRound,
  LayoutDashboard,
  Network,
  Plug,
  Server,
  Settings,
  ShieldCheck,
  Waypoints
} from "lucide-react";
import { useI18n } from "./i18n-provider";

const items = [
  { href: "/dashboard", key: "nav.dashboard", icon: LayoutDashboard },
  { href: "/providers", key: "nav.providers", icon: Server },
  { href: "/proxies", key: "nav.proxies", icon: Network },
  { href: "/endpoints", key: "nav.endpoints", icon: Waypoints },
  { href: "/api-manager", key: "nav.apiManager", icon: KeyRound },
  { href: "/mcp", key: "nav.mcp", icon: Plug },
  { href: "/cache", key: "nav.cache", icon: Database },
  { href: "/dlp", key: "nav.dlp", icon: ShieldCheck },
  { href: "/observability", key: "nav.observability", icon: Activity },
  { href: "/cost", key: "nav.cost", icon: CircleDollarSign },
  { href: "/settings", key: "nav.settings", icon: Settings }
] as const;

export function Sidebar() {
  const pathname = usePathname();
  const { t } = useI18n();
  return (
    <aside className="sidebar" aria-label={t("nav.primary")}>
      <Link className="brand" href="/dashboard">
        <span className="brand-dot" />
        DonStop IA
      </Link>
      <nav className="sidebar-nav">
        {items.map(item => {
          const active = pathname === item.href || pathname.startsWith(`${item.href}/`);
          const Icon = item.icon;
          return (
            <Link
              className="nav"
              href={item.href}
              key={item.href}
              aria-current={active ? "page" : undefined}
            >
              <Icon aria-hidden="true" size={18} strokeWidth={1.8} />
              {t(item.key)}
            </Link>
          );
        })}
      </nav>
      <div className="sidebar-foot">dontstop · v0.1</div>
    </aside>
  );
}
