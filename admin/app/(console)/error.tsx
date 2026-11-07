"use client";

import { useI18n } from "@/components/console/i18n-provider";

export default function ConsoleError({ reset }: { error: Error; reset: () => void }) {
  const { t } = useI18n();
  return (
    <div className="empty-state">
      <div className="empty-icon" aria-hidden="true">!</div>
      <h2>{t("common.error")}</h2>
      <p>{t("common.retry")}</p>
      <button className="btn primary" type="button" onClick={reset}>{t("common.retry")}</button>
    </div>
  );
}
