"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMemo, useState, useTransition } from "react";
import { LoaderCircle, Network, Plus, RefreshCw, X } from "lucide-react";
import { Breadcrumbs, EmptyState, PageHeader } from "@/components/console/page";
import { useI18n } from "@/components/console/i18n-provider";
import type { ProxyEndpoint, ProxyHealth, WebshareStatus } from "@/lib/proxies";
import { WebsharePanel } from "./webshare-panel";

type HealthFilter = ProxyHealth | "all";

const HEALTH_KEYS: Record<ProxyHealth, string> = {
  healthy: "providers.proxyHealthHealthy",
  quarantined: "providers.proxyHealthQuarantined",
  unhealthy: "providers.proxyHealthUnhealthy"
};

const MODE_KEYS = {
  off: "providers.proxyModeOff",
  preferred: "providers.proxyModePreferred",
  required: "providers.proxyModeRequired"
} as const;

function healthClass(health: ProxyHealth): string {
  if (health === "healthy") return "ok";
  if (health === "quarantined") return "warn";
  return "";
}

export function ProxiesView({
  endpoints,
  error = false,
  webshare,
  webshareError = false
}: {
  endpoints: ProxyEndpoint[];
  error?: boolean;
  webshare: WebshareStatus;
  webshareError?: boolean;
}) {
  const { t, tp } = useI18n();
  const router = useRouter();
  const [pending, startTransition] = useTransition();
  const [query, setQuery] = useState("");
  const [health, setHealth] = useState<HealthFilter>("all");
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [createError, setCreateError] = useState("");
  const [form, setForm] = useState({
    label: "",
    address: "",
    port: "",
    country: "",
    city: "",
    username: "",
    password: ""
  });

  const visible = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    return endpoints.filter(endpoint => {
      const haystack = [endpoint.address, endpoint.label, endpoint.cityName, endpoint.countryCode].join(" ").toLowerCase();
      const matchesQuery = !normalized || haystack.includes(normalized);
      const matchesHealth = health === "all" || endpoint.health === health;
      return matchesQuery && matchesHealth;
    });
  }, [endpoints, health, query]);

  const healthy = endpoints.filter(endpoint => endpoint.health === "healthy").length;
  const quarantined = endpoints.filter(endpoint => endpoint.health === "quarantined").length;
  const assigned = endpoints.reduce((total, endpoint) => total + endpoint.assignedCount, 0);

  function refresh() {
    startTransition(() => router.refresh());
  }

  async function createProxy(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;
    const port = Number(form.port);
    if (!form.address.trim() || !Number.isInteger(port) || port < 1 || port > 65535) {
      setCreateError(t("providers.proxyCreateFailed"));
      return;
    }
    setBusy(true);
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
      if (!response.ok) {
        setCreateError(t("providers.proxyCreateFailed"));
        return;
      }
      setOpen(false);
      setForm({ label: "", address: "", port: "", country: "", city: "", username: "", password: "" });
      refresh();
    } catch {
      setCreateError(t("providers.proxyCreateFailed"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <Breadcrumbs items={[{ label: t("nav.proxies") }]} />
      <PageHeader
        title={t("proxies.title")}
        subtitle={t("proxies.subtitle")}
        actions={(
          <>
            <button className="btn" type="button" onClick={refresh} disabled={pending}>
              {pending ? <LoaderCircle aria-hidden="true" size={16} className="spin" /> : <RefreshCw aria-hidden="true" size={16} />}
              {t("proxies.refresh")}
            </button>
            <button className="btn primary" type="button" onClick={() => setOpen(true)}>
              <Plus aria-hidden="true" size={16} />
              {t("proxies.create")}
            </button>
          </>
        )}
      />

      <WebsharePanel status={webshare} onRefresh={refresh} />
      {webshareError ? (
        <p className="small" role="status" style={{ color: "var(--danger)", marginBottom: 12 }}>{t("webshare.failed")}</p>
      ) : null}

      {error ? (
        <p className="small" role="status" style={{ color: "var(--danger)", marginBottom: 12 }}>
          {t("proxies.error")}
        </p>
      ) : null}

      <div className="card" style={{ marginBottom: 16 }}>
        <div className="toolbar">
          <input
            className="input"
            value={query}
            onChange={event => setQuery(event.target.value)}
            placeholder={t("proxies.search")}
            aria-label={t("proxies.search")}
          />
          <span className="muted">{tp("proxies.count", endpoints.length)}</span>
          <div className="actions">
            {(["all", "healthy", "quarantined", "unhealthy"] as HealthFilter[]).map(value => (
              <button
                className={`btn ${health === value ? "primary" : ""}`}
                type="button"
                key={value}
                aria-pressed={health === value}
                onClick={() => setHealth(value)}
              >
                {value === "all" ? t("proxies.all") : t(HEALTH_KEYS[value])}
              </button>
            ))}
          </div>
        </div>
        <div className="caps" style={{ display: "flex", gap: 8, flexWrap: "wrap", marginTop: 10 }}>
          <span className="chip ok">{tp("proxies.healthyCount", healthy)}</span>
          <span className="chip warn">{tp("proxies.quarantinedCount", quarantined)}</span>
          <span className="chip">{tp("proxies.assignedCount", assigned)}</span>
        </div>
      </div>

      {endpoints.length === 0 ? (
        <div className="card">
          <EmptyState
            title={t("proxies.emptyTitle")}
            body={t("proxies.emptyBody")}
            action={<button className="btn primary" type="button" onClick={() => setOpen(true)}><Plus aria-hidden="true" size={16} />{t("proxies.create")}</button>}
          />
        </div>
      ) : visible.length === 0 ? (
        <div className="card">
          <p className="muted" role="status">{t("proxies.noResults")}</p>
        </div>
      ) : (
        <div className="card" style={{ padding: 0, overflowX: "auto" }}>
          <table className="table">
            <thead>
              <tr>
                <th>{t("proxies.endpoint")}</th>
                <th>{t("proxies.region")}</th>
                <th>{t("proxies.health")}</th>
                <th>{t("proxies.accounts")}</th>
                <th>{t("proxies.source")}</th>
              </tr>
            </thead>
            <tbody>
              {visible.map(endpoint => (
                <tr key={endpoint.id}>
                  <td>
                    <div className="proxy-endpoint">{endpoint.address}:{endpoint.port}</div>
                    {endpoint.label ? <div className="meta">{endpoint.label}</div> : null}
                  </td>
                  <td>{[endpoint.cityName, endpoint.countryCode].filter(Boolean).join(", ") || "—"}</td>
                  <td>
                    <span className={`chip ${healthClass(endpoint.health)}`}>{t(HEALTH_KEYS[endpoint.health])}</span>
                  </td>
                  <td>
                    {endpoint.accounts.length === 0 ? (
                      <span className="muted small">{t("proxies.noAccounts")}</span>
                    ) : (
                      <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
                        {endpoint.accounts.map(account => (
                          <Link
                            className="chip"
                            key={account.id}
                            href={`/providers/${encodeURIComponent(account.providerId)}`}
                            title={t(MODE_KEYS[account.proxyMode])}
                          >
                            {account.providerName} · {account.name}
                          </Link>
                        ))}
                      </div>
                    )}
                  </td>
                  <td>
                    <span className="chip">{endpoint.source === "webshare" ? t("proxies.webshare") : t("proxies.manual")}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {open ? (
        <div className="modal-backdrop" role="presentation" onClick={event => { if (event.target === event.currentTarget) setOpen(false); }}>
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="proxy-create-title">
            <div className="modal-head">
              <div>
                <h2 id="proxy-create-title">{t("proxies.createTitle")}</h2>
                <p className="muted">{t("proxies.createBody")}</p>
              </div>
              <button className="modal-close" type="button" aria-label={t("common.close")} onClick={() => setOpen(false)}>
                <X aria-hidden="true" size={16} />
              </button>
            </div>
            <form onSubmit={createProxy}>
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
                <button className="btn" type="button" onClick={() => setOpen(false)}>{t("common.cancel")}</button>
                <button className="btn primary" type="submit" disabled={busy}>
                  {busy ? <LoaderCircle aria-hidden="true" size={16} className="spin" /> : <Network aria-hidden="true" size={16} />}
                  {busy ? t("proxies.saving") : t("proxies.save")}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </>
  );
}
