"use client";

import { Breadcrumbs, PageHeader } from "@/components/console/page";
import { BarChart, Progress } from "@/components/console/ui";
import { useI18n } from "@/components/console/i18n-provider";

export default function CostPage() {
  const { t } = useI18n();
  return (
    <>
      <Breadcrumbs items={[{ label: t("cost.title") }]} />
      <PageHeader
        title={t("cost.title")}
        subtitle={t("cost.subtitle")}
        actions={(
          <>
            <button className="btn" type="button">{t("common.export")}</button>
            <button className="btn primary" type="button">{t("common.newQuota")}</button>
          </>
        )}
      />
      <div className="kpis">
        {[
          ["cost.month", "US$ 8.420", "+18%"],
          ["cost.cacheSavings", "US$ 3.910", "-32%"],
          ["cost.budget", "62%", "US$ 13.500"],
          ["cost.projection", "US$ 13.580", t("cost.aboveTarget")]
        ].map(([key, value, delta]) => (
          <div className="card kpi" key={key}>
            <div className="label">{t(key)}</div>
            <div className="value num">{value}</div>
            <div className="delta">{delta}</div>
          </div>
        ))}
      </div>

      <div className="grid-2">
        <section className="card">
          <h2>{t("cost.byDay")}</h2>
          <BarChart values={[30, 38, 34, 48, 42, 58, 52, 64, 60, 72, 68, 76]} />
        </section>
        <section className="card">
          <h2>{t("cost.byProvider")}</h2>
          <div className="list">
            <div className="list-item"><span>OpenAI</span><span className="num">US$ 148 · 47%</span></div>
            <div className="list-item"><span>Anthropic</span><span className="num">US$ 96 · 31%</span></div>
            <div className="list-item"><span>Gemini</span><span className="num">US$ 42 · 13%</span></div>
            <div className="list-item"><span>{t("common.others")}</span><span className="num">US$ 26 · 9%</span></div>
          </div>
        </section>
      </div>

      <section className="card" style={{ marginTop: 16 }}>
        <h2>{t("cost.quotas")}</h2>
        <table className="table">
          <thead><tr><th>{t("common.name")}</th><th>{t("cost.metric")}</th><th>{t("common.limit")}</th><th>{t("common.used")}</th><th>{t("common.status")}</th></tr></thead>
          <tbody>
            {[
              ["tenant", t("cost.costValue"), "US$ 13.5k", 62, "ok"],
              ["key: prod-web", t("cost.tokensValue"), "50M", 96, "warn"],
              ["key: etl-batch", t("cost.tokensValue"), "10M", 48, "ok"]
            ].map(row => (
              <tr key={String(row[0])}>
                <td>{row[0]}</td>
                <td>{row[1]}</td>
                <td className="num">{row[2]}</td>
                <td><Progress value={Number(row[3])} tone={Number(row[3]) > 90 ? "bad" : Number(row[3]) > 75 ? "warn" : "ok"} /></td>
                <td><span className={`chip ${row[4] === "ok" ? "ok" : "warn"}`}>{row[3]}%</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </>
  );
}
