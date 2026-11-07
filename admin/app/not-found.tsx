"use client";

import Link from "next/link";
import { useI18n } from "@/components/console/i18n-provider";

export default function NotFound() {
  const { t } = useI18n();
  return (
    <div className="content">
      <div className="empty-state">
        <div className="empty-icon" aria-hidden="true">?</div>
        <h2>{t("common.error")}</h2>
        <p>{t("providers.noResultsBody")}</p>
        <Link className="btn primary" href="/dashboard">{t("nav.dashboard")}</Link>
      </div>
    </div>
  );
}
