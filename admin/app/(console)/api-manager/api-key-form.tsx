"use client";

import { useState } from "react";
import { useI18n } from "@/components/console/i18n-provider";
import type { AdminAPIKey } from "@/lib/admin-api";
import {
  emptyFormValues,
  scopeOptions,
  toFormValues,
  toPayload,
  validateForm,
  type APIKeyFormValues,
  type APIKeyPayload,
} from "@/lib/api-keys";

export function APIKeyForm({
  mode,
  initial,
  catalog,
  saving,
  serverError,
  onSubmit,
  onCancel,
}: {
  mode: "create" | "edit";
  initial: AdminAPIKey | null;
  catalog: string[];
  saving: boolean;
  serverError: string;
  onSubmit: (payload: APIKeyPayload) => void;
  onCancel: () => void;
}) {
  const { t } = useI18n();
  const [values, setValues] = useState<APIKeyFormValues>(() => (initial ? toFormValues(initial) : emptyFormValues()));
  const [error, setError] = useState("");
  const scopes = scopeOptions(catalog, initial);

  function update<K extends keyof APIKeyFormValues>(field: K, value: APIKeyFormValues[K]) {
    setValues(current => ({ ...current, [field]: value }));
  }

  function toggleScope(scope: string) {
    setValues(current => ({
      ...current,
      scopes: current.scopes.includes(scope)
        ? current.scopes.filter(item => item !== scope)
        : [...current.scopes, scope],
    }));
  }

  function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const issue = validateForm(values);
    if (issue) {
      setError(t(issue.key, issue.params));
      return;
    }
    setError("");
    onSubmit(toPayload(values));
  }

  return (
    <form onSubmit={submit}>
      <div className="form-grid">
        <label className="field">
          <span>{t("common.name")}</span>
          <input
            className="input"
            value={values.name}
            maxLength={80}
            placeholder={t("api.namePlaceholder")}
            onChange={event => update("name", event.target.value)}
            autoFocus
            required
          />
        </label>
      </div>

      <fieldset className="field" style={{ border: 0, padding: 0, margin: "12px 0" }}>
        <legend className="small muted" style={{ marginBottom: 8 }}>{t("api.scopes")}</legend>
        <div className="grid-2">
          {scopes.map(scope => (
            <label key={scope} style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <input type="checkbox" checked={values.scopes.includes(scope)} onChange={() => toggleScope(scope)} />
              <span className="mono small">{scope}</span>
            </label>
          ))}
        </div>
        <p className="small muted">{t("api.scopesHint")}</p>
      </fieldset>

      <div className="form-grid">
        <label className="field">
          <span>{t("api.expires")}</span>
          <span style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <input
              type="checkbox"
              checked={values.neverExpires}
              onChange={event => update("neverExpires", event.target.checked)}
            />
            <span className="small">{t("api.noExpiration")}</span>
          </span>
          <input
            className="input"
            type="datetime-local"
            value={values.expiresAt}
            disabled={values.neverExpires}
            onChange={event => update("expiresAt", event.target.value)}
          />
        </label>
        <label className="field">
          <span>{t("api.ipAllowlist")}</span>
          <textarea
            className="input"
            rows={3}
            value={values.ipAllowlist}
            placeholder={t("api.ipPlaceholder")}
            onChange={event => update("ipAllowlist", event.target.value)}
          />
          <span className="small muted">{t("api.ipHint")}</span>
        </label>
      </div>

      <fieldset className="field" style={{ border: 0, padding: 0, margin: "12px 0" }}>
        <legend className="small muted" style={{ marginBottom: 8 }}>{t("api.quotas")}</legend>
        <div className="grid-3">
          <label className="field"><span>{t("api.rpm")}</span><input className="input" type="number" min="0" step="1" value={values.rpmLimit} onChange={event => update("rpmLimit", event.target.value)} /></label>
          <label className="field"><span>{t("api.tpm")}</span><input className="input" type="number" min="0" step="1" value={values.tpmLimit} onChange={event => update("tpmLimit", event.target.value)} /></label>
          <label className="field"><span>{t("api.usdMonthly")}</span><input className="input" type="number" min="0" step="0.01" value={values.usdMonthly} onChange={event => update("usdMonthly", event.target.value)} /></label>
          <label className="field"><span>{t("api.usdDaily")}</span><input className="input" type="number" min="0" step="0.01" value={values.usdDaily} onChange={event => update("usdDaily", event.target.value)} /></label>
          <label className="field"><span>{t("api.usdWeekly")}</span><input className="input" type="number" min="0" step="0.01" value={values.usdWeekly} onChange={event => update("usdWeekly", event.target.value)} /></label>
        </div>
      </fieldset>

      {error ? <p className="small" role="alert" style={{ color: "var(--danger)" }}>{error}</p> : null}
      {serverError ? <p className="small" role="alert" style={{ color: "var(--danger)" }}>{serverError}</p> : null}

      <div className="modal-foot">
        <button className="btn" type="button" onClick={onCancel} disabled={saving}>{t("common.cancel")}</button>
        <button className="btn primary" type="submit" disabled={saving}>
          {saving ? t("common.loading") : mode === "create" ? t("common.createKey") : t("api.save")}
        </button>
      </div>
    </form>
  );
}
