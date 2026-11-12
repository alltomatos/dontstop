import { getProxyEndpointsResult, getWebshareStatusResult } from "@/lib/admin-api";
import { ProxiesView } from "./proxies-view";

export default async function ProxiesPage() {
  const [{ endpoints, error }, webshare] = await Promise.all([
    getProxyEndpointsResult(),
    getWebshareStatusResult()
  ]);
  return <ProxiesView endpoints={endpoints} error={error} webshare={webshare.status} webshareError={webshare.error} />;
}
