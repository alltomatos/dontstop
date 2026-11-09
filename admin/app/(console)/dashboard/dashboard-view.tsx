"use client";

import { PageHeader } from "@/components/console/page";
import { StatusChip } from "@/components/console/origin-chips";
import { BarChart } from "@/components/console/ui";
import { useI18n } from "@/components/console/i18n-provider";
import type { CacheStats, CostReport, ObservabilitySummary } from "@/lib/admin-api";
import type { Provider } from "@/lib/providers";

const empty = "—";

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
  const { t } = useI18n();
  const requests = observability?.requests ?? 0;
  const rps = Math.round(requests / 86400);
  const p95 = observability ? `${Math.round(observability.p95_ms)} ms` : empty;
  const errorRate = observability ? `${(observability.error_rate * 100).toFixed(1)}%` : empty;
  const hitRate = cache ? `${(cache.hit_rate * 100).toFixed(1)}%` : empty;
  const totalCost = cost?.summary.cost_usd ?? 0;
  const byDay = cost?.by_day ?? [];
  const maxDay = Math.max(...byDay.map(day => day.requests), 1);
  const traffic = byDay.map(day => Math.max(8, (day.requests / maxDay) * 100));
  const costProviders = cost?.by_provider ?? [];
  const totalProviderCost = costProviders.reduce((sum, provider) => sum + provider.cost_usd, 0) || 1;
  const topModels = cost?.by_model ?? [];

  const kpis = [
    { labelKey: "dashboard.kpiRps", value: observability ? String(rps) : empty },
    { labelKey: "dashboard.kpiRequests", value: observability ? requests.toLocaleString() : empty },
    { labelKey: "dashboard.kpiP95", value: p95 },
    { labelKey: "dashboard.kpiErrors", value: errorRate },
    { labelKey: "dashboard.kpiCache", value: hitRate },
    { labelKey: "dashboard.kpiCostToday", value: cost ? `US$ ${totalCost.toLocaleString()}` : empty }
  ];

  return (
    <>
      <PageHeader title={t("dashboard.title")} subtitle={t("dashboard.subtitle")} />
      {gatewayError ? (
        <p className="small" role="status" style={{ color: "var(--danger)", marginBottom: 12 }}>
          {t("common.gatewayError")}
        </p>
      ) : null}
      <div className="kpis">
        {kpis.map(item => (
          <div className="card kpi" key={item.labelKey}>
            <div className="label">{t(item.labelKey)}</div>
            <div className="value num">{item.value}</div>
          </div>
        ))}
      </div>

      <div className="two">
        <div className="card">
          <h2>{t("observability.traffic")}</h2>
          {traffic.length ? (
            <BarChart values={traffic} />
          ) : (
            <p className="muted small">{t("common.noData")}</p>
          )}
          <p className="muted small">{t("dashboard.chartHint")}</p>
        </div>
        <div className="card">
          <h2>{t("cost.byProvider")}</h2>
          <div className="list">
            {costProviders.slice(0, 4).map(provider => (
              <div className="list-item" key={provider.provider_name}>
                <span>{provider.provider_name}</span>
                <span className="num">US$ {provider.cost_usd.toLocaleString()} · {Math.round((provider.cost_usd / totalProviderCost) * 100)}%</span>
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
          <thead><tr><th>{t("common.provider")}</th><th>{t("common.accounts")}</th><th>{t("common.status")}</th><th>P95</th><th>{t("common.errors")}</th><th>{t("common.health")}</th></tr></thead>
          <tbody>
            {providers.slice(0, 5).map(provider => (
              <tr key={provider.id}>
                <td>{provider.name}</td>
                <td>{provider.accountsCount ?? provider.accounts.length}</td>
                <td><StatusChip status={provider.status} label={t(provider.statusKey)} /></td>
                <td className="num">{empty}</td>
                <td className="num">{empty}</td>
                <td className="num">{empty}</td>
              </tr>
            ))}
          </tbody>
        </table>
        {providers.length === 0 ? <p className="muted small">{t("common.noData")}</p> : null}
      </section>

      <div className="grid-2">
        <section className="card">
          <h2>{t("dashboard.incidents")}</h2>
          <p className="muted small">{t("common.noData")}</p>
        </section>
        <section className="card">
          <h2>{t("dashboard.topModels")}</h2>
          <div className="list">
            {topModels.slice(0, 3).map(model => (
              <div className="list-item" key={model.model}><span>{model.model}</span><span className="num">{model.requests.toLocaleString()}</span></div>
            ))}
            {topModels.length === 0 ? <p className="muted small">{t("common.noData")}</p> : null}
          </div>
        </section>
      </div>
    </>
  );
}
