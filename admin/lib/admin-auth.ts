import { cookies } from "next/headers";

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

export function consoleAuthRequired(): boolean {
  return process.env.DONTSTOP_CONSOLE_AUTH === "required";
}
