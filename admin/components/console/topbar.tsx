"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { LogOut, Search } from "lucide-react";
import { useI18n } from "./i18n-provider";

function guideSlug(pathname: string): string {
  const section = pathname.split("/")[1] || "dashboard";
  if (section === "providers") return "providers";
  return section;
}

export function Topbar() {
  const pathname = usePathname();
  const router = useRouter();
  const { t, guideUrl } = useI18n();
  return (
    <header className="topbar">
      <div className="search" role="search">
        <Search aria-hidden="true" size={16} />
        <input aria-label={t("common.searchPlaceholder")} placeholder={t("common.searchPlaceholder")} />
        <kbd>⌘K</kbd>
      </div>
      <div className="who">
        <Link className="btn help" href={guideUrl(guideSlug(pathname))} target="_blank" rel="noopener noreferrer">
          {t("common.help")}
        </Link>
        <button className="btn" type="button" aria-label={t("common.logout")} title={t("common.logout")} onClick={async () => { await fetch("/api/auth/logout", { method: "POST" }); router.replace("/login"); }}>
          <LogOut aria-hidden="true" size={16} />
        </button>
        <span className="avatar">AD</span>
      </div>
    </header>
  );
}
