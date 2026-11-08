import { NextResponse } from "next/server";
import { ADMIN_COOKIE } from "@/lib/admin-auth";

export async function POST(request: Request) {
  const body = await request.json().catch(() => null) as { key?: string } | null;
  const key = body?.key?.trim();
  const base = process.env.DONTSTOP_GATEWAY_URL;
  if (!key || !base) {
    return NextResponse.json({ error: "invalid_request" }, { status: 400 });
  }
  const upstream = await fetch(`${base}/admin/v1/providers`, {
    headers: { Authorization: `Bearer ${key}` },
    cache: "no-store"
  }).catch(() => null);
  if (!upstream || !upstream.ok) {
    return NextResponse.json({ error: "unauthorized" }, { status: 401 });
  }
  const response = new NextResponse(null, { status: 204 });
  response.cookies.set(ADMIN_COOKIE, key, {
    httpOnly: true,
    sameSite: "strict",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: 60 * 60 * 8
  });
  return response;
}
