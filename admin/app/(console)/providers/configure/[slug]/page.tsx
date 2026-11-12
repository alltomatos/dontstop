import { notFound, redirect } from "next/navigation";
import { getCatalogEntry, getConfiguredProviders } from "@/lib/admin-api";
import { ConfigureProvider } from "./configure-form";

export const dynamic = "force-dynamic";

export default async function ConfigureProviderPage({
  params
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const [entry, configured] = await Promise.all([getCatalogEntry(slug), getConfiguredProviders()]);
  if (!entry) notFound();
  const existing = configured.find(
    provider => (provider.catalogSlug ?? "").toLowerCase() === entry.slug.toLowerCase()
  );
  if (existing) redirect(`/providers/${existing.id}`);
  return <ConfigureProvider entry={entry} />;
}
