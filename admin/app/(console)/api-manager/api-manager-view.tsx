"use client";

import { useCallback, useEffect, useState } from "react";
import { Plus, X } from "lucide-react";
import { Breadcrumbs, PageHeader } from "@/components/console/page";
import { CopyButton, Tabs } from "@/components/console/ui";
import { useI18n } from "@/components/console/i18n-provider";
import type { AdminAPIKey } from "@/lib/admin-api";
import {
  countExpiring,
  formatDate,
  statusClass,
  statusKey,
  type APIKeyPayload,
} from "@/lib/api-keys";
import { APIKeyForm } from "./api-key-form";

type CreatedResponse = AdminAPIKey & { key: string };
type ConfirmAction = { kind: "rotate" | "revoke"; key: AdminAPIKey };
type SecretState = { key: string; mode: "create" | "rotate"; name: string };

export function ApiManagerView() {
  const { t } = useI18n();
  const [tab, setTab] = useState("keys");
  const [keys, setKeys] = useState<AdminAPIKey[]>([]);
  const [scopes, setScopes] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [actionError, setActionError] = useState("");
  const [formMode, setFormMode] = useState<"create" | "edit" | null>(null);
  const [editing, setEditing] = useState<AdminAPIKey | null>(null);
  const [saving, setSaving] = useState(false);
  const [secret, setSecret] = useState<SecretState | null>(null);
  const [confirm, setConfirm] = useState<ConfirmAction | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  const request = useCallback(async <T,>(url: string, init?: RequestInit): Promise<T> => {
    const response = await fetch(url, { cache: "no-store", ...init });
    const body = await response.json().catch(() => null) as { error?: { message?: string } } | null;
    if (!response.ok) {
      throw new Error(body?.error?.message || t("api.actionError"));
    }
    return body as T;
  }, [t]);

  const load = useCallback(async () => {
    try {
      const list = await request<{ data?: AdminAPIKey[] }>("/api/admin/api-keys");
      setKeys(Array.isArray(list.data) ? list.data : []);
      setError("");
      try {
        const catalog = await request<{ data?: string[] }>("/api/admin/api-keys/scopes");
        setScopes(Array.isArray(catalog.data) ? catalog.data : []);
      } catch {
        setScopes([]);
        setActionError(t("api.loadError"));
      }
    } catch (requestError) {
      setKeys([]);
      setError(requestError instanceof Error ? requestError.message : t("api.loadError"));
    } finally {
      setLoading(false);
    }
  }, [request, t]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- initial async fetch
    void load();
  }, [load]);

  function openCreate() {
    setActionError("");
    setEditing(null);
    setFormMode("create");
  }

  function openEdit(key: AdminAPIKey) {
    setActionError("");
    setEditing(key);
    setFormMode("edit");
  }

  async function submitForm(payload: APIKeyPayload) {
    setSaving(true);
    setActionError("");
    try {
      if (formMode === "create") {
        const created = await request<CreatedResponse>("/api/admin/api-keys", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        });
        setFormMode(null);
        setSecret({ key: created.key, mode: "create", name: created.name });
      } else if (editing) {
        await request<AdminAPIKey>(`/api/admin/api-keys/${encodeURIComponent(editing.id)}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        });
        setFormMode(null);
        setEditing(null);
      }
      await load();
    } catch (requestError) {
      setActionError(requestError instanceof Error ? requestError.message : t("api.actionError"));
    } finally {
      setSaving(false);
    }
  }

  async function runConfirm() {
    if (!confirm) return;
    setBusyId(confirm.key.id);
    setActionError("");
    try {
      if (confirm.kind === "rotate") {
        const rotated = await request<CreatedResponse>(`/api/admin/api-keys/${encodeURIComponent(confirm.key.id)}/rotate`, { method: "POST" });
        setSecret({ key: rotated.key, mode: "rotate", name: rotated.name });
      } else {
        await request<AdminAPIKey>(`/api/admin/api-keys/${encodeURIComponent(confirm.key.id)}/revoke`, { method: "POST" });
      }
      setConfirm(null);
      await load();
    } catch (requestError) {
      setActionError(requestError instanceof Error ? requestError.message : t("api.actionError"));
    } finally {
      setBusyId(null);
    }
  }

  const active = keys.filter(key => key.status === "active").length;
  const expiring = countExpiring(keys);
  const revoked = keys.filter(key => key.status === "revoked").length;

  function quotaText(key: AdminAPIKey): string {
    const parts = [
      key.rpm_limit != null ? `${t("api.rpm")} ${key.rpm_limit}` : null,
      key.tpm_limit != null ? `${t("api.tpm")} ${key.tpm_limit}` : null,
      key.usd_monthly_limit != null ? `${t("api.usdMonthly")} ${key.usd_monthly_limit}` : null,
      key.usd_daily_limit != null ? `${t("api.usdDaily")} ${key.usd_daily_limit}` : null,
      key.usd_weekly_limit != null ? `${t("api.usdWeekly")} ${key.usd_weekly_limit}` : null,
    ].filter(Boolean);
    return parts.length ? parts.join(" | ") : "-";
  }
  return (
    <>
      <Breadcrumbs items={[{ label: t("nav.apiManager") }]} />
      <PageHeader
        title={t("nav.apiManager")}
        subtitle={t("api.subtitle")}
        actions={<button className="btn primary" type="button" onClick={openCreate}><Plus aria-hidden="true" size={16} />{t("common.newKey")}</button>}
      />

      <div className="kpis">
        {[
          [t("api.totalKeys"), String(keys.length)],
          [t("api.activeKeys"), String(active)],
          [t("api.expiringKeys"), String(expiring)],
          [t("api.revokedKeys"), String(revoked)],
        ].map(([label, value]) => (
          <div className="card kpi" key={label}>
            <div className="label">{label}</div>
            <div className="value num">{value}</div>
          </div>
        ))}
      </div>

      <Tabs
        value={tab}
        onChange={setTab}
        items={[
          { id: "keys", label: t("api.keys") },
          { id: "scopes", label: t("api.scopes") },
        ]}
      />

      {actionError ? <p className="small" role="alert" style={{ color: "var(--danger)" }}>{actionError}</p> : null}

      {tab === "keys" ? (
        loading ? (
          <div className="stack" aria-busy="true" aria-live="polite">
            <div className="skeleton skeleton-card" />
            <div className="skeleton skeleton-table" />
          </div>
        ) : error ? (
          <section className="empty-state">
            <div className="empty-icon" aria-hidden="true">!</div>
            <h2>{t("api.loadError")}</h2>
            <p>{error}</p>
            <button className="btn primary" type="button" onClick={() => { setLoading(true); void load(); }}>{t("common.retry")}</button>
          </section>
        ) : keys.length === 0 ? (
          <section className="empty-state">
            <div className="empty-icon" aria-hidden="true">+</div>
            <h2>{t("api.emptyKeys")}</h2>
            <p>{t("api.emptyKeysHint")}</p>
            <button className="btn primary" type="button" onClick={openCreate}>{t("common.createKey")}</button>
          </section>
        ) : (
          <section className="card" style={{ overflowX: "auto" }}>
            <table className="table">
              <thead>
                <tr>
                  <th>{t("common.name")}</th>
                  <th>{t("api.scopes")}</th>
                  <th>{t("common.status")}</th>
                  <th>{t("api.expires")}</th>
                  <th>{t("api.ipAllowlist")}</th>
                  <th>{t("api.quotas")}</th>
                  <th>{t("api.createdAt")}</th>
                  <th>{t("api.lastRotation")}</th>
                  <th>{t("api.actions")}</th>
                </tr>
              </thead>
              <tbody>
                {keys.map(key => (
                  <tr key={key.id}>
                    <td><div>{key.name}</div><div className="mono small muted">{key.prefix}...</div></td>
                    <td className="mono small">{key.scopes.join(", ")}</td>
                    <td><span className={statusClass(key.status)}>{t(statusKey(key.status))}</span></td>
                    <td className="small">{key.expires_at ? formatDate(key.expires_at) : t("api.noExpiration")}</td>
                    <td className="mono small">{key.ip_allowlist.length ? key.ip_allowlist.join(", ") : "-"}</td>
                    <td className="small">{quotaText(key)}</td>
                    <td className="small">{formatDate(key.created_at)}</td>
                    <td className="small">{key.last_rotated_at ? formatDate(key.last_rotated_at) : "-"}</td>
                    <td>
                      <div className="actions">
                        <button className="btn" type="button" onClick={() => openEdit(key)} aria-label={`${t("common.edit")} ${key.name}`}>{t("common.edit")}</button>
                        <button className="btn" type="button" disabled={key.status !== "active" || busyId === key.id} onClick={() => setConfirm({ kind: "rotate", key })} aria-label={`${t("api.rotate")} ${key.name}`}>{t("api.rotate")}</button>
                        <button className="btn" type="button" disabled={key.status === "revoked" || busyId === key.id} onClick={() => setConfirm({ kind: "revoke", key })} aria-label={`${t("api.revoke")} ${key.name}`}>{t("api.revoke")}</button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>
        )
      ) : null}

      {tab === "scopes" ? (
        <section className="card">
          <table className="table">
            <thead><tr><th>{t("api.scopes")}</th><th>{t("common.status")}</th></tr></thead>
            <tbody>
              {scopes.length === 0 ? <tr><td colSpan={2}>{t("common.noData")}</td></tr> : scopes.map(scope => (
                <tr key={scope}><td className="mono">{scope}</td><td><span className="chip ok">{t("common.active")}</span></td></tr>
              ))}
            </tbody>
          </table>
        </section>
      ) : null}

      {formMode ? (
        <div className="modal-backdrop" role="presentation" onClick={event => { if (event.target === event.currentTarget && !saving) setFormMode(null); }}>
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="api-key-form-title">
            <div className="modal-head">
              <div>
                <h2 id="api-key-form-title">{formMode === "create" ? t("api.createTitle") : t("api.editTitle")}</h2>
                <p className="muted">{formMode === "create" ? t("api.subtitle") : t("api.scopesHint")}</p>
              </div>
              <button className="modal-close" type="button" aria-label={t("common.close")} onClick={() => { if (!saving) setFormMode(null); }}><X aria-hidden="true" size={16} /></button>
            </div>
            <APIKeyForm
              key={editing?.id ?? "create"}
              mode={formMode}
              initial={editing}
              catalog={scopes}
              saving={saving}
              serverError={actionError}
              onSubmit={payload => void submitForm(payload)}
              onCancel={() => { if (!saving) setFormMode(null); }}
            />
          </div>
        </div>
      ) : null}

      {secret ? (
        <div className="modal-backdrop" role="presentation">
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="api-key-secret-title">
            <div className="modal-head">
              <div>
                <h2 id="api-key-secret-title">{secret.mode === "create" ? t("api.secretTitle") : t("api.secretRotatedTitle")}</h2>
                <p className="muted">{t("api.secretOnce")}</p>
              </div>
            </div>
            <p className="small" style={{ color: "var(--danger)" }}>{t("api.secretWarning")}</p>
            <div className="codebox" style={{ wordBreak: "break-all" }}>{secret.key}</div>
            <div className="actions" style={{ marginTop: 12 }}>
              <CopyButton value={secret.key} label={t("common.copyKey")} />
              <button className="btn primary" type="button" onClick={() => setSecret(null)}>{t("common.done")}</button>
            </div>
          </div>
        </div>
      ) : null}

      {confirm ? (
        <div className="modal-backdrop" role="presentation" onClick={event => { if (event.target === event.currentTarget && !busyId) setConfirm(null); }}>
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="api-key-confirm-title">
            <div className="modal-head">
              <div>
                <h2 id="api-key-confirm-title">{confirm.kind === "rotate" ? t("api.confirmRotateTitle") : t("api.confirmRevokeTitle")}</h2>
                <p className="muted mono">{confirm.key.name}</p>
              </div>
            </div>
            <p>{confirm.kind === "rotate" ? t("api.confirmRotateBody") : t("api.confirmRevokeBody")}</p>
            <div className="modal-foot">
              <button className="btn" type="button" disabled={Boolean(busyId)} onClick={() => setConfirm(null)}>{t("common.cancel")}</button>
              <button className="btn primary" type="button" disabled={Boolean(busyId)} onClick={() => void runConfirm()}>{busyId ? t("common.loading") : t("api.confirm")}</button>
            </div>
          </div>
        </div>
      ) : null}
    </>
  );
}
