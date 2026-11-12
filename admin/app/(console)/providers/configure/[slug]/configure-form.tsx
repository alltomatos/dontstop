"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { ArrowLeft, ExternalLink, LoaderCircle, Save } from "lucide-react";
import { Breadcrumbs, PageHeader } from "@/components/console/page";
import { useI18n } from "@/components/console/i18n-provider";
import { PROVIDER_SETUP_GUIDE } from "@/lib/catalog";
import type { CatalogEntry } from "@/lib/admin-api";

export function ConfigureProvider({ entry }: { entry: CatalogEntry }) {
  const { t, tp, guideUrl } = useI18n();
  const router = useRouter();
  const [name, setName] = useState(entry.display_name);
  const [credential, setCredential] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const modelCount = entry.models?.length ?? 0;

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!credential.trim() || submitting) return;
    setSubmitting(true);
    setError("");
    try {
      const response = await fetch(
        `/api/admin/provider-catalog/${encodeURIComponent(entry.slug)}/connect`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ credential: credential.trim(), account_name: name.trim() || entry.display_name })
        }
      );
      if (response.status === 201) {
        const provider = await response.json() as { id?: string };
        if (provider.id) {
          router.push(`/providers/${provider.id}`);
          router.refresh();
          return;
        }
      }
      if (response.status === 409) {
        setError(t("providers.configureConflict"));
        router.refresh();
        return;
      }
      if (response.status === 404) {
        setError(t("providers.configureUnsupported"));
        return;
      }
      if (response.status === 503) {
        setError(t("common.gatewayError"));
        return;
      }
      setError(t("providers.configureFailed"));
    } catch {
      setError(t("providers.configureFailed"));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <>
      <Breadcrumbs
        items={[
          { label: t("nav.providers"), href: "/providers" },
          { label: t("providers.catalog"), href: "/providers/catalog" },
          { label: t("providers.configureTitle") }
        ]}
      />
      <PageHeader
        title={t("providers.configureTitleFor", { name: entry.display_name })}
        subtitle={t("providers.configureSubtitle")}
        actions={(
          <Link className="btn" href="/providers/catalog">
            <ArrowLeft aria-hidden="true" size={16} />
            {t("providers.backToCatalog")}
          </Link>
        )}
      />

      <div className="detail-grid">
        <div className="card">
          <h2>{t("providers.configureCredentialTitle")}</h2>
          <form onSubmit={submit}>
            <div className="form-grid">
              <label className="field">
                <span>{t("providers.accountName")}</span>
                <input
                  className="input"
                  required
                  value={name}
                  onChange={event => setName(event.target.value)}
                  placeholder={entry.display_name}
                />
              </label>
              <label className="field">
                <span>{t("providers.accountCredential")}</span>
                <input
                  className="input"
                  required
                  type="password"
                  value={credential}
                  onChange={event => setCredential(event.target.value)}
                  placeholder={t("providers.accountCredentialPlaceholder")}
                />
              </label>
            </div>
            <p className="muted small">{t("providers.configureCredentialHint")}</p>
            {error ? <p className="small" role="alert" style={{ color: "var(--danger)" }}>{error}</p> : null}
            <div className="modal-foot">
              <a className="btn" href={guideUrl(PROVIDER_SETUP_GUIDE, entry.slug)} target="_blank" rel="noopener noreferrer">
                {t("providers.setupGuide")}
                <ExternalLink aria-hidden="true" size={14} />
              </a>
              <button className="btn primary" type="submit" disabled={submitting || !credential.trim()}>
                {submitting ? <LoaderCircle aria-hidden="true" size={16} className="spin" /> : <Save aria-hidden="true" size={16} />}
                {submitting ? t("providers.configuring") : t("providers.configureSubmit")}
              </button>
            </div>
          </form>
        </div>

        <aside className="card">
          <h2>{t("providers.providerSummary")}</h2>
          <div className="list">
            <div className="list-item"><span>{t("providers.protocol")}</span><span>{entry.protocol}</span></div>
            <div className="list-item"><span>{t("providers.baseUrl")}</span><span>{entry.default_base_url}</span></div>
            <div className="list-item"><span>{t("providers.authType")}</span><span>{entry.auth_type}</span></div>
            <div className="list-item"><span>{t("providers.models")}</span><span>{modelCount}</span></div>
          </div>
        </aside>
      </div>
    </>
  );
}
