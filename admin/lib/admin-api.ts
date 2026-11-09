import { getAdminKey } from "./admin-auth";
import {
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
  const origin: ProviderOrigin = record.origin === "custom" ? "custom" : "native";
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
  const key = await getAdminKey();
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

export async function getConfiguredProvidersResult(): Promise<{ providers: Provider[]; error: boolean }> {
  const response = await adminRequest("/admin/v1/providers");
  if (!response) return { providers: [], error: true };
  if (!response.ok) {
    console.warn("admin providers request failed", { status: response.status });
    return { providers: [], error: true };
  }
  try {
    const body = await response.json() as { data?: AdminProvider[] };
    if (!Array.isArray(body.data)) return { providers: [], error: true };
    return { providers: body.data.map(mapProvider), error: false };
  } catch {
    return { providers: [], error: true };
  }
}

export async function getConfiguredProviders(): Promise<Provider[]> {
  return (await getConfiguredProvidersResult()).providers;
}

export async function getProvider(id: string): Promise<Provider | null> {
  const response = await adminRequest(`/admin/v1/providers/${encodeURIComponent(id)}`);
  if (response?.ok) {
    try {
      return mapProvider(await response.json() as AdminProvider);
    } catch {
      return null;
    }
  }
  return null;
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


export interface CacheStats {
  hits: number;
  misses: number;
  requests: number;
  hit_rate: number;
  avg_latency_ms: number;
}

export interface CacheEvent {
  time: string;
  status: string;
  model: string;
  latency_ms: number;
  tokens: number;
}

export async function getCacheStats(days = 1): Promise<CacheStats | null> {
  const response = await adminRequest(`/admin/v1/cache/stats?days=${days}`);
  if (!response?.ok) return null;
  try {
    return await response.json() as CacheStats;
  } catch {
    return null;
  }
}

export async function getCacheEvents(limit = 20): Promise<CacheEvent[]> {
  const response = await adminRequest(`/admin/v1/cache/events?limit=${limit}`);
  if (!response?.ok) return [];
  try {
    const body = await response.json() as { data?: CacheEvent[] };
    return Array.isArray(body.data) ? body.data : [];
  } catch {
    return [];
  }
}


export interface DLPRule {
  id: string;
  name: string;
  detector: string;
  action: "allow" | "redact" | "block" | "audit";
  severity: "low" | "medium" | "high" | "critical";
  enabled: boolean;
  created_at: string;
}

export interface DLPEvent {
  id: string;
  request_id?: string;
  detector: string;
  action: string;
  severity: string;
  value_hash?: string;
  created_at: string;
}

export async function getDlpRules(): Promise<DLPRule[]> {
  const response = await adminRequest("/admin/v1/dlp/rules");
  if (!response?.ok) return [];
  try {
    const body = await response.json() as { data?: DLPRule[] };
    return Array.isArray(body.data) ? body.data : [];
  } catch {
    return [];
  }
}

export async function getDlpEvents(limit = 20): Promise<DLPEvent[]> {
  const response = await adminRequest(`/admin/v1/dlp/events?limit=${limit}`);
  if (!response?.ok) return [];
  try {
    const body = await response.json() as { data?: DLPEvent[] };
    return Array.isArray(body.data) ? body.data : [];
  } catch {
    return [];
  }
}


export interface EndpointInfo {
  id: string;
  name: string;
  path: string;
  category: string;
  models: number;
  enabled: boolean;
}

export async function getEndpoints(): Promise<EndpointInfo[]> {
  const response = await adminRequest("/admin/v1/endpoints");
  if (!response?.ok) return [];
  try {
    const body = await response.json() as { data?: EndpointInfo[] };
    return Array.isArray(body.data) ? body.data : [];
  } catch {
    return [];
  }
}

export async function getSetupStatus(): Promise<boolean> {
  const base = process.env.DONTSTOP_GATEWAY_URL;
  if (!base) return false;
  try {
    const response = await fetch(`${base.replace(/\/$/, "")}/admin/v1/setup/status`, {
      cache: "no-store",
      signal: AbortSignal.timeout(4000)
    });
    if (!response.ok) return false;
    const body = await response.json() as { needs_setup?: boolean };
    return body.needs_setup === true;
  } catch {
    return false;
  }
}
