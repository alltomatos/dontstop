"use client";

import type { ProviderOrigin } from "@/lib/providers";
import { useI18n } from "./i18n-provider";

export function OriginChips({ origins }: { origins: ProviderOrigin[] }) {
  const { t } = useI18n();
  return (
    <div className="origin-tags">
      {origins.map(origin => (
        <span className="chip origin" key={origin}>{t(`providers.origin.${origin}`)}</span>
      ))}
    </div>
  );
}

export function StatusChip({ status, label }: { status: "ok" | "warn"; label: string }) {
  return (
    <span className={`chip ${status}`}>
      <span className="dot" />
      {label}
    </span>
  );
}
