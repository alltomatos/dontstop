"use client";

import { Breadcrumbs, PageHeader } from "@/components/console/page";
import { BarChart, Progress } from "@/components/console/ui";
import { useI18n } from "@/components/console/i18n-provider";
import type { CostReport } from "@/lib/admin-api";

const fallbackProviders = [
  { provider_name: "OpenAI", cost_usd: 148, requests: 47 },
  { provider_name: "Anthropic", cost_usd: 96, requests: 31 },
  { provider_name: "Gemini", cost_usd: 42, requests: 13 },
  { provider_name: "Outros", cost_usd: 26, requests: 9 }
];

export function CostView({ report }: { report: CostReport | null }) {
  const { t } = useI18n();
  const total = report?.summary.cost_usd ?? 8420;
  const providers = report?.by_provider.length ? report.by_provider : fallbackProviders;
  const days = report?.by_day.length ? report.by_day : [];
  const maxDay = Math.max(...days.map(day => day.cost_usd), 1);
  const bars = days.length ? days.map(day => Math.max(8, (day.cost_usd / maxDay) * 100)) : [30, 38, 34, 48, 42, 58, 52, 64, 60, 72, 68, 76];
  const totalProviders = providers.reduce((sum, provider) => sum + provider.cost_usd, 0) || 1;

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
          ["cost.month", `US$ ${total.toLocaleString()}`, "+18%"],
          ["cost.cacheSavings", "US$ 3.910", "-32%"],
          ["cost.budget", "62%", "US$ 13.500"],
          ["cost.projection", `US$ ${(total * 1.1).toLocaleString()}`, t("cost.aboveTarget")]
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
          <BarChart values={bars} />
        </section>
        <section className="card">
          <h2>{t("cost.byProvider")}</h2>
          <div className="list">
            {providers.slice(0, 4).map(provider => (
              <div className="list-item" key={provider.provider_name}>
                <span>{provider.provider_name}</span>
                <span className="num">US$ {provider.cost_usd.toLocaleString()} · {Math.round((provider.cost_usd / totalProviders) * 100)}%</span>
              </div>
            ))}
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
