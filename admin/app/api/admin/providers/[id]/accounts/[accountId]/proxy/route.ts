import { NextResponse } from "next/server";
import { getAdminKey } from "@/lib/admin-auth";

async function gateway(
  request: Request,
  method: "PUT" | "DELETE",
  id: string,
  accountId: string
): Promise<Response> {
  const base = process.env.DONTSTOP_GATEWAY_URL;
  const key = await getAdminKey();
  if (!base || !key) {
    return NextResponse.json({ error: "admin_api_not_configured" }, { status: 503 });
  }
  const body = method === "PUT" ? await request.text() : undefined;
  const response = await fetch(
    `${base.replace(/\/$/, "")}/admin/v1/providers/${encodeURIComponent(id)}/accounts/${encodeURIComponent(accountId)}/proxy`,
    {
      method,
      headers: {
        Authorization: `Bearer ${key}`,
        ...(body ? { "Content-Type": "application/json" } : {})
      },
      body
    }
  );
  return new NextResponse(await response.text(), {
    status: response.status,
    headers: { "Content-Type": response.headers.get("Content-Type") ?? "application/json" }
  });
}

export async function PUT(
  request: Request,
  { params }: { params: Promise<{ id: string; accountId: string }> }
) {
  const { id, accountId } = await params;
  return gateway(request, "PUT", id, accountId);
}

export async function DELETE(
  request: Request,
  { params }: { params: Promise<{ id: string; accountId: string }> }
) {
  const { id, accountId } = await params;
  return gateway(request, "DELETE", id, accountId);
}
