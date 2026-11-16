import type { AdminAPIKey } from "./admin-api";

export type APIKeyStatus = AdminAPIKey["status"];

export type APIKeyFormValues = {
  name: string;
  scopes: string[];
  neverExpires: boolean;
  expiresAt: string;
  ipAllowlist: string;
  rpmLimit: string;
  tpmLimit: string;
  usdMonthly: string;
  usdDaily: string;
  usdWeekly: string;
};

export type APIKeyPayload = {
  name: string;
  scopes: string[];
  expires_at: string | null;
  ip_allowlist: string[];
  rpm_limit: number | null;
  tpm_limit: number | null;
  usd_monthly_limit: number | null;
  usd_daily_limit: number | null;
  usd_weekly_limit: number | null;
};

export type ValidationIssue = { key: string; params?: Record<string, string> };

export function emptyFormValues(): APIKeyFormValues {
  return {
    name: "",
    scopes: [],
    neverExpires: true,
    expiresAt: "",
    ipAllowlist: "",
    rpmLimit: "",
    tpmLimit: "",
    usdMonthly: "",
    usdDaily: "",
    usdWeekly: "",
  };
}

export function toFormValues(key: AdminAPIKey): APIKeyFormValues {
  return {
    name: key.name,
    scopes: key.scopes,
    neverExpires: !key.expires_at,
    expiresAt: key.expires_at ? toLocalInput(key.expires_at) : "",
    ipAllowlist: (key.ip_allowlist ?? []).join("\n"),
    rpmLimit: key.rpm_limit == null ? "" : String(key.rpm_limit),
    tpmLimit: key.tpm_limit == null ? "" : String(key.tpm_limit),
    usdMonthly: key.usd_monthly_limit == null ? "" : String(key.usd_monthly_limit),
    usdDaily: key.usd_daily_limit == null ? "" : String(key.usd_daily_limit),
    usdWeekly: key.usd_weekly_limit == null ? "" : String(key.usd_weekly_limit),
  };
}

export function parseIPAllowlist(value: string): string[] {
  return value
    .split(/[\s,]+/)
    .map(item => item.trim())
    .filter(Boolean);
}

export function validateForm(values: APIKeyFormValues): ValidationIssue | null {
  if (!values.name.trim()) return { key: "api.invalidName" };
  if (values.scopes.length === 0) return { key: "api.invalidScopes" };
  if (!values.neverExpires) {
    const expires = new Date(values.expiresAt);
    if (!values.expiresAt || Number.isNaN(expires.getTime()) || expires.getTime() <= Date.now()) {
      return { key: "api.invalidExpiry" };
    }
  }
  for (const ip of parseIPAllowlist(values.ipAllowlist)) {
    if (!isIPOrCIDR(ip)) return { key: "api.invalidIps", params: { value: ip } };
  }
  for (const value of [values.rpmLimit, values.tpmLimit, values.usdMonthly, values.usdDaily, values.usdWeekly]) {
    if (value.trim() === "") continue;
    const number = Number(value);
    if (!Number.isFinite(number) || number < 0) return { key: "api.invalidLimits" };
  }
  return null;
}

export function toPayload(values: APIKeyFormValues): APIKeyPayload {
  return {
    name: values.name.trim(),
    scopes: values.scopes,
    expires_at: values.neverExpires ? null : new Date(values.expiresAt).toISOString(),
    ip_allowlist: parseIPAllowlist(values.ipAllowlist),
    rpm_limit: integerOrNull(values.rpmLimit),
    tpm_limit: integerOrNull(values.tpmLimit),
    usd_monthly_limit: numberOrNull(values.usdMonthly),
    usd_daily_limit: numberOrNull(values.usdDaily),
    usd_weekly_limit: numberOrNull(values.usdWeekly),
  };
}

export function statusKey(status: APIKeyStatus): string {
  if (status === "revoked") return "api.statusRevoked";
  if (status === "expired") return "api.statusExpired";
  return "api.statusActive";
}

export function statusClass(status: APIKeyStatus): string {
  return status === "active" ? "chip ok" : "chip warn";
}

export function formatDate(value: string | null | undefined): string {
  if (!value) return "â€”";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "â€”";
  return date.toLocaleString();
}

export function countExpiring(keys: AdminAPIKey[]): number {
  const now = Date.now();
  const soon = now + 7 * 24 * 60 * 60 * 1000;
  return keys.filter(key => key.status === "active" && key.expires_at && new Date(key.expires_at).getTime() <= soon).length;
}

export function scopeOptions(catalog: string[], key?: AdminAPIKey | null): string[] {
  const result = [...catalog];
  for (const scope of key?.scopes ?? []) {
    if (!result.includes(scope)) result.push(scope);
  }
  return result;
}

function integerOrNull(value: string): number | null {
  if (value.trim() === "") return null;
  const number = Number(value);
  return Number.isFinite(number) ? Math.trunc(number) : null;
}

function numberOrNull(value: string): number | null {
  if (value.trim() === "") return null;
  const number = Number(value);
  return Number.isFinite(number) ? number : null;
}

function isIPOrCIDR(value: string): boolean {
  if (!/^[0-9a-fA-F:.]+(?:\/\d{1,3})?$/.test(value)) return false;
  if (!value.includes("/")) return true;
  const prefix = Number(value.split("/")[1]);
  return Number.isInteger(prefix) && prefix >= 0 && prefix <= 128;
}

function toLocalInput(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000);
  return local.toISOString().slice(0, 16);
}
