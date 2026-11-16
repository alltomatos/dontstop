import { NextResponse } from "next/server";
import { getAdminKey } from "@/lib/admin-auth";

async function proxy(request: Request, path: string, method: "GET" | "POST") {
  const base = process.env.DONTSTOP_GATEWAY_URL;
  const key = await getAdminKey();
  if (!base || !key) {
    return NextResponse.json({ error: "admin_api_not_configured" }, { status: 503 });
  }
  try {
    const response = await fetch(`${base.replace(/\/$/, "")}${path}`, {
      method,
      headers: {
        Authorization: `Bearer ${key}`,
        ...(method === "POST" ? { "Content-Type": "application/json" } : {}),
      },
      body: method === "POST" ? await request.text() : undefined,
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

export async function GET(request: Request) {
  return proxy(request, "/admin/v1/api-keys", "GET");
}

export async function POST(request: Request) {
  return proxy(request, "/admin/v1/api-keys", "POST");
}
