"use client";

import { Breadcrumbs, PageHeader } from "@/components/console/page";
import { BarChart, Progress } from "@/components/console/ui";
import { useI18n } from "@/components/console/i18n-provider";
import type { ObservabilitySummary } from "@/lib/admin-api";

const empty = "—";

function formatPercent(value: number, locale: string): string {
  return new Intl.NumberFormat(locale, { style: "percent", maximumFractionDigits: 2 }).format(value);
}

export function ObservabilityView({ summary }: { summary: ObservabilitySummary | null }) {
  const { t, locale } = useI18n();
  const hasData = Boolean(summary && summary.requests > 0);
  const availability = summary && summary.requests > 0 ? 1 - summary.error_rate : null;
  const latencySeries = summary && summary.requests > 0
    ? [summary.p50_ms, summary.p95_ms, summary.p99_ms].map(value => Math.min(100, Math.round(value)))
    : [];

  const kpis = [
    {
      key: "observability.availability",
      value: availability === null ? empty : formatPercent(availability, locale),
      hint: summary ? `${summary.requests.toLocaleString(locale)} ${t("common.total").toLowerCase()}` : empty
    },
    {
      key: "observability.latency",
      value: summary ? `${Math.round(summary.p95_ms)} ms` : empty,
      hint: t("observability.metaP95")
    },
    {
      key: "observability.errorRate",
      value: summary ? formatPercent(summary.error_rate, locale) : empty,
      hint: summary ? `${summary.errors.toLocaleString(locale)} / ${summary.requests.toLocaleString(locale)}` : empty
    },
    {
      key: "observability.tokens",
      value: summary
        ? new Intl.NumberFormat(locale, { notation: "compact", maximumFractionDigits: 1 }).format(summary.tokens_in + summary.tokens_out)
        : empty,
      hint: `in ${summary?.tokens_in.toLocaleString(locale) ?? empty} · out ${summary?.tokens_out.toLocaleString(locale) ?? empty}`
    }
  ];

  return (
    <>
      <Breadcrumbs items={[{ label: t("nav.observability") }]} />
      <PageHeader title={t("nav.observability")} subtitle={t("observability.subtitle")} />
      {!hasData ? (
        <p className="muted small" role="status" style={{ marginBottom: 12 }}>{t("observability.noData")}</p>
      ) : null}
      <div className="kpis">
        {kpis.map(item => (
          <div className="card kpi" key={item.key}>
            <div className="label">{t(item.key)}</div>
            <div className="value num">{item.value}</div>
            <div className="hint">{item.hint}</div>
          </div>
        ))}
      </div>

      <div className="grid-2">
        <section className="card">
          <h2>{t("observability.latency")}</h2>
          {latencySeries.length ? (
            <>
              <BarChart values={latencySeries} />
              <div className="actions" style={{ marginTop: 12 }}>
                <span className="chip">p50 {Math.round(summary?.p50_ms ?? 0)} ms</span>
                <span className="chip">p95 {Math.round(summary?.p95_ms ?? 0)} ms</span>
                <span className="chip">p99 {Math.round(summary?.p99_ms ?? 0)} ms</span>
              </div>
            </>
          ) : (
            <p className="muted small">{t("common.noData")}</p>
          )}
          <p className="muted small">{t("observability.latencyHint")}</p>
        </section>
        <section className="card">
          <h2>{t("observability.traffic")}</h2>
          {hasData ? (
            <BarChart values={[Math.min(100, Math.round((availability ?? 0) * 100))]} alt />
          ) : (
            <p className="muted small">{t("common.noData")}</p>
          )}
          <p className="muted small">{t("observability.dataHint")}</p>
        </section>
      </div>

      <section className="card" style={{ marginTop: 16 }}>
        <h2>{t("observability.slo")}</h2>
        <table className="table">
          <thead><tr><th>{t("common.name")}</th><th>{t("common.used")}</th><th>{t("common.limit")}</th><th>{t("common.status")}</th></tr></thead>
          <tbody>
            {[
              {
                name: t("observability.sloAvailability"),
                used: availability === null ? empty : formatPercent(availability, locale),
                target: "99,9%",
                progress: availability === null ? 0 : Math.min(100, Math.round(availability * 100)),
                inverted: true
              },
              {
                name: t("observability.sloErrorBudget"),
                used: summary ? formatPercent(summary.error_rate, locale) : empty,
                target: "1%",
                progress: summary ? Math.min(100, Math.round((summary.error_rate / 0.01) * 100)) : 0,
                inverted: false
              }
            ].map(row => {
              const tone = row.inverted
                ? row.progress < 99 ? "warn" : "ok"
                : row.progress > 90 ? "bad" : row.progress > 75 ? "warn" : "ok";
              return (
                <tr key={row.name}>
                  <td>{row.name}</td>
                  <td className="num">{row.used}</td>
                  <td className="num">{row.target}</td>
                  <td><Progress value={row.progress} tone={tone} /></td>
                </tr>
              );
            })}
          </tbody>
        </table>
        <p className="muted small">{t("observability.sloHint")}</p>
      </section>
    </>
  );
}
