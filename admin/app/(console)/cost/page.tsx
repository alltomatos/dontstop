import { getCostReport } from "@/lib/admin-api";
import { CostView } from "./cost-view";

export default async function CostPage() {
  const report = await getCostReport(30);
  return <CostView report={report} />;
}
