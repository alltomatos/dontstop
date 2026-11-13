import { NextResponse } from "next/server";
import { getAdminKey } from "@/lib/admin-auth";

export async function GET() {
  return proxy("/admin/v1/combos", "GET");
}

export async function POST(request: Request) {
  return proxy("/admin/v1/combos", "POST", await request.text());
}

async function proxy(path: string, method: string, body?: string) {
  const base = process.env.DONTSTOP_GATEWAY_URL;
  const key = await getAdminKey();
  if (!base || !key) {
    return NextResponse.json({ error: "admin_api_not_configured" }, { status: 503 });
  }
  const response = await fetch(`${base.replace(/\/$/, "")}${path}`, {
    method,
    headers: {
      Authorization: `Bearer ${key}`,
      ...(body ? { "Content-Type": "application/json" } : {})
    },
    body,
    cache: "no-store"
  }).catch(() => null);
  if (!response) {
    return NextResponse.json({ error: "admin_api_unavailable" }, { status: 502 });
  }
  return new NextResponse(await response.text(), {
    status: response.status,
    headers: { "Content-Type": response.headers.get("Content-Type") ?? "application/json" }
  });
}
