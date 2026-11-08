import { NextResponse } from "next/server";

export async function GET() {
  const base = process.env.DONTSTOP_GATEWAY_URL;
  if (!base) {
    return NextResponse.json({ needs_setup: false });
  }
  const upstream = await fetch(`${base}/admin/v1/setup/status`, { cache: "no-store" }).catch(() => null);
  if (!upstream?.ok) {
    return NextResponse.json({ needs_setup: false });
  }
  return NextResponse.json(await upstream.json().catch(() => ({ needs_setup: false })));
}
