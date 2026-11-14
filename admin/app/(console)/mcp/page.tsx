import { getMcpActivity, getMcpStatus } from "@/lib/admin-api";
import { McpView } from "./mcp-view";

export default async function McpPage() {
  const [status, activity] = await Promise.all([getMcpStatus(), getMcpActivity()]);
  return <McpView initialStatus={status} initialActivity={activity} />;
}
