import { headers } from "next/headers";
import { getMcpActivity, getMcpStatus } from "@/lib/admin-api";
import publicUrl from "@/lib/public-url";
import { McpView } from "./mcp-view";

export default async function McpPage() {
  const [status, activity] = await Promise.all([getMcpStatus(), getMcpActivity()]);
  const resolved = publicUrl.resolvePublicEndpoint({ headers: await headers() });
  const streamableUrl = publicUrl.buildServiceUrl(resolved.base, resolved.basePath, "/mcp") ?? "";
  const sseUrl = publicUrl.buildServiceUrl(resolved.base, resolved.basePath, "/sse") ?? "";
  return <McpView initialStatus={status} initialActivity={activity} streamableUrl={streamableUrl} sseUrl={sseUrl} />;
}
