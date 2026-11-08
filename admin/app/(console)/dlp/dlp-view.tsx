"use client";

import { useState } from "react";
import { Breadcrumbs, PageHeader } from "@/components/console/page";
import { useI18n } from "@/components/console/i18n-provider";
import type { DLPEvent, DLPRule } from "@/lib/admin-api";

const fallbackRules = [
  { name: "CPF/CNPJ", type: "PII", action: "common.redact", severity: "common.high", status: "common.active" },
  { name: "AWS keys", type: "Secret", action: "common.block", severity: "common.critical", status: "common.active" },
  { name: "Prompt injection", type: "Injection", action: "common.auditAction", severity: "common.medium", status: "common.auditAction" }
];

const fallbackEvents = [
  ["10:45", "block", "AWS keys", "tenant A", "openai", "8f2a"],
  ["10:44", "redact", "CPF/CNPJ", "tenant A", "claude", "7c1b"]
];

export function DlpView({ rules, events }: { rules: DLPRule[]; events: DLPEvent[] }) {
  const { t } = useI18n();
  const [text, setText] = useState("Meu CPF e 123.456.789-00 e meu e-mail e teste@exemplo.com");
  const [tested, setTested] = useState(false);
  const actionKeys: Record<string, string> = { allow: "common.active", redact: "common.redact", block: "common.block", audit: "common.auditAction" };
  const severityKeys: Record<string, string> = { low: "common.medium", medium: "common.medium", high: "common.high", critical: "common.critical" };
  const ruleRows = rules.length ? rules.map(rule => ({ name: rule.name, type: rule.detector, action: actionKeys[rule.action] ?? rule.action, severity: severityKeys[rule.severity] ?? rule.severity, status: rule.enabled ? "common.active" : "common.disabled" })) : fallbackRules;
  const eventRows = events.length ? events.map(event => [new Date(event.created_at).toLocaleTimeString(), event.action, event.detector, event.severity]) : fallbackEvents;

  return (
    <>
      <Breadcrumbs items={[{ label: t("dlp.title") }]} />
      <PageHeader
        title={t("dlp.title")}
        subtitle={t("dlp.subtitle")}
        actions={(
          <>
            <button className="btn" type="button">{t("common.newRule")}</button>
            <button className="btn primary" type="button">{t("common.test")}</button>
          </>
        )}
      />
      <section className="card">
        <table className="table">
          <thead><tr><th>{t("dlp.rule")}</th><th>{t("common.type")}</th><th>{t("common.action")}</th><th>{t("common.severity")}</th><th>{t("common.status")}</th></tr></thead>
          <tbody>
            {ruleRows.map(rule => (
              <tr key={rule.name}>
                <td>{rule.name === "AWS keys" ? t("dlp.awsKeys") : rule.name}</td>
                <td>{rule.type}</td>
                <td>{t(rule.action)}</td>
                <td>{t(rule.severity)}</td>
                <td><span className="chip ok">{t(rule.status)}</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <div className="grid-2">
        <section className="card">
          <h2>{t("dlp.ruleTest")}</h2>
          <label className="field">
            <span>{t("common.name")}</span>
            <textarea className="input" rows={5} value={text} onChange={event => setText(event.target.value)} />
          </label>
          <div className="actions" style={{ marginTop: 12 }}>
            <button className="btn primary" type="button" onClick={() => setTested(true)}>{t("dlp.runTest")}</button>
            <button className="btn" type="button">{t("dlp.saveCase")}</button>
          </div>
          {tested ? (
            <div className="card" style={{ marginTop: 12 }}>
              <strong>{t("common.redact")}</strong>
              <p className="mono small">Meu CPF é ***.***.***-** e meu e-mail é ***@***.com</p>
            </div>
          ) : null}
        </section>

        <section className="card">
          <h2>{t("dlp.events")}</h2>
          <p className="muted small">{t("dlp.eventsHint")}</p>
          <table className="table">
            <thead><tr><th>{t("common.time")}</th><th>{t("common.action")}</th><th>{t("dlp.rule")}</th><th>{t("common.provider")}</th></tr></thead>
            <tbody>
              {eventRows.map(row => <tr key={row.join("-")}>{row.slice(0, 4).map((cell, index) => <td key={`${cell}-${index}`}>{cell}</td>)}</tr>)}
            </tbody>
          </table>
        </section>
      </div>
    </>
  );
}
