import { NextResponse } from "next/server";
import { getAdminKey } from "@/lib/admin-auth";

async function proxy(
  request: Request,
  path: string,
  method: "PATCH" | "PUT" | "DELETE"
): Promise<Response> {
  const base = process.env.DONTSTOP_GATEWAY_URL;
  const key = await getAdminKey();
  if (!base || !key) {
    return NextResponse.json({ error: "admin_api_not_configured" }, { status: 503 });
  }
  const body = method === "DELETE" ? undefined : await request.text();
  const response = await fetch(`${base.replace(/\/$/, "")}${path}`, {
    method,
    headers: {
      Authorization: `Bearer ${key}`,
      ...(body ? { "Content-Type": "application/json" } : {})
    },
    body
  });
  return new NextResponse(await response.text(), {
    status: response.status,
    headers: { "Content-Type": response.headers.get("Content-Type") ?? "application/json" }
  });
}

export async function PATCH(
  request: Request,
  { params }: { params: Promise<{ id: string; accountId: string }> }
) {
  const { id, accountId } = await params;
  return proxy(
    request,
    `/admin/v1/providers/${encodeURIComponent(id)}/accounts/${encodeURIComponent(accountId)}`,
    "PATCH"
  );
}

export async function DELETE(
  _request: Request,
  { params }: { params: Promise<{ id: string; accountId: string }> }
) {
  const base = process.env.DONTSTOP_GATEWAY_URL;
  const key = await getAdminKey();
  if (!base || !key) {
    return NextResponse.json({ error: "admin_api_not_configured" }, { status: 503 });
  }
  const { id, accountId } = await params;
  const response = await fetch(
    `${base.replace(/\/$/, "")}/admin/v1/providers/${encodeURIComponent(id)}/accounts/${encodeURIComponent(accountId)}`,
    {
      method: "DELETE",
      headers: { Authorization: `Bearer ${key}` }
    }
  );
  return new NextResponse(await response.text(), {
    status: response.status,
    headers: { "Content-Type": response.headers.get("Content-Type") ?? "application/json" }
  });
}
