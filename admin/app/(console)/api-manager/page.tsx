"use client";

import { useState } from "react";
import { Plus, X } from "lucide-react";
import { Breadcrumbs, PageHeader } from "@/components/console/page";
import { CopyButton, Tabs } from "@/components/console/ui";
import { useI18n } from "@/components/console/i18n-provider";

interface KeyRow {
  name: string;
  prefix: string;
  scopes: string;
  quota: string;
  lastUse: string;
  status: "active" | "warning" | "expiring";
}

const initialKeys: KeyRow[] = [
  { name: "prod-web", prefix: "sk-9f2…", scopes: "chat", quota: "50M", lastUse: "2 min", status: "active" },
  { name: "etl-batch", prefix: "sk-1a4…", scopes: "embed", quota: "10M", lastUse: "1 h", status: "warning" },
  { name: "sandbox", prefix: "sk-7c1…", scopes: "chat", quota: "1M", lastUse: "3 d", status: "active" },
  { name: "legacy-report", prefix: "sk-0b8…", scopes: "models", quota: "5M", lastUse: "12 d", status: "expiring" }
];

export default function ApiManagerPage() {
  const { t } = useI18n();
  const [tab, setTab] = useState("keys");
  const [keys, setKeys] = useState(initialKeys);
  const [open, setOpen] = useState(false);
  const [step, setStep] = useState(1);
  const [name, setName] = useState("prod-web");
  const [secret, setSecret] = useState("");

  function createKey() {
    const generated = `sk-ds-${Math.random().toString(36).slice(2, 10)}${Math.random().toString(36).slice(2, 10)}`;
    setSecret(generated);
    setStep(2);
  }

  function finish() {
    setKeys(current => [
      { name: name || "new-key", prefix: `${secret.slice(0, 6)}…`, scopes: "chat", quota: "50M", lastUse: "—", status: "active" },
      ...current
    ]);
    setOpen(false);
    setStep(1);
    setName("");
    setSecret("");
  }

  return (
    <>
      <Breadcrumbs items={[{ label: t("nav.apiManager") }]} />
      <PageHeader
        title={t("nav.apiManager")}
        subtitle={t("api.subtitle")}
        actions={(
          <>
            <button className="btn" type="button">{t("common.export")}</button>
            <button className="btn primary" type="button" onClick={() => { setOpen(true); setStep(1); }}>
              <Plus aria-hidden="true" size={16} />
              {t("common.newKey")}
            </button>
          </>
        )}
      />

      <div className="kpis">
        {[
          ["dashboard.kpiActiveKeys", String(keys.length), "+3"],
          ["dashboard.kpiRequests", "184,2k", "+12%"],
          ["dashboard.kpiCost", "US$ 312", "-18%"],
          ["dashboard.kpiExpiring", "3", "30 d"]
        ].map(([key, value, delta]) => (
          <div className="card kpi" key={key}>
            <div className="label">{t(key)}</div>
            <div className="value num">{value}</div>
            <div className="delta">{delta}</div>
          </div>
        ))}
      </div>

      <Tabs
        value={tab}
        onChange={setTab}
        items={[
          { id: "keys", label: t("api.keys") },
          { id: "consumers", label: t("api.consumers") },
          { id: "scopes", label: t("api.scopes") },
          { id: "usage", label: t("api.usage") },
          { id: "audit", label: t("api.audit") }
        ]}
      />

      {tab === "keys" ? (
        <section className="card">
          <table className="table">
            <thead><tr><th>{t("common.name")}</th><th>{t("api.prefix")}</th><th>{t("api.scopes")}</th><th>{t("common.quota")}</th><th>{t("api.lastUse")}</th><th>{t("common.status")}</th></tr></thead>
            <tbody>
              {keys.map(row => (
                <tr key={`${row.name}-${row.prefix}`}>
                  <td>{row.name}</td>
                  <td className="mono">{row.prefix}</td>
                  <td className="mono">{row.scopes}</td>
                  <td className="num">{row.quota}</td>
                  <td>{row.lastUse}</td>
                  <td><span className={`chip ${row.status === "active" ? "ok" : row.status === "warning" ? "warn" : "warn"}`}>{row.status === "active" ? t("common.active") : row.status === "warning" ? "96%" : t("dashboard.kpiExpiring")}</span></td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      ) : null}

      {tab === "consumers" ? <SimpleTable headers={[t("common.name"), t("api.keys")]} rows={[["prod-web", "1"], ["etl", "2"]]} /> : null}
      {tab === "scopes" ? <SimpleTable headers={[t("api.scopes"), t("common.status")]} rows={[["chat:write", t("common.active")], ["models:read", t("common.active")], ["embeddings:write", t("common.active")]]} /> : null}
      {tab === "usage" ? <SimpleTable headers={[t("common.model"), t("common.used")]} rows={[["gpt-4.1", "82,4k"], ["claude-sonnet-4", "61,2k"]]} /> : null}
      {tab === "audit" ? <SimpleTable headers={[t("common.time"), t("common.action")]} rows={[["10:45", "api_key.create"], ["10:40", "api_key.rotate"]]} /> : null}

      {open ? (
        <div className="modal-backdrop" role="presentation" onClick={event => { if (event.target === event.currentTarget) setOpen(false); }}>
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="new-key-title">
            <div className="modal-head">
              <div>
                <h2 id="new-key-title">{step === 1 ? t("common.newKey") : t("common.done")}</h2>
                <p className="muted">{step === 1 ? t("api.subtitle") : t("common.copyKey")}</p>
              </div>
              <button className="modal-close" type="button" aria-label={t("common.close")} onClick={() => setOpen(false)}><X aria-hidden="true" size={16} /></button>
            </div>
            {step === 1 ? (
              <>
                <div className="form-grid">
                  <label className="field"><span>{t("common.name")}</span><input className="input" value={name} onChange={event => setName(event.target.value)} /></label>
                  <label className="field"><span>{t("common.quota")}</span><input className="input" defaultValue="50M" /></label>
                </div>
                <div className="modal-foot">
                  <button className="btn" type="button" onClick={() => setOpen(false)}>{t("common.cancel")}</button>
                  <button className="btn primary" type="button" onClick={createKey}>{t("common.createKey")}</button>
                </div>
              </>
            ) : (
              <>
                <div className="codebox">{secret}</div>
                <div className="actions" style={{ marginTop: 12 }}>
                  <CopyButton value={secret} label={t("common.copyKey")} />
                  <button className="btn primary" type="button" onClick={finish}>{t("common.done")}</button>
                </div>
              </>
            )}
          </div>
        </div>
      ) : null}
    </>
  );
}

function SimpleTable({ headers, rows }: { headers: string[]; rows: string[][] }) {
  return (
    <section className="card">
      <table className="table">
        <thead><tr>{headers.map(header => <th key={header}>{header}</th>)}</tr></thead>
        <tbody>{rows.map(row => <tr key={row.join("-")}>{row.map((cell, index) => <td key={`${cell}-${index}`}>{cell}</td>)}</tr>)}</tbody>
      </table>
    </section>
  );
}
