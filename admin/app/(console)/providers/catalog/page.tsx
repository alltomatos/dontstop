import { getConfiguredProviders, getProviderCatalogResult } from "@/lib/admin-api";
import { availableCatalogProviders } from "@/lib/catalog";
import { CatalogView } from "./catalog-view";

export const dynamic = "force-dynamic";

export default async function ProviderCatalogPage() {
  const [{ entries, error }, configured] = await Promise.all([
    getProviderCatalogResult(),
    getConfiguredProviders()
  ]);
  const providers = availableCatalogProviders(entries, configured);
  return <CatalogView providers={providers} error={error} />;
}
