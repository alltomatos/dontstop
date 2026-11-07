"use client";

import Link from "next/link";
import { useState } from "react";
import { ArrowLeft, ExternalLink, Plus, X } from "lucide-react";
import { Breadcrumbs } from "@/components/console/page";
import { OriginChips, StatusChip } from "@/components/console/origin-chips";
import { useI18n } from "@/components/console/i18n-provider";
import type { Provider, ProviderAccount } from "@/lib/providers";

function maskCredential(value: string): string {
  const clean = value.trim();
  if (!clean) return "••••";
  const suffix = clean.slice(-4);
  return `${clean.startsWith("sk-") ? "sk-..." : "••••"}${suffix}`;
}

export function ProviderDetail({ provider }: { provider: Provider }) {
  const { t, tp, guideUrl } = useI18n();
  const [accounts, setAccounts] = useState<ProviderAccount[]>(provider.accounts);
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [credential, setCredential] = useState("");
  const [accountError, setAccountError] = useState("");

  async function addAccount(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!name.trim() || !credential.trim()) return;
    setAccountError("");
    const fallback = () => {
      setAccounts(current => [
        ...current,
        { name: name.trim(), meta: maskCredential(credential), status: "ok", statusKey: "common.healthy" }
      ]);
      setName("");
      setCredential("");
      setOpen(false);
    };
    try {
      const response = await fetch(`/api/admin/providers/${encodeURIComponent(provider.id)}/accounts`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: name.trim(), credential })
      });
      if (response.status === 503) {
        fallback();
        return;
      }
      if (!response.ok) {
        setAccountError(t("common.error"));
        return;
      }
      const account = await response.json() as { id: string; name: string; status: string; base_url?: string };
      setAccounts(current => [
        ...current,
        {
          name: account.name,
          meta: account.base_url || account.id,
          status: account.status === "cooldown" ? "warn" : "ok",
          statusKey: account.status === "cooldown" ? "common.cooldown" : "common.healthy"
        }
      ]);
      setName("");
      setCredential("");
      setOpen(false);
    } catch {
      fallback();
    }
  }

  return (
    <>
      <Breadcrumbs
        items={[
          { label: t("nav.providers"), href: "/providers" },
          ...(provider.configured ? [] : [{ label: t("providers.catalog"), href: "/providers/catalog" }]),
          { label: provider.name }
        ]}
      />
      <div className="page-header">
        <div className="provider-detail-title">
          <span className="provider-icon">{provider.icon}</span>
          <div>
            <h1>{provider.name}</h1>
            <p className="sub">{t("providers.detailSubtitle")}</p>
          </div>
          <StatusChip status={provider.status} label={t(provider.statusKey)} />
        </div>
        <div className="actions">
          <Link className="btn" href={provider.configured ? "/providers" : "/providers/catalog"}>
            <ArrowLeft aria-hidden="true" size={16} />
            {provider.configured ? t("providers.backToProviders") : t("providers.backToCatalog")}
          </Link>
          <a className="btn" href={guideUrl(provider.guide)} target="_blank" rel="noopener noreferrer">
            {t("providers.guideLink")}
          </a>
          <a className="btn" href={provider.signupUrl} target="_blank" rel="noopener noreferrer">
            {t("providers.signupLink")}
            <ExternalLink aria-hidden="true" size={14} />
          </a>
          <button className="btn" type="button">{t("common.test")}</button>
          <button className="btn primary" type="button" onClick={() => setOpen(true)}>
            <Plus aria-hidden="true" size={16} />
            {t("providers.addAccount")}
          </button>
        </div>
      </div>

      <div className="detail-grid">
        <div className="card">
          <div className="row">
            <h2>{t("providers.linkedAccounts")}</h2>
            <span className="chip">{tp("providers.accountsCount", accounts.length)}</span>
          </div>
          {accounts.length === 0 ? (
            <div className="empty-state" style={{ marginTop: 12 }}>
              <div className="empty-icon" aria-hidden="true">+</div>
              <h2>{t("providers.noAccounts")}</h2>
              <p>{t("providers.noAccountsBody")}</p>
            </div>
          ) : (
            <div className="account-list">
              {accounts.map(account => (
                <div className="account-row" key={`${account.name}-${account.meta}`}>
                  <div>
                    <div className="name">{account.name}</div>
                    <div className="meta">{account.meta}</div>
                  </div>
                  <div className="actions">
                    <StatusChip status={account.status} label={t(account.statusKey)} />
                    <button className="btn" type="button">{t("common.test")}</button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <aside className="card">
          <h2>{t("providers.providerSummary")}</h2>
          <div className="list">
            <div className="list-item"><span>{t("common.status")}</span><span>{t(provider.statusKey)}</span></div>
            <div className="list-item"><span>{t("providers.origin.label")}</span><OriginChips origins={provider.origins} /></div>
            <div className="list-item"><span>{t("providers.type")}</span><span>{provider.type}</span></div>
            <div className="list-item">
              <span>{t("providers.capabilities")}</span>
              <span className="caps" style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
                {provider.capabilities.map(capability => <span className="chip" key={capability}>{capability}</span>)}
              </span>
            </div>
          </div>
        </aside>
      </div>

      {open ? (
        <div className="modal-backdrop" role="presentation" onClick={event => { if (event.target === event.currentTarget) setOpen(false); }}>
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="account-modal-title">
            <div className="modal-head">
              <div>
                <h2 id="account-modal-title">{t("providers.addAccount")}</h2>
                <p className="muted">{t("providers.accountModalBody")}</p>
              </div>
              <button className="modal-close" type="button" aria-label={t("common.close")} onClick={() => setOpen(false)}>
                <X aria-hidden="true" size={16} />
              </button>
            </div>
            <form onSubmit={addAccount}>
              <div className="form-grid">
                <label className="field">
                  <span>{t("providers.accountName")}</span>
                  <input className="input" required value={name} onChange={event => setName(event.target.value)} placeholder={t("providers.accountNamePlaceholder")} />
                </label>
                <label className="field">
                  <span>{t("providers.accountCredential")}</span>
                  <input className="input" required type="password" value={credential} onChange={event => setCredential(event.target.value)} placeholder={t("providers.accountCredentialPlaceholder")} />
                </label>
              </div>
              {accountError ? <p className="small" style={{ color: "var(--danger)" }}>{accountError}</p> : null}
              <div className="modal-foot">
                <button className="btn" type="button" onClick={() => setOpen(false)}>{t("common.cancel")}</button>
                <button className="btn primary" type="submit">{t("providers.saveAccount")}</button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </>
  );
}
