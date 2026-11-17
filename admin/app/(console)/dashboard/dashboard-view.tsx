"use client";

import { PageHeader } from "@/components/console/page";
import { StatusChip } from "@/components/console/origin-chips";
import { BarChart } from "@/components/console/ui";
import { useI18n } from "@/components/console/i18n-provider";
import type { CacheStats, CostReport, ObservabilitySummary } from "@/lib/admin-api";
import type { Provider } from "@/lib/providers";

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

export function DashboardView({
  providers,
  cost,
  observability,
  cache,
  gatewayError = false
}: {
  providers: Provider[];
  cost: CostReport | null;
  observability: ObservabilitySummary | null;
  cache: CacheStats | null;
  gatewayError?: boolean;
}) {
  const { t, locale } = useI18n();
  const summary = cost?.summary ?? null;
  const requests = observability?.requests ?? 0;
  const dailyRequests = summary?.days_with_data ? summary.requests / summary.days_with_data : 0;

  const kpis = [
    {
      labelKey: "dashboard.kpiRequests",
      value: observability ? observability.requests.toLocaleString(locale) : empty,
      hint: t("common.last24h")
    },
    {
      labelKey: "dashboard.kpiRequestsPerDay",
      value: summary?.days_with_data ? Math.round(dailyRequests).toLocaleString(locale) : empty,
      hint: t("dashboard.windowHint", { days: summary?.days ?? 0 })
    },
    {
      labelKey: "dashboard.kpiCostWindow",
      value: summary ? formatUSD(summary.cost_usd, locale) : empty,
      hint: t("dashboard.periodHint", {
        start: summary?.period_start ?? empty,
        end: summary?.period_end ?? empty
      })
    },
    {
      labelKey: "dashboard.kpiCostPerRequest",
      value: summary && summary.requests > 0 ? formatUSD(summary.avg_cost_per_request_usd, locale) : empty,
      hint: t("dashboard.billableOnly")
    },
    {
      labelKey: "dashboard.kpiP95",
      value: observability ? `${Math.round(observability.p95_ms)} ms` : empty,
      hint: t("dashboard.p95Hint", { p50: Math.round(observability?.p50_ms ?? 0) })
    },
    {
      labelKey: "dashboard.kpiErrors",
      value: observability ? formatPercent(observability.error_rate, locale) : empty,
      hint: `${observability?.errors ?? 0} / ${requests}`
    },
    {
      labelKey: "dashboard.kpiCache",
      value: cache && cache.cacheable > 0 ? formatPercent(cache.hit_rate, locale) : empty,
      hint: t("dashboard.cacheHint", { hits: cache?.hits ?? 0, cacheable: cache?.cacheable ?? 0 })
    },
    {
      labelKey: "dashboard.kpiTokens",
      value: summary ? formatTokens(summary.tokens_total, locale) : empty,
      hint: `in ${formatTokens(summary?.tokens_in ?? 0, locale)} · out ${formatTokens(summary?.tokens_out ?? 0, locale)}`
    }
  ];

  const byDay = cost?.by_day ?? [];
  const maxDay = Math.max(...byDay.map(day => day.requests), 1);
  const traffic = byDay.map(day => Math.max(8, (day.requests / maxDay) * 100));
  const costProviders = cost?.by_provider ?? [];
  const totalProviderCost = costProviders.reduce((sum, provider) => sum + provider.cost_usd, 0);
  const topModels = cost?.by_model ?? [];

  return (
    <>
      <PageHeader title={t("dashboard.title")} subtitle={t("dashboard.subtitle")} />
      {gatewayError ? (
        <p className="small" role="status" style={{ color: "var(--danger)", marginBottom: 12 }}>
          {t("common.gatewayError")}
        </p>
      ) : null}
      {summary && summary.days_with_data < summary.days ? (
        <p className="muted small" style={{ marginBottom: 12 }}>
          {t("dashboard.partialWindow", { withData: summary.days_with_data, days: summary.days })}
        </p>
      ) : null}
      <div className="kpis">
        {kpis.map(item => (
          <div className="card kpi" key={item.labelKey}>
            <div className="label">{t(item.labelKey)}</div>
            <div className="value num">{item.value}</div>
            <div className="hint">{item.hint}</div>
          </div>
        ))}
      </div>

      <div className="two">
        <div className="card">
          <h2>{t("observability.traffic")}</h2>
          {traffic.length ? (
            <>
              <BarChart values={traffic} />
              <div className="actions" style={{ marginTop: 12 }}>
                <span className="chip">
                  {t("dashboard.peakDay", {
                    day: byDay.reduce((best, day) => (day.requests > best.requests ? day : best), byDay[0]).day,
                    requests: Math.max(...byDay.map(day => day.requests)).toLocaleString(locale)
                  })}
                </span>
              </div>
            </>
          ) : (
            <p className="muted small">{t("common.noData")}</p>
          )}
          <p className="muted small">{t("dashboard.trafficHint")}</p>
        </div>
        <div className="card">
          <h2>{t("cost.byProvider")}</h2>
          <div className="list">
            {costProviders.slice(0, 4).map(provider => (
              <div className="list-item" key={provider.provider_id}>
                <span>{provider.provider_name}</span>
                <span className="num">
                  {formatUSD(provider.cost_usd, locale)} · {provider.requests.toLocaleString(locale)}
                  {totalProviderCost > 0 ? ` · ${formatPercent(provider.cost_usd / totalProviderCost, locale)}` : ""}
                </span>
              </div>
            ))}
            {costProviders.length === 0 ? <p className="muted small">{t("common.noData")}</p> : null}
          </div>
        </div>
      </div>

      <section className="card" style={{ marginTop: 16 }}>
        <div className="row">
          <h2>{t("dashboard.health")}</h2>
          <span className="chip">{providers.length}</span>
        </div>
        <table className="table">
          <thead><tr><th>{t("common.provider")}</th><th>{t("common.accounts")}</th><th>{t("common.status")}</th><th>{t("providerHealth.modelsUsable")}</th><th>{t("dashboard.errorsWindow")}</th><th>{t("dashboard.costWindow")}</th></tr></thead>
          <tbody>
            {providers.slice(0, 5).map(provider => {
              const usage = costProviders.find(item => item.provider_name === provider.name);
              return (
                <tr key={provider.id}>
                  <td>{provider.name}</td>
                  <td>{provider.accountsCount ?? provider.accounts.length}</td>
                  <td><StatusChip status={provider.status} label={t(provider.statusKey)} /></td>
                  <td className="num">{provider.modelsUsable ?? empty}</td>
                  <td className="num">{usage ? usage.errors.toLocaleString(locale) : empty}</td>
                  <td className="num">{usage ? formatUSD(usage.cost_usd, locale) : empty}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
        {providers.length === 0 ? <p className="muted small">{t("common.noData")}</p> : null}
        <p className="muted small">{t("dashboard.healthHint")}</p>
      </section>

      <div className="grid-2">
        <section className="card">
          <h2>{t("dashboard.topModels")}</h2>
          <div className="list">
            {topModels.slice(0, 5).map(model => (
              <div className="list-item" key={model.model}>
                <span>{model.model}</span>
                <span className="num">
                  {model.requests.toLocaleString(locale)} · {formatTokens(model.tokens, locale)} tok · {formatUSD(model.cost_usd, locale)}
                </span>
              </div>
            ))}
            {topModels.length === 0 ? <p className="muted small">{t("common.noData")}</p> : null}
          </div>
        </section>
        <section className="card">
          <h2>{t("dashboard.cache")}</h2>
          <div className="list">
            <div className="list-item"><span>{t("cache.hitTotal")}</span><span className="num">{cache ? formatPercent(cache.hit_rate, locale) : empty}</span></div>
            <div className="list-item"><span>{t("dashboard.cacheHits")}</span><span className="num">{cache ? cache.hits.toLocaleString(locale) : empty}</span></div>
            <div className="list-item"><span>{t("dashboard.cacheMisses")}</span><span className="num">{cache ? cache.misses.toLocaleString(locale) : empty}</span></div>
            <div className="list-item"><span>{t("dashboard.cacheBypass")}</span><span className="num">{cache ? cache.bypass.toLocaleString(locale) : empty}</span></div>
            <div className="list-item"><span>{t("dashboard.cacheTokensReused")}</span><span className="num">{cache ? formatTokens(cache.tokens_reused, locale) : empty}</span></div>
          </div>
          <p className="muted small">{t("dashboard.cacheBypassHint")}</p>
        </section>
      </div>
    </>
  );
}
