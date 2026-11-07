import {
  CONFIGURED_PROVIDERS,
  PROVIDERS,
  type Provider,
  type ProviderAccount,
  type ProviderOrigin
} from "./providers";

interface AdminAccount {
  id: string;
  name: string;
  status: string;
  base_url?: string;
  priority?: number;
  weight?: number;
}

interface AdminProvider {
  id: string;
  name: string;
  kind: string;
  origin: string;
  enabled: boolean;
  accounts_count: number;
  models_count: number;
  accounts?: AdminAccount[];
}

function staticProvider(record: AdminProvider): Provider | undefined {
  return PROVIDERS[record.kind] ??
    Object.values(PROVIDERS).find(provider => provider.name.toLowerCase() === record.name.toLowerCase());
}

function mapAccount(account: AdminAccount): ProviderAccount {
  const warn = account.status === "cooldown";
  return {
    name: account.name,
    meta: account.base_url || account.id,
    status: warn ? "warn" : "ok",
    statusKey: warn ? "common.cooldown" : "common.healthy"
  };
}

function mapProvider(record: AdminProvider): Provider {
  const base = staticProvider(record);
  const origin = record.origin as ProviderOrigin;
  return {
    ...(base ?? {
      id: record.id,
      name: record.name,
      icon: record.name.slice(0, 2).toUpperCase(),
      type: record.kind,
      status: "ok" as const,
      statusKey: "common.healthy",
      capabilities: [],
      origins: [origin],
      configured: true,
      signupUrl: "",
      guide: "providers",
      accounts: []
    }),
    id: record.id,
    name: record.name,
    origins: [origin],
    configured: true,
    accountsCount: record.accounts_count,
    accounts: (record.accounts ?? []).map(mapAccount)
  };
}

async function adminRequest(path: string): Promise<Response | null> {
  const base = process.env.DONTSTOP_GATEWAY_URL;
  const key = process.env.DONTSTOP_ADMIN_API_KEY;
  if (!base || !key) return null;
  try {
    return await fetch(`${base.replace(/\/$/, "")}${path}`, {
      headers: { Authorization: `Bearer ${key}` },
      cache: "no-store",
      signal: AbortSignal.timeout(4000)
    });
  } catch {
    return null;
  }
}

export async function getConfiguredProviders(): Promise<Provider[]> {
  const response = await adminRequest("/admin/v1/providers");
  if (!response?.ok) return CONFIGURED_PROVIDERS;
  try {
    const body = await response.json() as { data?: AdminProvider[] };
    if (!Array.isArray(body.data) || body.data.length === 0) return CONFIGURED_PROVIDERS;
    return body.data.map(mapProvider);
  } catch {
    return CONFIGURED_PROVIDERS;
  }
}

export async function getProvider(id: string): Promise<Provider | null> {
  const response = await adminRequest(`/admin/v1/providers/${encodeURIComponent(id)}`);
  if (response?.ok) {
    try {
      return mapProvider(await response.json() as AdminProvider);
    } catch {
      // Fall back to the static catalog below.
    }
  }
  return PROVIDERS[id] ?? null;
}


export interface AdminAPIKey {
  id: string;
  name: string;
  prefix: string;
  scopes: string[];
  expires_at?: string;
  revoked_at?: string;
  created_at: string;
  last_used_at?: string;
  key?: string;
}

export async function getApiKeys(): Promise<AdminAPIKey[] | null> {
  const response = await adminRequest("/admin/v1/api-keys");
  if (!response?.ok) return null;
  try {
    const body = await response.json() as { data?: AdminAPIKey[] };
    return Array.isArray(body.data) ? body.data : null;
  } catch {
    return null;
  }
}


export interface CostReport {
  summary: { days: number; cost_usd: number; requests: number; tokens_in: number; tokens_out: number };
  by_provider: Array<{ provider_id: string; provider_name: string; cost_usd: number; requests: number }>;
  by_day: Array<{ day: string; cost_usd: number; requests: number }>;
  by_model: Array<{ model: string; cost_usd: number; requests: number; tokens: number }>;
}

export async function getCostReport(days = 30): Promise<CostReport | null> {
  const response = await adminRequest(`/admin/v1/costs/summary?days=${days}`);
  if (!response?.ok) return null;
  try {
    return await response.json() as CostReport;
  } catch {
    return null;
  }
}

export interface ObservabilitySummary {
  requests: number;
  errors: number;
  error_rate: number;
  p50_ms: number;
  p95_ms: number;
  p99_ms: number;
  cost_usd: number;
  tokens_in: number;
  tokens_out: number;
}

export async function getObservabilitySummary(days = 1): Promise<ObservabilitySummary | null> {
  const response = await adminRequest(`/admin/v1/observability/summary?days=${days}`);
  if (!response?.ok) return null;
  try {
    return await response.json() as ObservabilitySummary;
  } catch {
    return null;
  }
}


export interface MCPTool {
  id: string;
  name: string;
  scope: string;
  enabled: boolean;
}

export interface MCPStatus {
  enabled: boolean;
  transport: "stdio" | "sse" | "streamable-http";
  tools: MCPTool[];
}

export async function getMcpStatus(): Promise<MCPStatus | null> {
  const response = await adminRequest("/admin/v1/mcp/status");
  if (!response?.ok) return null;
  try {
    return await response.json() as MCPStatus;
  } catch {
    return null;
  }
}
