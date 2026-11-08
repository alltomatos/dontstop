import { NextResponse } from "next/server";
import { getAdminKey } from "@/lib/admin-auth";

export async function POST(request: Request) {
  const base = process.env.DONTSTOP_GATEWAY_URL;
  const key = await getAdminKey();
  if (!base || !key) {
    return NextResponse.json({ error: "gateway_not_configured" }, { status: 503 });
  }
  const body = await request.json().catch(() => null);
  if (!body) {
    return NextResponse.json({ error: "invalid_request" }, { status: 400 });
  }
  const upstream = await fetch(`${base}/admin/v1/providers`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${key}`,
      "Content-Type": "application/json"
    },
    body: JSON.stringify(body),
    cache: "no-store"
  }).catch(() => null);
  if (!upstream) {
    return NextResponse.json({ error: "gateway_unavailable" }, { status: 502 });
  }
  const payload = await upstream.json().catch(() => ({ error: "invalid_response" }));
  return NextResponse.json(payload, { status: upstream.status });
}
