import { headers } from "next/headers";
import { getEndpoints } from "@/lib/admin-api";
import publicUrl from "@/lib/public-url";
import { EndpointsView } from "./endpoints-view";

export default async function EndpointsPage() {
  const endpoints = await getEndpoints();
  const resolved = publicUrl.resolvePublicEndpoint({ headers: await headers() });
  return (
    <EndpointsView
      endpoints={endpoints}
      endpointUrl={resolved.url ?? ""}
      localEndpointUrl={resolved.localUrl ?? ""}
      endpointSource={resolved.source}
    />
  );
}
