import { getConfiguredProvidersResult } from "@/lib/admin-api";
import { ProvidersView } from "./providers-view";

export default async function ProvidersPage({
  searchParams
}: {
  searchParams: Promise<{ empty?: string }>;
}) {
  const params = await searchParams;
  const { providers, error } = await getConfiguredProvidersResult();
  return <ProvidersView providers={providers} error={error} empty={params.empty === "1"} />;
}
