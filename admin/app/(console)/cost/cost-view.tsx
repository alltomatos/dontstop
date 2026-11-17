"use client";

import { Breadcrumbs, PageHeader } from "@/components/console/page";
import { BarChart } from "@/components/console/ui";
import { useI18n } from "@/components/console/i18n-provider";
import type { CostReport } from "@/lib/admin-api";

const empty = "—";

function formatUSD(value: number, locale: string): string {
  return new Intl.NumberFormat(locale, {
    style: "currency",
    currency: "USD",
    maximumFractionDigits: value < 1 ? 4 : 2
  }).format(value);
}

function formatTokens(value: number, locale: string): string {
  return new Intl.NumberFormat(locale, { notation: "compact", maximumFractionDigits: 1 }).format(value);
}

function formatPercent(value: number, locale: string): string {
  return new Intl.NumberFormat(locale, { style: "percent", maximumFractionDigits: 1 }).format(value);
}

export function CostView({ report }: { report: CostReport | null }) {
  const { t, locale } = useI18n();
  const summary = report?.summary ?? null;
  const providers = report?.by_provider ?? [];
  const models = report?.by_model ?? [];
  const days = report?.by_day ?? [];
  const maxDay = Math.max(...days.map(day => day.cost_usd), 0);
  const bars = days.length
    ? days.map(day => Math.max(8, maxDay > 0 ? (day.cost_usd / maxDay) * 100 : 0))
    : [];
  const totalProviders = providers.reduce((sum, provider) => sum + provider.cost_usd, 0);

  const kpis = [
    {
      key: "cost.window",
      value: summary ? formatUSD(summary.cost_usd, locale) : empty,
      delta: summary
        ? t("dashboard.periodHint", {
            start: summary.period_start ?? empty,
            end: summary.period_end ?? empty
          })
        : t("common.noData")
    },
    {
      key: "cost.billableRequests",
      value: summary ? summary.requests.toLocaleString(locale) : empty,
      delta: summary ? t("dashboard.cacheHitBonus", { hits: summary.cache_hits.toLocaleString(locale) }) : empty
    },
    {
      key: "cost.avgPerRequest",
      value: summary && summary.requests > 0 ? formatUSD(summary.avg_cost_per_request_usd, locale) : empty,
      delta: summary ? `${formatTokens(summary.tokens_total, locale)} tokens` : empty
    },
    {
      key: "cost.errorsWindow",
      value: summary ? formatPercent(summary.error_rate, locale) : empty,
      delta: summary ? `${summary.errors.toLocaleString(locale)} / ${summary.requests.toLocaleString(locale)}` : empty
    }
  ];

  return (
    <>
      <Breadcrumbs items={[{ label: t("cost.title") }]} />
      <PageHeader title={t("cost.title")} subtitle={t("cost.subtitle")} />
      {!summary ? (
        <p className="muted small" role="status" style={{ marginBottom: 12 }}>{t("cost.noData")}</p>
      ) : null}
      <div className="kpis">
        {kpis.map(item => (
          <div className="card kpi" key={item.key}>
            <div className="label">{t(item.key)}</div>
            <div className="value num">{item.value}</div>
            <div className="hint">{item.delta}</div>
          </div>
        ))}
      </div>

      <div className="grid-2">
        <section className="card">
          <h2>{t("cost.byDay")}</h2>
          {bars.length ? <BarChart values={bars} /> : <p className="muted small">{t("common.noData")}</p>}
          {days.length ? (
            <p className="muted small">
              {t("cost.seriesHint", { days: days.length, window: summary?.days ?? days.length })}
            </p>
          ) : null}
        </section>
        <section className="card">
          <h2>{t("cost.byProvider")}</h2>
          <div className="list">
            {providers.slice(0, 6).map(provider => (
              <div className="list-item" key={provider.provider_id}>
                <span>{provider.provider_name}</span>
                <span className="num">
                  {formatUSD(provider.cost_usd, locale)} · {provider.requests.toLocaleString(locale)}
                  {totalProviders > 0 ? ` · ${formatPercent(provider.cost_usd / totalProviders, locale)}` : ""}
                </span>
              </div>
            ))}
            {providers.length === 0 ? <p className="muted small">{t("common.noData")}</p> : null}
          </div>
        </section>
      </div>

      <section className="card" style={{ marginTop: 16 }}>
        <h2>{t("cost.byModel")}</h2>
        <table className="table">
          <thead><tr><th>{t("common.model")}</th><th>{t("cost.costValue")}</th><th>{t("cost.tokensValue")}</th><th>{t("common.total")}</th></tr></thead>
          <tbody>
            {models.map(model => (
              <tr key={model.model}>
                <td>{model.model}</td>
                <td className="num">{formatUSD(model.cost_usd, locale)}</td>
                <td className="num">{formatTokens(model.tokens, locale)}</td>
                <td className="num">{model.requests.toLocaleString(locale)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        {models.length === 0 ? <p className="muted small">{t("common.noData")}</p> : null}
        <p className="muted small">{t("cost.gapsHint")}</p>
      </section>
    </>
  );
}
