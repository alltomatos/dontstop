import { NextResponse } from "next/server";

export async function PATCH(request: Request) {
  const base = process.env.DONTSTOP_GATEWAY_URL;
  const key = process.env.DONTSTOP_ADMIN_API_KEY;
  if (!base || !key) {
    return NextResponse.json({ error: "admin_api_not_configured" }, { status: 503 });
  }
  try {
    const response = await fetch(`${base.replace(/\/$/, "")}/admin/v1/mcp/settings`, {
      method: "PATCH",
      headers: { Authorization: `Bearer ${key}`, "Content-Type": "application/json" },
      body: await request.text()
    });
    return new NextResponse(await response.text(), {
      status: response.status,
      headers: { "Content-Type": response.headers.get("Content-Type") ?? "application/json" }
    });
  } catch {
    return NextResponse.json({ error: "admin_api_unavailable" }, { status: 503 });
  }
}
