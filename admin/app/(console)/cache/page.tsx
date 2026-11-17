import { getCacheReport } from "@/lib/admin-api";
import { CacheView } from "./cache-view";

export default async function CachePage() {
  const report = await getCacheReport(7);
  return <CacheView report={report} />;
}
