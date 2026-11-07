"use client";

import { Breadcrumbs, PageHeader } from "./page";
import { useI18n } from "./i18n-provider";

export function ComingSoon({ titleKey }: { titleKey: string }) {
  const { t } = useI18n();
  return (
    <>
      <Breadcrumbs items={[{ label: t(titleKey) }]} />
      <PageHeader title={t(titleKey)} subtitle={t("common.comingSoonBody")} />
      <div className="empty-state">
        <div className="empty-icon" aria-hidden="true">+</div>
        <h2>{t("common.comingSoon")}</h2>
        <p>{t("common.comingSoonBody")}</p>
      </div>
    </>
  );
}
