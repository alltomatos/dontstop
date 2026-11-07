"use client";

import { PageHeader } from "@/components/console/page";
import { StatusChip } from "@/components/console/origin-chips";
import { useI18n } from "@/components/console/i18n-provider";
import { CONFIGURED_PROVIDERS } from "@/lib/providers";

const kpis = [
  { labelKey: "dashboard.kpiRps", value: "842", delta: "dashboard.vsYesterday" },
  { labelKey: "dashboard.kpiStreams", value: "3.184", delta: "+8%" },
  { labelKey: "dashboard.kpiP95", value: "28 ms", delta: "-3 ms" },
  { labelKey: "dashboard.kpiErrors", value: "0,4%", delta: "-0,1 pp" },
  { labelKey: "dashboard.kpiCache", value: "41,2%", delta: "+5,1 pp" },
  { labelKey: "dashboard.kpiCostToday", value: "US$ 312", delta: "-18%" }
];

export default function DashboardPage() {
  const { t } = useI18n();
  return (
    <>
      <PageHeader title={t("dashboard.title")} subtitle={t("dashboard.subtitle")} />
      <div className="kpis">
        {kpis.map(item => (
          <div className="card kpi" key={item.labelKey}>
            <div className="label">{t(item.labelKey)}</div>
            <div className="value num">{item.value}</div>
            <div className="delta">{item.delta.startsWith("dashboard.") ? t(item.delta) : item.delta}</div>
          </div>
        ))}
      </div>
      <div className="two">
        <div className="card">
          <h2>{t("observability.traffic")}</h2>
          <p className="muted">{t("dashboard.chartHint")}</p>
        </div>
        <div className="card">
          <h2>{t("cost.byProvider")}</h2>
          <div className="list">
            <div className="list-item"><span>OpenAI</span><span className="num">US$ 148 · 47%</span></div>
            <div className="list-item"><span>Anthropic</span><span className="num">US$ 96 · 31%</span></div>
            <div className="list-item"><span>Gemini</span><span className="num">US$ 42 · 13%</span></div>
            <div className="list-item"><span>Outros</span><span className="num">US$ 26 · 9%</span></div>
          </div>
        </div>
      </div>

      <section className="card" style={{ marginTop: 16 }}>
        <div className="row">
          <h2>{t("dashboard.health")}</h2>
          <span className="chip">{CONFIGURED_PROVIDERS.length}</span>
        </div>
        <table className="table">
          <thead><tr><th>{t("common.provider")}</th><th>{t("common.accounts")}</th><th>{t("common.status")}</th><th>P95</th><th>{t("common.errors")}</th><th>{t("common.health")}</th></tr></thead>
          <tbody>
            {CONFIGURED_PROVIDERS.slice(0, 5).map((provider, index) => (
              <tr key={provider.id}>
                <td>{provider.name}</td>
                <td>{provider.accounts.length}</td>
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
            <div className="list-item"><span><span className="chip warn"><span className="dot" />Anthropic</span> Cooldown ? 12 min</span><button className="btn" type="button">{t("common.viewTrace")}</button></div>
            <div className="list-item"><span><span className="chip ok"><span className="dot" />Gateway</span> p95 28 ms</span><span className="muted">24h</span></div>
          </div>
        </section>
        <section className="card">
          <h2>{t("dashboard.topModels")}</h2>
          <div className="list">
            <div className="list-item"><span>gpt-4.1</span><span className="num">82,4k</span></div>
            <div className="list-item"><span>claude-sonnet-4</span><span className="num">61,2k</span></div>
            <div className="list-item"><span>gemini-2.5-pro</span><span className="num">32,8k</span></div>
          </div>
        </section>
      </div>
    </>
  );
}
