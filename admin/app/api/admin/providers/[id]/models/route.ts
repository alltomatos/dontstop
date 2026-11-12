import { NextResponse } from "next/server";
import { getAdminKey } from "@/lib/admin-auth";

export async function GET(_request: Request, context: { params: Promise<{ id: string }> }) {
  const base = process.env.DONTSTOP_GATEWAY_URL;
  const key = await getAdminKey();
  const { id } = await context.params;
  if (!base || !key) {
    return NextResponse.json({ error: "admin_api_not_configured" }, { status: 503 });
  }
  const response = await fetch(`${base.replace(/\/$/, "")}/admin/v1/providers/${encodeURIComponent(id)}/models`, {
    method: "GET",
    headers: { Authorization: `Bearer ${key}`, "Content-Type": "application/json" },
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

export async function PATCH(request: Request, context: { params: Promise<{ id: string }> }) {
  const base = process.env.DONTSTOP_GATEWAY_URL;
  const key = await getAdminKey();
  const { id } = await context.params;
  if (!base || !key) {
    return NextResponse.json({ error: "admin_api_not_configured" }, { status: 503 });
  }
  const response = await fetch(`${base.replace(/\/$/, "")}/admin/v1/providers/${encodeURIComponent(id)}/models`, {
    method: "PATCH",
    headers: { Authorization: `Bearer ${key}`, "Content-Type": "application/json" },
    body: await request.text()
  }).catch(() => null);
  if (!response) {
    return NextResponse.json({ error: "admin_api_unavailable" }, { status: 502 });
  }
  return new NextResponse(await response.text(), {
    status: response.status,
    headers: { "Content-Type": response.headers.get("Content-Type") ?? "application/json" }
  });
}
