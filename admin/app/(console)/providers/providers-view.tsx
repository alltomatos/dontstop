"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { Plus } from "lucide-react";
import { Breadcrumbs, EmptyState, PageHeader } from "@/components/console/page";
import { ProviderCard } from "@/components/console/provider-card";
import { useI18n } from "@/components/console/i18n-provider";
import { ORIGIN_ORDER, type Provider, type ProviderOrigin } from "@/lib/providers";

type OriginFilter = ProviderOrigin | "all";

export function ProvidersView({ providers, error = false, empty = false }: { providers: Provider[]; error?: boolean; empty?: boolean }) {
  const { t, tp } = useI18n();
  const [query, setQuery] = useState("");
  const [origin, setOrigin] = useState<OriginFilter>("all");
  const [view, setView] = useState<"grid" | "list">("grid");

  const visible = useMemo(() => {
    if (empty) return [];
    const normalized = query.trim().toLowerCase();
    return providers.filter(provider => {
      const matchesQuery = !normalized || provider.name.toLowerCase().includes(normalized);
      const matchesOrigin = origin === "all" || provider.origins.includes(origin);
      return matchesQuery && matchesOrigin;
    });
  }, [empty, origin, providers, query]);

  const isEmpty = !empty && providers.length === 0;
  const noResults = !empty && providers.length > 0 && visible.length === 0;

  return (
    <>
      <Breadcrumbs items={[{ label: t("nav.providers"), href: "/providers" }, { label: t("providers.configuredTitle") }]} />
      <PageHeader
        title={t("providers.configuredTitle")}
        subtitle={t("providers.configuredSubtitle")}
        actions={(
          <>
            <button className="btn" type="button">{t("common.testAll")}</button>
            <button className="btn" type="button">{t("common.import")}</button>
            <Link className="btn primary" href="/providers/catalog">
              <Plus aria-hidden="true" size={16} />
              {t("common.addProvider")}
            </Link>
          </>
        )}
      />

      {error ? (
        <p className="small" role="status" style={{ color: "var(--danger)", marginBottom: 12 }}>
          {t("common.gatewayError")}
        </p>
      ) : null}

      <div className="card" style={{ marginBottom: 16 }}>
        <div className="toolbar">
          <input
            className="input"
            value={query}
            onChange={event => setQuery(event.target.value)}
            placeholder={t("providers.searchConfigured")}
            aria-label={t("providers.searchConfigured")}
          />
          <span className="muted">{tp("providers.configuredCount", empty ? 0 : visible.length)}</span>
          <div className="actions">
            <button className={`btn ${view === "grid" ? "primary" : ""}`} type="button" aria-pressed={view === "grid"} onClick={() => setView("grid")}>
              {t("providers.grid")}
            </button>
            <button className={`btn ${view === "list" ? "primary" : ""}`} type="button" aria-pressed={view === "list"} onClick={() => setView("list")}>
              {t("providers.list")}
            </button>
          </div>
        </div>
        <div className="filter-chips" style={{ marginTop: 12 }}>
          <button className={`filter-chip ${origin === "all" ? "active" : ""}`} type="button" onClick={() => setOrigin("all")}>
            {t("providers.origin.all")}
          </button>
          {ORIGIN_ORDER.map(item => (
            <button className={`filter-chip ${origin === item ? "active" : ""}`} type="button" key={item} onClick={() => setOrigin(item)}>
              {t(`providers.origin.${item}`)}
            </button>
          ))}
        </div>
      </div>

      {empty || isEmpty || noResults ? (
        <EmptyState
          title={noResults ? t("providers.noResultsTitle") : t("providers.emptyTitle")}
          body={noResults ? t("providers.noResultsBody") : t("providers.emptyBody")}
          action={<Link className="btn primary" href="/providers/catalog">{t("common.addProvider")}</Link>}
        />
      ) : (
        <div className={`provider-grid ${view === "list" ? "list" : ""}`}>
          {visible.map(provider => <ProviderCard key={provider.id} provider={provider} />)}
        </div>
      )}
    </>
  );
}
