"use client";

import { useState } from "react";
import { Cloud, LoaderCircle, RefreshCw, Settings2, X } from "lucide-react";
import { useI18n } from "@/components/console/i18n-provider";
import type { WebshareStatus } from "@/lib/proxies";

interface TestResult {
  ok: boolean;
  plan_id: string;
  state: string;
  available: number;
}

interface SyncResult {
  imported: number;
  total: number;
  status: string;
  error?: string;
}

function syncErrorKey(message: string): string {
  if (message.includes("invalid api key")) return "webshare.unauthorized";
  if (message.includes("plan access")) return "webshare.forbidden";
  if (message.includes("rate limited")) return "webshare.rateLimited";
  if (message.includes("unavailable") || message.includes("invalid webshare")) return "webshare.unavailable";
  return "webshare.failed";
}

function errorKey(type: string): string {
  if (type.includes("unauthorized")) return "webshare.unauthorized";
  if (type.includes("forbidden")) return "webshare.forbidden";
  if (type.includes("rate_limited")) return "webshare.rateLimited";
  if (type.includes("unavailable") || type.includes("invalid_response")) return "webshare.unavailable";
  return "webshare.failed";
}

async function responseError(response: Response): Promise<string> {
  const body = await response.json().catch(() => null) as { error?: { type?: string } } | null;
  return errorKey(body?.error?.type ?? "");
}

export function WebsharePanel({
  status,
  onRefresh
}: {
  status: WebshareStatus;
  onRefresh: () => void;
}) {
  const { t, tp } = useI18n();
  const [open, setOpen] = useState(false);
  const [apiKey, setApiKey] = useState("");
  const [planID, setPlanID] = useState(status.planId);
  const [testBusy, setTestBusy] = useState(false);
  const [testResult, setTestResult] = useState<TestResult | null>(null);
  const [error, setError] = useState("");
  const [saveBusy, setSaveBusy] = useState(false);
  const [syncBusy, setSyncBusy] = useState(false);
  const [message, setMessage] = useState("");

  function openModal() {
    setOpen(true);
    setPlanID(status.planId);
    setApiKey("");
    setTestResult(null);
    setError("");
    setMessage("");
  }

  async function testConnection() {
    setTestBusy(true);
    setError("");
    setTestResult(null);
    try {
      const response = await fetch("/api/admin/proxies/webshare/test", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ api_key: apiKey, plan_id: planID })
      });
      if (!response.ok) {
        setError(await responseError(response));
        return;
      }
      setTestResult(await response.json() as TestResult);
    } catch {
      setError(t("webshare.failed"));
    } finally {
      setTestBusy(false);
    }
  }

  async function saveAndSync() {
    if (saveBusy || syncBusy) return;
    setSaveBusy(true);
    setError("");
    setMessage("");
    try {
      const response = await fetch("/api/admin/proxies/webshare", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ api_key: apiKey, plan_id: planID })
      });
      if (!response.ok) {
        setError(await responseError(response));
        return;
      }
      setSaveBusy(false);
      const synced = await runSync("webshare.savedSyncDone");
      if (synced) setOpen(false);
    } catch {
      setError(t("webshare.failed"));
    } finally {
      setSaveBusy(false);
    }
  }

  async function syncNow() {
    if (syncBusy) return;
    setMessage("");
    setError("");
    await runSync("webshare.synced");
  }

  async function runSync(template: string): Promise<boolean> {
    setSyncBusy(true);
    try {
      const response = await fetch("/api/admin/proxies/webshare/sync", { method: "POST" });
      if (!response.ok) {
        setError(await responseError(response));
        return false;
      }
      const result = await response.json() as SyncResult;
      setMessage(t(template, { imported: result.imported, total: result.total }));
      onRefresh();
      return true;
    } catch {
      setError(t("webshare.failed"));
      return false;
    } finally {
      setSyncBusy(false);
    }
  }

  const statusChip = !status.configured
    ? <span className="chip">{t("webshare.notConfigured")}</span>
    : status.lastSyncStatus === "failed"
      ? <span className="chip warn">{t("webshare.errorState")}</span>
      : <span className="chip ok">{t("webshare.connected")}</span>;

  return (
    <div className="card" style={{ marginBottom: 16 }}>
      <div className="row">
        <div className="proxy-config-title">
          <Cloud aria-hidden="true" size={16} />
          <h2>{t("webshare.title")}</h2>
          {statusChip}
        </div>
        <div className="actions">
          <button className="btn" type="button" onClick={openModal}>
            <Settings2 aria-hidden="true" size={16} />
            {status.configured ? t("webshare.configure") : t("webshare.setup")}
          </button>
          {status.configured ? (
            <button className="btn" type="button" onClick={syncNow} disabled={syncBusy}>
              {syncBusy ? <LoaderCircle aria-hidden="true" size={16} className="spin" /> : <RefreshCw aria-hidden="true" size={16} />}
              {t("webshare.importNow")}
            </button>
          ) : null}
          <button className="btn primary" type="button" onClick={syncNow} disabled={!status.configured || syncBusy}>
            {syncBusy ? <LoaderCircle aria-hidden="true" size={16} className="spin" /> : <RefreshCw aria-hidden="true" size={16} />}
            {syncBusy ? t("webshare.syncing") : t("webshare.syncNow")}
          </button>
        </div>
      </div>

      <div className="list" style={{ marginTop: 10 }}>
        <div className="list-item"><span>{t("webshare.plan")}</span><span>{status.planId || "—"}</span></div>
        <div className="list-item"><span>{t("webshare.proxies")}</span><span>{tp("webshare.proxyCount", status.proxyCount)}</span></div>
        <div className="list-item"><span>{t("webshare.healthy")}</span><span>{status.healthyCount}</span></div>
        <div className="list-item">
          <span>{t("webshare.lastSync")}</span>
          <span>{status.lastSyncAt ? new Date(status.lastSyncAt).toLocaleString() : "—"}</span>
        </div>
        <div className="list-item">
          <span>{t("webshare.imported")}</span>
          <span>{status.lastSyncImported} / {status.lastSyncTotal}</span>
        </div>
      </div>
      {status.lastSyncError ? (
        <p className="small" role="status" style={{ color: "var(--warning)" }}>{t(syncErrorKey(status.lastSyncError))}</p>
      ) : null}
      {message ? <p className="small" role="status">{message}</p> : null}
      {error && !open ? <p className="small" role="alert" style={{ color: "var(--danger)" }}>{error}</p> : null}

      {open ? (
        <div className="modal-backdrop" role="presentation" onClick={event => { if (event.target === event.currentTarget) setOpen(false); }}>
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="webshare-modal-title">
            <div className="modal-head">
              <div>
                <h2 id="webshare-modal-title">{t("webshare.modalTitle")}</h2>
                <p className="muted">{t("webshare.modalBody")}</p>
              </div>
              <button className="modal-close" type="button" aria-label={t("common.close")} onClick={() => setOpen(false)}>
                <X aria-hidden="true" size={16} />
              </button>
            </div>
            <div className="form-grid">
              <label className="field">
                <span>{t("webshare.apiKey")}</span>
                <input
                  className="input"
                  type="password"
                  autoComplete="new-password"
                  value={apiKey}
                  onChange={event => setApiKey(event.target.value)}
                  placeholder={status.configured ? t("webshare.apiKeyKeep") : t("webshare.apiKeyPlaceholder")}
                />
              </label>
              <label className="field">
                <span>{t("webshare.planId")}</span>
                <input
                  className="input"
                  value={planID}
                  onChange={event => setPlanID(event.target.value)}
                  placeholder={t("webshare.planIdPlaceholder")}
                />
              </label>
            </div>
            <p className="helper">{t("webshare.keyHint")}</p>
            <div className="actions" style={{ marginTop: 8 }}>
              <button className="btn" type="button" onClick={testConnection} disabled={testBusy || (!apiKey && !status.configured)}>
                {testBusy ? <LoaderCircle aria-hidden="true" size={16} className="spin" /> : null}
                {testBusy ? t("webshare.testing") : t("webshare.test")}
              </button>
              {testResult ? (
                <span className="chip ok" role="status">
                  {t("webshare.testOK", { plan: testResult.plan_id || "—", state: testResult.state, count: testResult.available })}
                </span>
              ) : null}
            </div>
            {error ? <p className="small" role="alert" style={{ color: "var(--danger)" }}>{error}</p> : null}
            <div className="modal-foot">
              <button className="btn" type="button" onClick={() => setOpen(false)}>{t("common.cancel")}</button>
              <button className="btn primary" type="button" onClick={saveAndSync} disabled={saveBusy || syncBusy || (!apiKey && !status.configured)}>
                {saveBusy || syncBusy ? <LoaderCircle aria-hidden="true" size={16} className="spin" /> : <RefreshCw aria-hidden="true" size={16} />}
                {saveBusy ? t("webshare.saving") : syncBusy ? t("webshare.syncing") : t("webshare.saveAndSync")}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
