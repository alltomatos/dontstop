import { getAdminKey } from "./admin-auth";
import type { ProxyEndpoint, WebshareStatus } from "./proxies";
import {
  PROVIDERS,
  SIGNUP_URLS,
  type Provider,
  type ProviderAccount,
  type ProviderOrigin
} from "./providers";

interface AdminAccountProxy {
  id: string;
  label: string;
  address: string;
  port: number;
  country_code: string;
  city_name: string;
  health: string;
}

interface AdminAccount {
  id: string;
  name: string;
  status: string;
  base_url?: string;
  priority?: number;
  weight?: number;
  proxy_mode?: string;
  proxy?: AdminAccountProxy | null;
  models_available?: number;
  models_usable?: number;
  capabilities?: string[];
  health?: string;
  active?: boolean;
}

interface AdminProvider {
  id: string;
  name: string;
  kind: string;
  origin: string;
  catalog_slug?: string;
  enabled: boolean;
  accounts_count: number;
  usable_accounts?: number;
  models_count: number;
  models_usable?: number;
  accounts?: AdminAccount[];
}

function staticProvider(record: AdminProvider): Provider | undefined {
  if (record.catalog_slug && PROVIDERS[record.catalog_slug]) return PROVIDERS[record.catalog_slug];
  const byName = Object.values(PROVIDERS).find(
    provider => provider.name.toLowerCase() === record.name.toLowerCase()
  );
  if (byName) return byName;
  // Legacy rows have no catalog slug; only native rows may inherit adapter metadata.
  return record.origin === "custom" ? undefined : PROVIDERS[record.kind];
}

function mapAccount(account: AdminAccount): ProviderAccount {
  const warn = account.status === "cooldown";
  return {
    id: account.id,
    name: account.name,
    meta: account.base_url || account.id,
    status: warn ? "warn" : "ok",
    statusKey: warn ? "common.cooldown" : "common.healthy",
    proxyMode: (account.proxy_mode as ProviderAccount["proxyMode"]) ?? "off",
    modelsAvailable: account.models_available ?? 0,
    modelsUsable: account.models_usable ?? 0,
    capabilities: account.capabilities ?? [],
    health: (account.health as ProviderAccount["health"]) ?? (account.status === "cooldown" ? "cooldown" : account.status === "disabled" ? "disabled" : "active"),
    active: account.active ?? account.status === "active",
    proxy: account.proxy
      ? {
          id: account.proxy.id,
          label: account.proxy.label,
          address: account.proxy.address,
          port: account.proxy.port,
          countryCode: account.proxy.country_code,
          cityName: account.proxy.city_name,
          health: account.proxy.health as NonNullable<ProviderAccount["proxy"]>["health"]
        }
      : null
  };
}

function mapProvider(record: AdminProvider): Provider {
  const base = staticProvider(record);
  const origin: ProviderOrigin = record.origin === "custom" ? "custom" : "native";
  const signupUrl = (record.catalog_slug && SIGNUP_URLS[record.catalog_slug]) || base?.signupUrl || "";
  return {
    ...(base ?? {
      id: record.id,
      name: record.name,
      icon: record.name.slice(0, 2).toUpperCase(),
      type: `API key · ${record.name}`,
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
    slug: record.catalog_slug || base?.id,
    catalogSlug: record.catalog_slug,
    signupUrl,
    origins: [origin],
    configured: true,
    accountsCount: record.accounts_count,
    usableAccounts: record.usable_accounts ?? 0,
    modelsCount: record.models_count,
    modelsUsable: record.models_usable ?? record.models_count,
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
    // A provider without accounts is not usable and must not appear as configured.
    return { providers: body.data.filter(provider => provider.accounts_count > 0).map(mapProvider), error: false };
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


export interface CatalogModel {
  model_id: string;
  display_name?: string;
  context_window?: number;
  max_output_tokens?: number;
  supports_text?: boolean;
  supports_stream?: boolean;
  supports_tools?: boolean;
  supports_vision?: boolean;
  supports_reasoning?: boolean;
  supports_audio?: boolean;
  supports_video?: boolean;
  supports_embeddings?: boolean;
  input_price_per_million?: number;
  output_price_per_million?: number;
  pricing_source?: string;
}

export interface CatalogEntry {
  slug: string;
  display_name: string;
  protocol: string;
  api_version?: string;
  default_base_url: string;
  models_path?: string;
  auth_type: string;
  auth_header: string;
  auth_scheme: string;
  headers?: Record<string, string>;
  capabilities?: Record<string, boolean>;
  network?: Record<string, unknown>;
  quirks?: string[];
  source_ref?: string;
  catalog_version?: string;
  models?: CatalogModel[];
}

export async function getProviderCatalogResult(): Promise<{ entries: CatalogEntry[]; error: boolean }> {
  const response = await adminRequest("/admin/v1/provider-catalog");
  if (!response) return { entries: [], error: true };
  if (!response.ok) {
    console.warn("admin provider catalog request failed", { status: response.status });
    return { entries: [], error: true };
  }
  try {
    const body = await response.json() as { data?: CatalogEntry[] };
    if (!Array.isArray(body.data)) return { entries: [], error: true };
    return { entries: body.data, error: false };
  } catch {
    return { entries: [], error: true };
  }
}

export async function getProviderCatalog(): Promise<CatalogEntry[]> {
  return (await getProviderCatalogResult()).entries;
}

export async function getCatalogEntry(slug: string): Promise<CatalogEntry | null> {
  if (!slug) return null;
  const normalized = slug.trim().toLowerCase();
  const entries = await getProviderCatalog();
  return entries.find(entry => entry.slug.toLowerCase() === normalized) ?? null;
}

export interface ConfiguredModel {
  model_id: string;
  display_name: string;
  active: boolean;
  capabilities?: Record<string, unknown>;
}

export interface ModelTestRunItem {
  model_id: string;
  status: "pending" | "running" | "ok" | "failed";
  latency_ms: number;
  error?: string;
}

export interface ModelTestRun {
  id: string;
  provider_id: string;
  status: "queued" | "running" | "completed" | "failed";
  total: number;
  completed: number;
  ok_count: number;
  failed_count: number;
  error?: string;
  created_at: string;
  started_at?: string;
  finished_at?: string;
  items?: ModelTestRunItem[];
}

export async function getLatestModelTestRun(providerId: string): Promise<ModelTestRun | null> {
  const response = await adminRequest(`/admin/v1/providers/${encodeURIComponent(providerId)}/models/test-runs/latest`);
  if (!response?.ok) return null;
  try {
    return await response.json() as ModelTestRun;
  } catch {
    return null;
  }
}

export async function getConfiguredModels(providerId: string): Promise<ConfiguredModel[]> {
  const response = await adminRequest(`/admin/v1/providers/${encodeURIComponent(providerId)}/models/configured`);
  if (!response?.ok) return [];
  try {
    const body = await response.json() as { data?: ConfiguredModel[] };
    return Array.isArray(body.data) ? body.data : [];
  } catch {
    return [];
  }
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

interface AdminProxyAccount {
  id: string;
  name: string;
  provider_id: string;
  provider_name: string;
  proxy_mode: string;
}

interface AdminProxyEndpoint {
  id: string;
  label: string;
  address: string;
  port: number;
  country_code: string;
  city_name: string;
  source: string;
  valid: boolean;
  health: string;
  quarantined_until?: string;
  last_verified_at?: string;
  assigned_count: number;
  accounts?: AdminProxyAccount[];
}

function mapProxyEndpoint(record: AdminProxyEndpoint): ProxyEndpoint {
  return {
    id: record.id,
    label: record.label,
    address: record.address,
    port: record.port,
    countryCode: record.country_code,
    cityName: record.city_name,
    source: record.source === "webshare" ? "webshare" : "manual",
    valid: record.valid,
    health: record.health as ProxyEndpoint["health"],
    quarantinedUntil: record.quarantined_until,
    lastVerifiedAt: record.last_verified_at,
    assignedCount: record.assigned_count,
    accounts: (record.accounts ?? []).map(account => ({
      id: account.id,
      name: account.name,
      providerId: account.provider_id,
      providerName: account.provider_name,
      proxyMode: account.proxy_mode as ProxyEndpoint["accounts"][number]["proxyMode"]
    }))
  };
}

export async function getProxyEndpointsResult(): Promise<{ endpoints: ProxyEndpoint[]; error: boolean }> {
  const response = await adminRequest("/admin/v1/proxies");
  if (!response) return { endpoints: [], error: true };
  if (!response.ok) {
    console.warn("admin proxies request failed", { status: response.status });
    return { endpoints: [], error: true };
  }
  try {
    const body = await response.json() as { data?: AdminProxyEndpoint[] };
    if (!Array.isArray(body.data)) return { endpoints: [], error: true };
    return { endpoints: body.data.map(mapProxyEndpoint), error: false };
  } catch {
    return { endpoints: [], error: true };
  }
}

interface AdminWebshareStatus {
  configured: boolean;
  plan_id: string;
  enabled: boolean;
  key_version: number;
  last_sync_at?: string;
  last_sync_status: string;
  last_sync_error: string;
  last_sync_total: number;
  last_sync_imported: number;
  proxy_count: number;
  healthy_count: number;
}

export async function getWebshareStatusResult(): Promise<{ status: WebshareStatus; error: boolean }> {
  const empty: WebshareStatus = {
    configured: false,
    planId: "",
    enabled: false,
    keyVersion: 0,
    lastSyncStatus: "",
    lastSyncError: "",
    lastSyncTotal: 0,
    lastSyncImported: 0,
    proxyCount: 0,
    healthyCount: 0
  };
  const response = await adminRequest("/admin/v1/proxies/webshare");
  if (!response?.ok) return { status: empty, error: true };
  try {
    const record = await response.json() as AdminWebshareStatus;
    return {
      status: {
        configured: record.configured,
        planId: record.plan_id,
        enabled: record.enabled,
        keyVersion: record.key_version,
        lastSyncAt: record.last_sync_at,
        lastSyncStatus: record.last_sync_status,
        lastSyncError: record.last_sync_error,
        lastSyncTotal: record.last_sync_total,
        lastSyncImported: record.last_sync_imported,
        proxyCount: record.proxy_count,
        healthyCount: record.healthy_count
      },
      error: false
    };
  } catch {
    return { status: empty, error: true };
  }
}
