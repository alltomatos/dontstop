import { capabilitiesOf, filterAvailableCatalog } from "@/src/catalog";
import type { CatalogEntry } from "./admin-api";
import { PROVIDERS, SIGNUP_URLS, type Provider } from "./providers";

export const PROVIDER_SETUP_GUIDE = "provider-setup";

/** Maps a versioned catalog entry to the console provider view model. */
export function catalogEntryToProvider(entry: CatalogEntry): Provider {
  const base = PROVIDERS[entry.slug];
  const name = entry.display_name || base?.name || entry.slug;
  const capabilities = capabilitiesOf(entry);
  return {
    id: entry.slug,
    slug: entry.slug,
    catalogSlug: entry.slug,
    name,
    icon: base?.icon ?? name.slice(0, 2).toUpperCase(),
    type: `API key · ${name}`,
    status: "ok",
    statusKey: "providers.available",
    capabilities: capabilities.length > 0 ? capabilities : base?.capabilities ?? [],
    origins: ["native"],
    configured: false,
    modelsCount: entry.models?.length ?? 0,
    signupUrl: SIGNUP_URLS[entry.slug] ?? base?.signupUrl ?? "",
    guide: PROVIDER_SETUP_GUIDE,
    accounts: []
  };
}

/** Catalog entries that are not yet configured for the current tenant. */
export function availableCatalogProviders(entries: CatalogEntry[], configured: Provider[]): Provider[] {
  return filterAvailableCatalog(entries, configured).map(catalogEntryToProvider);
}
