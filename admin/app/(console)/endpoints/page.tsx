import { getEndpoints } from "@/lib/admin-api";
import { EndpointsView } from "./endpoints-view";

export default async function EndpointsPage() {
  const endpoints = await getEndpoints();
  return <EndpointsView endpoints={endpoints} />;
}
