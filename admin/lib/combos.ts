import { getAdminKey } from "./admin-auth";

export type ComboStatus = "draft" | "active" | "inactive";
export type ComboStrategy = "priority" | "round_robin" | "priority_round_robin";

export interface ComboMember {
  id?: string;
  provider_id: string;
  provider_name?: string;
  model_id: string;
  model_name?: string;
  priority: number;
  position: number;
  weight: number;
  enabled: boolean;
  constraints?: Record<string, unknown>;
  usable?: boolean;
  reason?: string;
}

export interface Combo {
  id: string;
  slug: string;
  name: string;
  description: string;
  strategy: ComboStrategy;
  failover_enabled: boolean;
  status: ComboStatus;
  revision: number;
  rpm_limit?: number | null;
  tpm_limit?: number | null;
  max_attempts: number;
  source: string;
  member_count?: number;
  provider_count?: number;
  usable_members?: number;
  members?: ComboMember[];
}

export interface ComboCatalogModel {
  provider_id: string;
  provider_name: string;
  model_id: string;
  display_name: string;
  capabilities: Record<string, unknown>;
  usable: boolean;
  reason?: string;
}

export interface ComboAIInput {
  description: string;
  provider_id?: string;
  model_id?: string;
  messages: Array<{ role: string; content: string }>;
}

export interface ComboAIDraft {
  message: string;
  name: string;
  description: string;
  strategy: ComboStrategy;
  failover_enabled: boolean;
  members: ComboMember[];
  warnings: string[];
  assistant_provider_id: string;
  assistant_model_id: string;
}

async function adminGet(path: string): Promise<Response | null> {
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

export async function getCombosResult(): Promise<{ combos: Combo[]; error: boolean }> {
  const response = await adminGet("/admin/v1/combos");
  if (!response?.ok) return { combos: [], error: true };
  try {
    const body = await response.json() as { data?: Combo[] };
    return { combos: Array.isArray(body.data) ? body.data : [], error: false };
  } catch {
    return { combos: [], error: true };
  }
}

export async function getComboCatalogResult(): Promise<{ models: ComboCatalogModel[]; error: boolean }> {
  const response = await adminGet("/admin/v1/combos/catalog");
  if (!response?.ok) return { models: [], error: true };
  try {
    const body = await response.json() as { data?: ComboCatalogModel[] };
    return { models: Array.isArray(body.data) ? body.data : [], error: false };
  } catch {
    return { models: [], error: true };
  }
}
