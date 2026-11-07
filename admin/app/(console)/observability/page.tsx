import { getObservabilitySummary } from "@/lib/admin-api";
import { ObservabilityView } from "./observability-view";

export default async function ObservabilityPage() {
  const summary = await getObservabilitySummary(1);
  return <ObservabilityView summary={summary} />;
}
