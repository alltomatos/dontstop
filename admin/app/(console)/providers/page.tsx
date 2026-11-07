import { getConfiguredProviders } from "@/lib/admin-api";
import { ProvidersView } from "./providers-view";

export default async function ProvidersPage({
  searchParams
}: {
  searchParams: Promise<{ empty?: string }>;
}) {
  const params = await searchParams;
  const providers = await getConfiguredProviders();
  return <ProvidersView providers={providers} empty={params.empty === "1"} />;
}
