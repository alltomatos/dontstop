import { notFound } from "next/navigation";
import { getCatalogEntry, getConfiguredModels, getConfiguredProviders, getLatestModelTestRun, getProvider } from "@/lib/admin-api";
import { catalogEntryToProvider } from "@/lib/catalog";
import { ProviderDetail } from "./provider-detail";

export const dynamic = "force-dynamic";

export default async function ProviderDetailPage({
  params
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const provider = await getProvider(id);
  if (provider) {
    const [models, latestRun] = await Promise.all([
      getConfiguredModels(provider.id),
      getLatestModelTestRun(provider.id)
    ]);
    return <ProviderDetail provider={provider} models={models} latestRun={latestRun} />;
  }
  const entry = await getCatalogEntry(id);
  if (entry) {
    const configured = await getConfiguredProviders();
    const existing = configured.find(
      provider => (provider.catalogSlug ?? "").toLowerCase() === entry.slug.toLowerCase()
    );
    if (existing) {
      const [models, latestRun] = await Promise.all([
        getConfiguredModels(existing.id),
        getLatestModelTestRun(existing.id)
      ]);
      return <ProviderDetail provider={existing} models={models} latestRun={latestRun} />;
    }
    return <ProviderDetail provider={catalogEntryToProvider(entry)} models={[]} />;
  }
  notFound();
}
