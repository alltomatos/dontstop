import { getCacheStats, getCostReport, getObservabilitySummary } from "@/lib/admin-api";
import { getConfiguredProvidersResult } from "@/lib/admin-api";
import { DashboardView } from "./dashboard-view";

export default async function DashboardPage() {
  const [providersResult, cost, observability, cache] = await Promise.all([
    getConfiguredProvidersResult(),
    getCostReport(1),
    getObservabilitySummary(1),
    getCacheStats(1)
  ]);
  return (
    <DashboardView
      providers={providersResult.providers}
      gatewayError={providersResult.error || cost === null || observability === null}
      cost={cost}
      observability={observability}
      cache={cache}
    />
  );
}
