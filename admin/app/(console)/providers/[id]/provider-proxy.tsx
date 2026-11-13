"use client";

import { LoaderCircle, Network, Plus, X } from "lucide-react";
import { useState } from "react";
import { useI18n } from "@/components/console/i18n-provider";
import type { ProviderAccount, ProviderProxyMode } from "@/lib/providers";

interface ProxyEndpoint {
  id: string;
  label: string;
  address: string;
  port: number;
  country_code: string;
  city_name: string;
  health: string;
  assigned_count: number;
  accounts?: Array<{ id: string; provider_id: string }>;
}

const MODES: ProviderProxyMode[] = ["inherit", "off", "manual", "auto"];

const MODE_KEYS: Record<ProviderProxyMode, string> = {
  inherit: "providers.proxyModeInherit",
  off: "providers.proxyModeOff",
  manual: "providers.proxyModeManual",
  auto: "providers.proxyModeAuto"
};

const MODE_HINT_KEYS: Record<ProviderProxyMode, string> = {
  inherit: "providers.proxyModeInheritHint",
  off: "providers.proxyModeOffHint",
  manual: "providers.proxyModeManualHint",
  auto: "providers.proxyModeAutoHint"
};

const HEALTH_KEYS: Record<string, string> = {
  healthy: "providers.proxyHealthHealthy",
  quarantined: "providers.proxyHealthQuarantined",
  unhealthy: "providers.proxyHealthUnhealthy"
};

function healthClass(health: string): string {
  if (health === "healthy") return "ok";
  if (health === "quarantined") return "warn";
  return "";
}

export function AccountProxyControl({
  providerId,
  account,
  onUpdated
}: {
  providerId: string;
  account: ProviderAccount;
  onUpdated?: () => void;
}) {
  const { t } = useI18n();
  const [mode, setMode] = useState<ProviderProxyMode>(account.proxyMode ?? "inherit");
  const [modeBusy, setModeBusy] = useState(false);
  const [modeError, setModeError] = useState("");
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState("");
  const [endpoints, setEndpoints] = useState<ProxyEndpoint[]>([]);
  const [selected, setSelected] = useState(account.proxy?.id ?? "");
  const [saveBusy, setSaveBusy] = useState(false);
  const [saveError, setSaveError] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [form, setForm] = useState({
    label: "",
    address: "",
    port: "",
    country: "",
    city: "",
    username: "",
    password: ""
  });
  const [createBusy, setCreateBusy] = useState(false);
  const [createError, setCreateError] = useState("");

  async function changeMode(next: ProviderProxyMode) {
    if (!account.id || modeBusy || next === mode) return;
    setModeBusy(true);
    setModeError("");
    try {
      const response = await fetch(
        `/api/admin/providers/${encodeURIComponent(providerId)}/accounts/${encodeURIComponent(account.id)}`,
        {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ proxy_mode: next })
        }
      );
      if (!response.ok) {
        setModeError(t("providers.proxyModeFailed"));
        return;
      }
      setMode(next);
      onUpdated?.();
    } catch {
      setModeError(t("providers.proxyModeFailed"));
    } finally {
      setModeBusy(false);
    }
  }

  async function loadEndpoints() {
    setLoading(true);
    setLoadError("");
    try {
      const response = await fetch("/api/admin/proxies", { cache: "no-store" });
      const body = await response.json().catch(() => null) as { data?: ProxyEndpoint[] } | null;
      if (!response.ok || !Array.isArray(body?.data)) {
        setLoadError(t("providers.proxyLoadFailed"));
        return;
      }
      setEndpoints(body.data);
    } catch {
      setLoadError(t("providers.proxyLoadFailed"));
    } finally {
      setLoading(false);
    }
  }

  function openAssignment() {
    setOpen(true);
    setSaveError("");
    setCreateError("");
    setSelected(account.proxy?.id ?? "");
    void loadEndpoints();
  }

  async function saveAssignment() {
    if (!account.id || !selected || saveBusy) return;
    setSaveBusy(true);
    setSaveError("");
    try {
      const response = await fetch(
        `/api/admin/providers/${encodeURIComponent(providerId)}/accounts/${encodeURIComponent(account.id)}/proxy`,
        {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ proxy_endpoint_id: selected })
        }
      );
      if (response.status === 409) {
        setSaveError(t("providers.proxyLimitReached"));
        return;
      }
      if (!response.ok) {
        setSaveError(t("providers.proxySaveFailed"));
        return;
      }
      setOpen(false);
      onUpdated?.();
    } catch {
      setSaveError(t("providers.proxySaveFailed"));
    } finally {
      setSaveBusy(false);
    }
  }

  async function removeAssignment() {
    if (!account.id || saveBusy) return;
    setSaveBusy(true);
    setSaveError("");
    try {
      const response = await fetch(
        `/api/admin/providers/${encodeURIComponent(providerId)}/accounts/${encodeURIComponent(account.id)}/proxy`,
        { method: "DELETE" }
      );
      if (!response.ok) {
        setSaveError(t("providers.proxySaveFailed"));
        return;
      }
      setOpen(false);
      onUpdated?.();
    } catch {
      setSaveError(t("providers.proxySaveFailed"));
    } finally {
      setSaveBusy(false);
    }
  }

  async function createProxy(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (createBusy) return;
    const port = Number(form.port);
    if (!form.address.trim() || !Number.isInteger(port) || port < 1 || port > 65535) {
      setCreateError(t("providers.proxyCreateFailed"));
      return;
    }
    setCreateBusy(true);
    setCreateError("");
    try {
      const response = await fetch("/api/admin/proxies", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          label: form.label,
          address: form.address.trim(),
          port,
          country_code: form.country,
          city_name: form.city,
          username: form.username,
          password: form.password
        })
      });
      const body = await response.json().catch(() => null) as ProxyEndpoint | null;
      if (!response.ok || !body?.id) {
        setCreateError(t("providers.proxyCreateFailed"));
        return;
      }
      setEndpoints(current => [body, ...current]);
      setSelected(body.id);
      setCreateOpen(false);
      setForm({ label: "", address: "", port: "", country: "", city: "", username: "", password: "" });
    } catch {
      setCreateError(t("providers.proxyCreateFailed"));
    } finally {
      setCreateBusy(false);
    }
  }

  return (
    <div className="proxy-config">
      <div className="proxy-config-head">
        <span className="proxy-config-title">
          <Network aria-hidden="true" size={14} />
          {t("providers.proxyTitle")}
        </span>
        <label className="proxy-mode-field">
          <span className="muted small">{t("providers.proxyMode")}</span>
          <select
            className="input"
            value={mode}
            disabled={!account.id || modeBusy}
            onChange={event => changeMode(event.target.value as ProviderProxyMode)}
            aria-label={`${t("providers.proxyMode")}: ${account.name}`}
          >
            {MODES.map(value => (
              <option key={value} value={value}>{t(MODE_KEYS[value])}</option>
            ))}
          </select>
          {modeBusy ? <LoaderCircle aria-hidden="true" size={14} className="spin" /> : null}
        </label>
        {mode === "auto" ? <span className="chip ok">{t("providers.proxyAutoBadge")}</span> : null}
      </div>
      <p className="helper">{t(MODE_HINT_KEYS[mode])}</p>
      <div className="proxy-config-row">
        <span className="muted small">{t("providers.proxyAssigned")}</span>
        {account.proxy ? (
          <>
            <span className="proxy-endpoint">{account.proxy.address}:{account.proxy.port}</span>
            <span className="muted small">
              {[account.proxy.cityName, account.proxy.countryCode].filter(Boolean).join(", ")}
            </span>
            <span className={`chip ${healthClass(account.proxy.health)}`}>
              {t(HEALTH_KEYS[account.proxy.health] ?? "providers.proxyHealthUnhealthy")}
            </span>
          </>
        ) : (
          <span className="muted small">{t("providers.proxyUnassigned")}</span>
        )}
        <button className="btn" type="button" onClick={openAssignment} disabled={!account.id || mode === "auto"}>
          <Network aria-hidden="true" size={14} />
          {account.proxy ? t("providers.proxyReassign") : t("providers.proxyAssign")}
        </button>
        {account.proxy ? (
          <button className="btn" type="button" onClick={removeAssignment} disabled={!account.id || saveBusy}>
            {t("providers.proxyRemove")}
          </button>
        ) : null}
      </div>
      {modeError ? <p className="small" role="alert" style={{ color: "var(--danger)" }}>{modeError}</p> : null}

      {open ? (
        <div className="modal-backdrop" role="presentation" onClick={event => { if (event.target === event.currentTarget) setOpen(false); }}>
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby={`proxy-modal-${account.id}`}>
            <div className="modal-head">
              <div>
                <h2 id={`proxy-modal-${account.id}`}>{t("providers.proxyModalTitle")}</h2>
                <p className="muted">{t("providers.proxyModalBody")}</p>
              </div>
              <button className="modal-close" type="button" aria-label={t("common.close")} onClick={() => setOpen(false)}>
                <X aria-hidden="true" size={16} />
              </button>
            </div>

            {loading ? (
              <p className="muted"><LoaderCircle aria-hidden="true" size={16} className="spin" /> {t("common.loading")}</p>
            ) : loadError ? (
              <p className="small" role="alert" style={{ color: "var(--danger)" }}>{loadError}</p>
            ) : endpoints.length === 0 ? (
              <div className="empty-state" style={{ marginTop: 8 }}>
                <div className="empty-icon" aria-hidden="true">+</div>
                <h2>{t("providers.proxyEmpty")}</h2>
                <p>{t("providers.proxyEmptyBody")}</p>
              </div>
            ) : (
              <div className="model-list">
                {endpoints.map(endpoint => {
                  const usedBySameProvider = (endpoint.accounts ?? []).some(
                    assigned => assigned.provider_id === providerId && assigned.id !== account.id
                  );
                  const atLimit = usedBySameProvider && account.proxy?.id !== endpoint.id;
                  return (
                    <label className="model-row" key={endpoint.id} style={{ opacity: atLimit ? 0.55 : 1 }}>
                      <input
                        type="radio"
                        name="proxy-endpoint"
                        checked={selected === endpoint.id}
                        disabled={atLimit}
                        onChange={() => setSelected(endpoint.id)}
                      />
                      <span>
                        <strong>{endpoint.address}:{endpoint.port}</strong>
                        <small>
                          {[endpoint.label, endpoint.city_name, endpoint.country_code].filter(Boolean).join(" · ")}
                        </small>
                        <small>{t("providers.proxyAssignedCount", { count: endpoint.assigned_count })}</small>
                      </span>
                      <span className={`chip ${healthClass(endpoint.health)}`}>
                        {t(HEALTH_KEYS[endpoint.health] ?? "providers.proxyHealthUnhealthy")}
                      </span>
                    </label>
                  );
                })}
              </div>
            )}

            {loadError ? (
              <button className="btn" type="button" onClick={() => void loadEndpoints()}>{t("common.retry")}</button>
            ) : null}
            {saveError ? <p className="small" role="alert" style={{ color: "var(--danger)" }}>{saveError}</p> : null}

            {createOpen ? (
              <form onSubmit={createProxy} style={{ marginTop: 12 }}>
                <div className="form-grid">
                  <label className="field">
                    <span>{t("providers.proxyAddress")}</span>
                    <input className="input" required value={form.address} placeholder={t("providers.proxyAddressPlaceholder")} onChange={event => setForm({ ...form, address: event.target.value })} />
                  </label>
                  <label className="field">
                    <span>{t("providers.proxyPort")}</span>
                    <input className="input" required inputMode="numeric" value={form.port} placeholder={t("providers.proxyPortPlaceholder")} onChange={event => setForm({ ...form, port: event.target.value })} />
                  </label>
                  <label className="field">
                    <span>{t("providers.proxyLabel")}</span>
                    <input className="input" value={form.label} placeholder={t("providers.proxyLabelPlaceholder")} onChange={event => setForm({ ...form, label: event.target.value })} />
                  </label>
                  <label className="field">
                    <span>{t("providers.proxyCountry")}</span>
                    <input className="input" value={form.country} placeholder={t("providers.proxyCountryPlaceholder")} onChange={event => setForm({ ...form, country: event.target.value })} />
                  </label>
                  <label className="field">
                    <span>{t("providers.proxyCity")}</span>
                    <input className="input" value={form.city} placeholder={t("providers.proxyCityPlaceholder")} onChange={event => setForm({ ...form, city: event.target.value })} />
                  </label>
                  <label className="field">
                    <span>{t("providers.proxyUsername")}</span>
                    <input className="input" autoComplete="off" value={form.username} onChange={event => setForm({ ...form, username: event.target.value })} />
                  </label>
                  <label className="field">
                    <span>{t("providers.proxyPassword")}</span>
                    <input className="input" type="password" autoComplete="new-password" value={form.password} onChange={event => setForm({ ...form, password: event.target.value })} />
                  </label>
                </div>
                {createError ? <p className="small" role="alert" style={{ color: "var(--danger)" }}>{createError}</p> : null}
                <div className="modal-foot">
                  <button className="btn" type="button" onClick={() => setCreateOpen(false)}>{t("common.cancel")}</button>
                  <button className="btn primary" type="submit" disabled={createBusy}>
                    {createBusy ? <LoaderCircle aria-hidden="true" size={16} className="spin" /> : null}
                    {t("providers.proxyCreate")}
                  </button>
                </div>
              </form>
            ) : (
              <button className="btn" type="button" onClick={() => setCreateOpen(true)} style={{ marginTop: 8 }}>
                <Plus aria-hidden="true" size={14} />
                {t("providers.proxyCreate")}
              </button>
            )}

            <div className="modal-foot">
              {account.proxy ? (
                <button className="btn" type="button" onClick={removeAssignment} disabled={saveBusy}>
                  {t("providers.proxyRemove")}
                </button>
              ) : <span />}
              <button className="btn" type="button" onClick={() => setOpen(false)}>{t("common.cancel")}</button>
              <button className="btn primary" type="button" onClick={saveAssignment} disabled={saveBusy || !selected || loading}>
                {saveBusy ? <LoaderCircle aria-hidden="true" size={16} className="spin" /> : null}
                {t("providers.proxySave")}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
