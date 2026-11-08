import { getCacheStats, getCostReport, getObservabilitySummary } from "@/lib/admin-api";
import { getConfiguredProviders } from "@/lib/admin-api";
import { DashboardView } from "./dashboard-view";

export default async function DashboardPage() {
  const [providers, cost, observability, cache] = await Promise.all([
    getConfiguredProviders(),
    getCostReport(1),
    getObservabilitySummary(1),
    getCacheStats(1)
  ]);
  return <DashboardView providers={providers} cost={cost} observability={observability} cache={cache} />;
}
