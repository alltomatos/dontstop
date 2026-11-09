import { cookies } from "next/headers";
import { consoleAuthDisabled } from "./console-auth";

export const ADMIN_COOKIE = "dontstop_admin_key";
export const SESSION_COOKIE = "dontstop_session";

export async function getAdminKey(): Promise<string | undefined> {
  const store = await cookies();
  return (
    store.get(SESSION_COOKIE)?.value ||
    store.get(ADMIN_COOKIE)?.value ||
    process.env.DONTSTOP_ADMIN_API_KEY
  );
}

export type ConsoleSessionState = "authenticated" | "unauthenticated" | "unavailable";

export async function consoleSessionState(): Promise<ConsoleSessionState> {
  if (consoleAuthDisabled()) return "authenticated";
  const store = await cookies();
  const token = store.get(SESSION_COOKIE)?.value;
  if (!token) return "unauthenticated";
  const base = process.env.DONTSTOP_GATEWAY_URL;
  if (!base) return "unavailable";
  const response = await fetch(`${base.replace(/\/$/, "")}/admin/v1/auth/me`, {
    headers: { Authorization: `Bearer ${token}` },
    cache: "no-store",
    signal: AbortSignal.timeout(5000)
  }).catch(() => null);
  if (!response) return "unavailable";
  if (response.status === 401 || response.status === 403) return "unauthenticated";
  if (!response.ok) return "unavailable";
  return "authenticated";
}
