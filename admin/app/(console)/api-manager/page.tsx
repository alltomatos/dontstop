import { getApiKeys } from "@/lib/admin-api";
import { ApiManagerView, MOCK_KEYS, type KeyRow } from "./api-manager-view";

function relative(value?: string): string {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";
  return date.toLocaleDateString();
}

export default async function ApiManagerPage() {
  const remote = await getApiKeys();
  const initialKeys: KeyRow[] = remote?.map(key => ({
    id: key.id,
    name: key.name,
    prefix: `${key.prefix}...`,
    scopes: key.scopes.join(", "),
    quota: "-",
    lastUse: relative(key.last_used_at),
    status: key.revoked_at ? "expiring" : "active"
  })) ?? MOCK_KEYS;
  return <ApiManagerView initialKeys={initialKeys} />;
}
