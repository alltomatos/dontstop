import { NextResponse } from "next/server";
import { getAdminKey } from "@/lib/admin-auth";

async function proxy(request: Request, id: string, method: "GET" | "PUT") {
  const base = process.env.DONTSTOP_GATEWAY_URL;
  const key = await getAdminKey();
  if (!base || !key) {
    return NextResponse.json({ error: "admin_api_not_configured" }, { status: 503 });
  }
  try {
    const response = await fetch(`${base.replace(/\/$/, "")}/admin/v1/api-keys/${encodeURIComponent(id)}`, {
      method,
      headers: {
        Authorization: `Bearer ${key}`,
        ...(method === "PUT" ? { "Content-Type": "application/json" } : {}),
      },
      body: method === "PUT" ? await request.text() : undefined,
      cache: "no-store",
    });
    return new NextResponse(await response.text(), {
      status: response.status,
      headers: { "Content-Type": response.headers.get("Content-Type") ?? "application/json" },
    });
  } catch {
    return NextResponse.json({ error: "admin_api_unavailable" }, { status: 503 });
  }
}

export async function GET(
  request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  const { id } = await params;
  return proxy(request, id, "GET");
}

export async function PUT(
  request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  const { id } = await params;
  return proxy(request, id, "PUT");
}
