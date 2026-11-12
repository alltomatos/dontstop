import { NextResponse } from "next/server";
import { getAdminKey } from "@/lib/admin-auth";

async function forward(request: Request, path: string, method: "GET" | "PUT"): Promise<Response> {
  const base = process.env.DONTSTOP_GATEWAY_URL;
  const key = await getAdminKey();
  if (!base || !key) {
    return NextResponse.json({ error: "admin_api_not_configured" }, { status: 503 });
  }
  const body = method === "PUT" ? await request.text() : undefined;
  const response = await fetch(`${base.replace(/\/$/, "")}${path}`, {
    method,
    headers: {
      Authorization: `Bearer ${key}`,
      ...(body ? { "Content-Type": "application/json" } : {})
    },
    body,
    cache: "no-store"
  });
  return new NextResponse(await response.text(), {
    status: response.status,
    headers: { "Content-Type": response.headers.get("Content-Type") ?? "application/json" }
  });
}

export async function GET(request: Request) {
  return forward(request, "/admin/v1/proxies/webshare", "GET");
}

export async function PUT(request: Request) {
  return forward(request, "/admin/v1/proxies/webshare", "PUT");
}
