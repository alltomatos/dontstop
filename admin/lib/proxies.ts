export type ProxyHealth = "healthy" | "quarantined" | "unhealthy";

export interface ProxyAccountSummary {
  id: string;
  name: string;
  providerId: string;
  providerName: string;
  proxyMode: "off" | "preferred" | "required";
}

export interface ProxyEndpoint {
  id: string;
  label: string;
  address: string;
  port: number;
  countryCode: string;
  cityName: string;
  source: "manual" | "webshare";
  valid: boolean;
  health: ProxyHealth;
  quarantinedUntil?: string;
  lastVerifiedAt?: string;
  assignedCount: number;
  accounts: ProxyAccountSummary[];
}

export interface WebshareStatus {
  configured: boolean;
  planId: string;
  enabled: boolean;
  keyVersion: number;
  lastSyncAt?: string;
  lastSyncStatus: string;
  lastSyncError: string;
  lastSyncTotal: number;
  lastSyncImported: number;
  proxyCount: number;
  healthyCount: number;
}
