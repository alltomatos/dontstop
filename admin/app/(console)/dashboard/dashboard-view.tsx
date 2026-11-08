"use client";

import { PageHeader } from "@/components/console/page";
import { StatusChip } from "@/components/console/origin-chips";
import { BarChart } from "@/components/console/ui";
import { useI18n } from "@/components/console/i18n-provider";
import type { CacheStats, CostReport, ObservabilitySummary } from "@/lib/admin-api";
import type { Provider } from "@/lib/providers";

export function DashboardView({
  providers,
  cost,
  observability,
  cache
}: {
  providers: Provider[];
  cost: CostReport | null;
  observability: ObservabilitySummary | null;
  cache: CacheStats | null;
}) {
  const { t } = useI18n();
  const requests = observability?.requests ?? 72800;
  const rps = Math.max(1, Math.round(requests / 86400));
  const p95 = observability ? `${Math.round(observability.p95_ms)} ms` : "28 ms";
  const errorRate = observability ? `${(observability.error_rate * 100).toFixed(1)}%` : "0,4%";
  const hitRate = cache ? `${(cache.hit_rate * 100).toFixed(1)}%` : "41,2%";
  const totalCost = cost?.summary.cost_usd ?? 312;
  const byDay = cost?.by_day ?? [];
  const maxDay = Math.max(...byDay.map(day => day.requests), 1);
  const traffic = byDay.length ? byDay.map(day => Math.max(8, (day.requests / maxDay) * 100)) : [38, 52, 44, 66, 58, 74, 62, 81, 70, 88, 76, 92];
  const costProviders = cost?.by_provider.length ? cost.by_provider : [];
  const totalProviderCost = costProviders.reduce((sum, provider) => sum + provider.cost_usd, 0) || 1;
  const topModels = cost?.by_model.length ? cost.by_model : [
    { model: "gpt-4.1", requests: 82400, cost_usd: 148, tokens: 0 },
    { model: "claude-sonnet-4", requests: 61200, cost_usd: 96, tokens: 0 },
    { model: "gemini-2.5-pro", requests: 32800, cost_usd: 42, tokens: 0 }
  ];

  const kpis = [
    { labelKey: "dashboard.kpiRps", value: String(rps), delta: t("dashboard.vsYesterday") },
    { labelKey: "dashboard.kpiStreams", value: "3.184", delta: "+8%" },
    { labelKey: "dashboard.kpiP95", value: p95, delta: "-3 ms" },
    { labelKey: "dashboard.kpiErrors", value: errorRate, delta: "-0,1 pp" },
    { labelKey: "dashboard.kpiCache", value: hitRate, delta: "+5,1 pp" },
    { labelKey: "dashboard.kpiCostToday", value: `US$ ${totalCost.toLocaleString()}`, delta: "-18%" }
  ];

  return (
    <>
      <PageHeader title={t("dashboard.title")} subtitle={t("dashboard.subtitle")} />
      <div className="kpis">
        {kpis.map(item => (
          <div className="card kpi" key={item.labelKey}>
            <div className="label">{t(item.labelKey)}</div>
            <div className="value num">{item.value}</div>
            <div className="delta">{item.delta}</div>
          </div>
        ))}
      </div>

      <div className="two">
        <div className="card">
          <h2>{t("observability.traffic")}</h2>
          <BarChart values={traffic} />
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
            {costProviders.length === 0 ? (
              <>
                <div className="list-item"><span>OpenAI</span><span className="num">US$ 148 · 47%</span></div>
                <div className="list-item"><span>Anthropic</span><span className="num">US$ 96 · 31%</span></div>
                <div className="list-item"><span>Gemini</span><span className="num">US$ 42 · 13%</span></div>
                <div className="list-item"><span>{t("common.others")}</span><span className="num">US$ 26 · 9%</span></div>
              </>
            ) : null}
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
            {providers.slice(0, 5).map((provider, index) => (
              <tr key={provider.id}>
                <td>{provider.name}</td>
                <td>{provider.accountsCount ?? provider.accounts.length}</td>
                <td><StatusChip status={provider.status} label={t(provider.statusKey)} /></td>
                <td className="num">{180 + index * 20} ms</td>
                <td className="num">{(0.2 + index * 0.3).toFixed(1)}%</td>
                <td className="num">{38 + index * 3}%</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <div className="grid-2">
        <section className="card">
          <h2>{t("dashboard.incidents")}</h2>
          <div className="list">
            <div className="list-item"><span><span className="chip warn"><span className="dot" />Anthropic</span> {t("common.cooldown")} · 12 min</span><button className="btn" type="button">{t("common.viewTrace")}</button></div>
            <div className="list-item"><span><span className="chip ok"><span className="dot" />Gateway</span> p95 {p95}</span><span className="muted">24h</span></div>
          </div>
        </section>
        <section className="card">
          <h2>{t("dashboard.topModels")}</h2>
          <div className="list">
            {topModels.slice(0, 3).map(model => (
              <div className="list-item" key={model.model}><span>{model.model}</span><span className="num">{model.requests.toLocaleString()}</span></div>
            ))}
          </div>
        </section>
      </div>
    </>
  );
}
