"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Search } from "lucide-react";
import { LOCALES, type Locale } from "@/lib/i18n";
import { useI18n } from "./i18n-provider";

function guideSlug(pathname: string): string {
  const section = pathname.split("/")[1] || "dashboard";
  if (section === "providers") return "providers";
  return section;
}

export function Topbar() {
  const pathname = usePathname();
  const { locale, setLocale, t, guideUrl } = useI18n();
  return (
    <header className="topbar">
      <div className="search" role="search">
        <Search aria-hidden="true" size={16} />
        <input aria-label={t("common.searchPlaceholder")} placeholder={t("common.searchPlaceholder")} />
        <kbd>⌘K</kbd>
      </div>
      <div className="who">
        <select
          className="locale-select"
          aria-label={t("common.language")}
          value={locale}
          onChange={event => setLocale(event.target.value as Locale)}
        >
          {LOCALES.map(item => (
            <option key={item.code} value={item.code}>{item.name}</option>
          ))}
        </select>
        <Link className="btn help" href={guideUrl(guideSlug(pathname))} target="_blank" rel="noopener noreferrer">
          {t("common.help")}
        </Link>
        <span className="tenant">{t("common.tenantDefault")}</span>
        <span className="chip">{t("common.production")}</span>
        <span className="avatar">AD</span>
      </div>
    </header>
  );
}
