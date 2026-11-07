"use client";

import { Breadcrumbs, PageHeader } from "@/components/console/page";
import { useI18n } from "@/components/console/i18n-provider";
import { LOCALES, type Locale } from "@/lib/i18n";

export default function SettingsPage() {
  const { locale, setLocale, t } = useI18n();
  return (
    <>
      <Breadcrumbs items={[{ label: t("nav.settings") }]} />
      <PageHeader title={t("nav.settings")} subtitle={t("settings.subtitle")} />
      <section className="card">
        <h2>{t("settings.language")}</h2>
        <p className="muted">{t("settings.languageHint")}</p>
        <fieldset className="language-grid" style={{ marginTop: 16 }}>
          <legend className="sr-only">{t("settings.language")}</legend>
          {LOCALES.map(item => (
            <label className="language-card" key={item.code}>
              <input
                type="radio"
                name="locale"
                value={item.code}
                checked={locale === item.code}
                onChange={() => setLocale(item.code as Locale)}
              />
              <span>
                <strong>{item.name}</strong>
                <span>{item.code}</span>
              </span>
            </label>
          ))}
        </fieldset>
        <p className="muted" style={{ marginTop: 16 }}>{t("settings.persistenceHint")}</p>
      </section>
    </>
  );
}
