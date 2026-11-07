"use client";

import { Breadcrumbs, PageHeader } from "@/components/console/page";
import { BarChart, Progress } from "@/components/console/ui";
import { useI18n } from "@/components/console/i18n-provider";
import type { ObservabilitySummary } from "@/lib/admin-api";

export function ObservabilityView({ summary }: { summary: ObservabilitySummary | null }) {
  const { t } = useI18n();
  const availability = summary && summary.requests > 0 ? `${(100 - summary.error_rate * 100).toFixed(2)}%` : "99,94%";
  const p95 = summary ? `${Math.round(summary.p95_ms)} ms` : "28 ms";
  const errorRate = summary ? `${(summary.error_rate * 100).toFixed(1)}%` : "0,4%";
  return (
    <>
      <Breadcrumbs items={[{ label: t("nav.observability") }]} />
      <PageHeader
        title={t("nav.observability")}
        subtitle={t("observability.subtitle")}
        actions={<button className="btn" type="button">{t("common.openBackend")}</button>}
      />
      <div className="kpis">
        {[
          ["observability.availability", availability, t("observability.metaAvailability")],
          ["observability.latency", p95, t("observability.metaP95")],
          ["observability.errorRate", errorRate, t("observability.metaError")],
          ["observability.saturation", "62%", t("observability.replicas", { count: 8 })]
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
          <h2>{t("observability.latency")}</h2>
          <BarChart values={[24, 38, 32, 48, 42, 58, 52, 64, 60, 72, 68, 76]} />
          <div className="actions" style={{ marginTop: 12 }}>
            <span className="chip">p50 18 ms</span>
            <span className="chip">p95 28 ms</span>
            <span className="chip">p99 64 ms</span>
          </div>
        </section>
        <section className="card">
          <h2>{t("observability.traffic")}</h2>
          <BarChart values={[30, 36, 34, 42, 39, 48, 45, 55, 50, 60, 56, 64]} alt />
          <p className="muted small">{t("observability.dataHint")}</p>
        </section>
      </div>

      <section className="card" style={{ marginTop: 16 }}>
        <h2>{t("observability.slo")}</h2>
        <table className="table">
          <thead><tr><th>{t("common.name")}</th><th>{t("common.status")}</th><th>{t("common.limit")}</th><th>{t("common.used")}</th></tr></thead>
          <tbody>
            {[
              [t("observability.sloAvailability"), "99,94%", "99,9%", 43],
              [t("observability.latency"), p95, "30 ms", summary ? Math.min(100, Math.round(summary.p95_ms / 30 * 100)) : 80],
              [t("observability.errorRate"), "0,4%", "1%", 40]
            ].map(row => (
              <tr key={row[0]}>
                <td>{row[0]}</td>
                <td>{row[1]}</td>
                <td>{row[2]}</td>
                <td><Progress value={Number(row[3])} tone={Number(row[3]) > 75 ? "warn" : "ok"} /></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="card" style={{ marginTop: 16 }}>
        <h2>{t("observability.alerts")}</h2>
        <div className="list-item">
          <span><span className="chip warn"><span className="dot" />{t("common.critical")}</span> {t("observability.incidentAnthropic429")} · 12 min</span>
          <div className="actions">
            <button className="btn" type="button">{t("common.viewTrace")}</button>
            <button className="btn" type="button">{t("common.silence")}</button>
          </div>
        </div>
      </section>
    </>
  );
}
