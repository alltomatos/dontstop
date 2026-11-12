"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { ArrowLeft } from "lucide-react";
import { Breadcrumbs, EmptyState, PageHeader } from "@/components/console/page";
import { OriginChips } from "@/components/console/origin-chips";
import { useI18n } from "@/components/console/i18n-provider";
import { configurePath } from "@/src/catalog";
import { ORIGIN_ORDER, type Provider, type ProviderOrigin } from "@/lib/providers";

type OriginFilter = ProviderOrigin | "all";
type TypeFilter = "all" | "oauth" | "api" | "compatible" | "local";

function matchesType(type: string, filter: TypeFilter): boolean {
  if (filter === "all") return true;
  if (filter === "oauth") return type.includes("OAuth");
  if (filter === "api") return type.includes("API key") || type.includes("Anthropic");
  if (filter === "compatible") return type.includes("Compatível");
  return type.includes("Local");
}

export function CatalogView({ providers, error = false }: { providers: Provider[]; error?: boolean }) {
  const { t, tp } = useI18n();
  const [query, setQuery] = useState("");
  const [origin, setOrigin] = useState<OriginFilter>("all");
  const [type, setType] = useState<TypeFilter>("all");

  const visible = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    return providers.filter(provider => {
      const matchesQuery = !normalized || provider.name.toLowerCase().includes(normalized);
      const matchesOrigin = origin === "all" || provider.origins.includes(origin);
      return matchesQuery && matchesOrigin && matchesType(provider.type, type);
    });
  }, [origin, providers, query, type]);

  return (
    <>
      <Breadcrumbs items={[{ label: t("nav.providers"), href: "/providers" }, { label: t("providers.catalogTitle") }]} />
      <PageHeader
        title={t("providers.catalogTitle")}
        subtitle={t("providers.catalogSubtitle")}
        actions={<Link className="btn" href="/providers"><ArrowLeft aria-hidden="true" size={16} />{t("providers.backToProviders")}</Link>}
      />

      {error ? (
        <p className="small" role="status" style={{ color: "var(--danger)", marginBottom: 12 }}>
          {t("common.gatewayError")}
        </p>
      ) : null}

      <div className="card">
        <div className="toolbar">
          <input
            className="input"
            value={query}
            onChange={event => setQuery(event.target.value)}
            placeholder={t("providers.catalogSearch")}
            aria-label={t("providers.catalogSearch")}
          />
          <span className="muted">{tp("providers.catalogCount", visible.length)}</span>
        </div>
        <div className="filter-chips" style={{ marginTop: 12 }}>
          {([
            ["all", "providers.catalogAll"],
            ["oauth", "providers.oauth"],
            ["api", "providers.apiKeyFilter"],
            ["compatible", "providers.compatible"],
            ["local", "providers.local"]
          ] as Array<[TypeFilter, string]>).map(([value, key]) => (
            <button className={`filter-chip ${type === value ? "active" : ""}`} type="button" key={value} onClick={() => setType(value)}>
              {t(key)}
            </button>
          ))}
        </div>
        <div className="filter-chips" style={{ marginTop: 8 }}>
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

      {visible.length === 0 ? (
        <EmptyState
          title={t("providers.catalogEmptyTitle")}
          body={error ? t("providers.catalogUnavailable") : t("providers.catalogEmptyBody")}
          action={<Link className="btn" href="/providers">{t("providers.backToProviders")}</Link>}
        />
      ) : null}

      <div className="provider-grid" style={{ marginTop: 16 }}>
        {visible.map(provider => (
          <article className="provider-card" key={provider.id}>
            <div className="top">
              <span className="provider-icon">{provider.icon}</span>
              <div>
                <div className="name">{provider.name}</div>
                <div className="meta">{provider.type}</div>
              </div>
            </div>
            <OriginChips origins={provider.origins} />
            <div className="caps">
              {provider.capabilities.map(capability => <span className="chip" key={capability}>{capability}</span>)}
            </div>
            <div className="foot">
              <span className="meta">{t("providers.available")}</span>
              <Link className="btn" href={configurePath(provider.id)}>{t("providers.configure")}</Link>
            </div>
          </article>
        ))}
      </div>
    </>
  );
}
