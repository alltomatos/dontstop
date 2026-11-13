import { NextResponse } from "next/server";
import { getAdminKey } from "@/lib/admin-auth";

export async function POST() {
  const base = process.env.DONTSTOP_GATEWAY_URL;
  const key = await getAdminKey();
  if (!base || !key) {
    return NextResponse.json({ error: "admin_api_not_configured" }, { status: 503 });
  }
  const response = await fetch(
    `${base.replace(/\/$/, "")}/admin/v1/proxies/webshare/refresh`,
    {
      method: "POST",
      headers: { Authorization: `Bearer ${key}` }
    }
  );
  return new NextResponse(await response.text(), {
    status: response.status,
    headers: { "Content-Type": response.headers.get("Content-Type") ?? "application/json" }
  });
}
