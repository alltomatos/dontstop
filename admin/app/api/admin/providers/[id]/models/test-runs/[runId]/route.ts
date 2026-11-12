import { NextResponse } from "next/server";
import { getAdminKey } from "@/lib/admin-auth";

export async function GET(
  _request: Request,
  context: { params: Promise<{ id: string; runId: string }> }
) {
  const base = process.env.DONTSTOP_GATEWAY_URL;
  const key = await getAdminKey();
  const { id, runId } = await context.params;
  if (!base || !key) {
    return NextResponse.json({ error: "admin_api_not_configured" }, { status: 503 });
  }
  const response = await fetch(
    `${base.replace(/\/$/, "")}/admin/v1/providers/${encodeURIComponent(id)}/models/test-runs/${encodeURIComponent(runId)}`,
    {
      method: "GET",
      headers: { Authorization: `Bearer ${key}` },
      cache: "no-store"
    }
  ).catch(() => null);
  if (!response) {
    return NextResponse.json({ error: "admin_api_unavailable" }, { status: 502 });
  }
  return new NextResponse(await response.text(), {
    status: response.status,
    headers: { "Content-Type": response.headers.get("Content-Type") ?? "application/json" }
  });
}
