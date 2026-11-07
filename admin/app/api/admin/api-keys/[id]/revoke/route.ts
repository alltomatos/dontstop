import { NextResponse } from "next/server";

export async function POST(
  _request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  const base = process.env.DONTSTOP_GATEWAY_URL;
  const key = process.env.DONTSTOP_ADMIN_API_KEY;
  if (!base || !key) {
    return NextResponse.json({ error: "admin_api_not_configured" }, { status: 503 });
  }
  const { id } = await params;
  try {
    const response = await fetch(`${base.replace(/\/$/, "")}/admin/v1/api-keys/${encodeURIComponent(id)}/revoke`, {
      method: "POST",
      headers: { Authorization: `Bearer ${key}` }
    });
    return new NextResponse(await response.text(), {
      status: response.status,
      headers: { "Content-Type": response.headers.get("Content-Type") ?? "application/json" }
    });
  } catch {
    return NextResponse.json({ error: "admin_api_unavailable" }, { status: 503 });
  }
}
