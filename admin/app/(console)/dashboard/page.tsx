"use client";

import { PageHeader } from "@/components/console/page";
import { useI18n } from "@/components/console/i18n-provider";

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
    </>
  );
}
