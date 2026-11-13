import { NextResponse } from "next/server";
import { getAdminKey } from "@/lib/admin-auth";

type Context = { params: Promise<{ path: string[] }> };

export async function GET(request: Request, context: Context) {
  return proxy(request, context, "GET");
}

export async function POST(request: Request, context: Context) {
  return proxy(request, context, "POST");
}

export async function PATCH(request: Request, context: Context) {
  return proxy(request, context, "PATCH");
}

export async function DELETE(request: Request, context: Context) {
  return proxy(request, context, "DELETE");
}

async function proxy(request: Request, context: Context, method: string) {
  const base = process.env.DONTSTOP_GATEWAY_URL;
  const key = await getAdminKey();
  if (!base || !key) {
    return NextResponse.json({ error: "admin_api_not_configured" }, { status: 503 });
  }
  const { path } = await context.params;
  const suffix = path.map(encodeURIComponent).join("/");
  const query = new URL(request.url).search;
  const body = method === "GET" || method === "DELETE" ? undefined : await request.text();
  const response = await fetch(`${base.replace(/\/$/, "")}/admin/v1/combos/${suffix}${query}`, {
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
