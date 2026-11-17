"use client";

import { Breadcrumbs, PageHeader } from "@/components/console/page";
import { BarChart } from "@/components/console/ui";
import { useI18n } from "@/components/console/i18n-provider";
import type { CacheReport } from "@/lib/admin-api";

const empty = "—";

function formatPercent(value: number, locale: string): string {
  return new Intl.NumberFormat(locale, { style: "percent", maximumFractionDigits: 1 }).format(value);
}

function formatNumber(value: number, locale: string): string {
  return new Intl.NumberFormat(locale).format(value);
}

export function CacheView({ report }: { report: CacheReport | null }) {
  const { t, locale } = useI18n();
  const stats = report?.stats ?? null;
  const series = report?.by_day ?? [];
  const events = report?.events ?? [];
  const hasTraffic = Boolean(stats && stats.requests > 0);

  const hitsBars = series.map(day => (day.hits + day.misses > 0 ? Math.max(2, Math.round(day.hit_rate * 100)) : 2));
  const missBars = series.map(day => (day.hits + day.misses > 0 ? Math.max(2, Math.round((1 - day.hit_rate) * 100)) : 2));

  const kpis = [
    {
      key: "cache.hitTotal",
      value: stats && stats.cacheable > 0 ? formatPercent(stats.hit_rate, locale) : empty,
      delta: stats ? t("dashboard.cacheHint", { hits: stats.hits, cacheable: stats.cacheable }) : empty
    },
    {
      key: "cache.cacheableTraffic",
      value: stats ? formatNumber(stats.cacheable, locale) : empty,
      delta: stats ? t("cache.bypassExcluded", { bypass: formatNumber(stats.bypass, locale) }) : empty
    },
    {
      key: "cache.latencySaved",
      value: stats && stats.latency_saved_ms > 0 ? `${Math.round(stats.latency_saved_ms)} ms` : empty,
      delta: stats
        ? t("cache.latencyCompare", {
            hit: Math.round(stats.hit_latency_ms),
            miss: Math.round(stats.miss_latency_ms)
          })
        : empty
    },
    {
      key: "cache.tokensReused",
      value: stats ? formatNumber(stats.tokens_reused, locale) : empty,
      delta: t("cache.tokensReusedHint")
    }
  ];

  return (
    <>
      <Breadcrumbs items={[{ label: t("nav.cache") }]} />
      <PageHeader title={t("nav.cache")} subtitle={t("cache.subtitle")} />
      {!hasTraffic ? (
        <p className="muted small" role="status" style={{ marginBottom: 12 }}>{t("cache.noTraffic")}</p>
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
          <h2>{t("cache.hitRateByDay")}</h2>
          {series.length ? (
            <>
              <BarChart values={hitsBars} />
              <BarChart values={missBars} alt />
              <div className="actions" style={{ marginTop: 12 }}>
                <span className="chip">{t("cache.legendHits")}</span>
                <span className="chip">{t("cache.legendMisses")}</span>
              </div>
            </>
          ) : (
            <p className="muted small">{t("common.noData")}</p>
          )}
        </section>
        <section className="card">
          <h2>{t("cache.layers")}</h2>
          <div className="list">
            <div className="list-item"><span>{t("cache.l1")}</span><span className="mono">10 s</span></div>
            <div className="list-item"><span>{t("cache.l2")}</span><span className="mono">1 h</span></div>
            <div className="list-item"><span>{t("cache.semanticLayer")}</span><span className="mono">{t("cache.notEnabled")}</span></div>
          </div>
          <p className="muted small">{t("cache.layersHint")}</p>
        </section>
      </div>

      <section className="card" style={{ marginTop: 16 }}>
        <h2>{t("cache.recentEvents")}</h2>
        <table className="table">
          <thead><tr><th>{t("common.time")}</th><th>{t("common.type")}</th><th>{t("common.model")}</th><th>{t("common.latency")}</th><th>{t("common.total")} tokens</th></tr></thead>
          <tbody>
            {events.map((event, index) => (
              <tr key={`${event.time}-${event.model}-${index}`}>
                <td className="mono">{event.time}</td>
                <td><span className={`chip ${event.status === "hit" ? "ok" : event.status === "miss" ? "warn" : ""}`}>{event.status}</span></td>
                <td>{event.model}</td>
                <td className="num">{event.latency_ms} ms</td>
                <td className="num">{formatNumber(event.tokens, locale)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        {events.length === 0 ? <p className="muted small">{t("common.noData")}</p> : null}
        <p className="muted small">{t("cache.eventsHint")}</p>
      </section>
    </>
  );
}
