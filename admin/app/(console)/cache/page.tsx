"use client";

import { Breadcrumbs, PageHeader } from "@/components/console/page";
import { BarChart } from "@/components/console/ui";
import { useI18n } from "@/components/console/i18n-provider";

const events = [
  ["10:44", "exact hit", "gpt-4.1", "12 ms", "1.2k"],
  ["10:43", "semantic", "claude-sonnet-4", "0,97", "2.4k"],
  ["10:41", "miss", "gemini-2.5-pro", "—", "—"],
  ["10:39", "exact hit", "gpt-4.1-mini", "8 ms", "0.8k"]
];

export default function CachePage() {
  const { t } = useI18n();
  return (
    <>
      <Breadcrumbs items={[{ label: t("nav.cache") }]} />
      <PageHeader
        title={t("nav.cache")}
        subtitle={t("cache.subtitle")}
        actions={<button className="btn" type="button">{t("common.invalidate")}</button>}
      />
      <div className="kpis">
        {[
          ["cache.hitTotal", "41,2%", "+5,1 pp"],
          ["cache.exact", "26,4%", "+2,0 pp"],
          ["cache.semantic", "14,8%", "+3,1 pp"],
          ["cache.savings24h", "US$ 1.284", t("cache.costSaving")]
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
          <h2>{t("cache.hitRateByDay")}</h2>
          <BarChart values={[38, 52, 44, 66, 58, 74, 62, 81, 70, 88, 76, 92]} />
          <BarChart values={[30, 36, 34, 42, 39, 48, 45, 55, 50, 60, 56, 64]} alt />
        </section>
        <section className="card">
          <h2>{t("cache.layers")}</h2>
          <div className="list">
            <div className="list-item"><span>{t("cache.l1")}</span><span className="mono">10 s</span></div>
            <div className="list-item"><span>{t("cache.l2")}</span><span className="mono">1 h</span></div>
            <div className="list-item"><span>{t("cache.semanticLayer")}</span><span className="mono">24 h · 0,95</span></div>
          </div>
          <div className="actions" style={{ marginTop: 12 }}>
            <button className="btn" type="button">{t("common.editTtl")}</button>
            <button className="btn" type="button">{t("common.reindex")}</button>
          </div>
        </section>
      </div>

      <section className="card" style={{ marginTop: 16 }}>
        <h2>{t("cache.recentEvents")}</h2>
        <table className="table">
          <thead><tr><th>{t("common.time")}</th><th>{t("common.type")}</th><th>{t("common.model")}</th><th>{t("common.latency")}</th><th>{t("common.savings")}</th></tr></thead>
          <tbody>
            {events.map(row => <tr key={row.join("-")}>{row.map((cell, index) => <td key={`${cell}-${index}`} className={index === 3 || index === 4 ? "num" : ""}>{cell}</td>)}</tr>)}
          </tbody>
        </table>
      </section>
    </>
  );
}
