import { getMcpStatus } from "@/lib/admin-api";
import { McpView } from "./mcp-view";

export default async function McpPage() {
  const status = await getMcpStatus();
  return <McpView initialStatus={status} />;
}
