import { ProvidersView } from "./providers-view";

export default async function ProvidersPage({
  searchParams
}: {
  searchParams: Promise<{ empty?: string }>;
}) {
  const params = await searchParams;
  return <ProvidersView empty={params.empty === "1"} />;
}
