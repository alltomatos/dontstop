"use client";

import Link from "next/link";
import { ArrowRight } from "lucide-react";
import type { Provider } from "@/lib/providers";
import { useI18n } from "./i18n-provider";
import { OriginChips, StatusChip } from "./origin-chips";

export function ProviderCard({ provider }: { provider: Provider }) {
  const { t, tp } = useI18n();
  return (
    <Link className="provider-card provider-card-link" href={`/providers/${provider.id}`}>
      <div className="top">
        <span className="provider-icon">{provider.icon}</span>
        <div>
          <div className="name">{provider.name}</div>
          <div className="meta">{provider.type}</div>
        </div>
        <StatusChip status={provider.status} label={t(provider.statusKey)} />
      </div>
      <OriginChips origins={provider.origins} />
      <div className="caps">
        {provider.capabilities.map(capability => (
          <span className="chip" key={capability}>{capability}</span>
        ))}
      </div>
      <div className="foot">
        <span className="meta">{tp("providers.connected", provider.accountsCount ?? provider.accounts.length)}</span>
        <span className="detail-link">
          {t("providers.viewDetails")}
          <ArrowRight aria-hidden="true" size={16} />
        </span>
      </div>
    </Link>
  );
}
