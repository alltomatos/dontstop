import { getCacheEvents, getCacheStats } from "@/lib/admin-api";
import { CacheView } from "./cache-view";

export default async function CachePage() {
  const [stats, events] = await Promise.all([getCacheStats(1), getCacheEvents(20)]);
  return <CacheView stats={stats} events={events} />;
}
