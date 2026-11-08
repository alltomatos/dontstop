import { NextResponse } from "next/server";
import { SESSION_COOKIE } from "@/lib/admin-auth";

export async function POST(request: Request) {
  const body = await request.json().catch(() => null) as {
    name?: string;
    email?: string;
    password?: string;
  } | null;
  const base = process.env.DONTSTOP_GATEWAY_URL;
  if (!body?.name || !body.email || !body.password || !base) {
    return NextResponse.json({ error: "invalid_request" }, { status: 400 });
  }
  const upstream = await fetch(`${base}/admin/v1/setup`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    cache: "no-store"
  }).catch(() => null);
  if (!upstream?.ok) {
    return NextResponse.json({ error: "setup_failed" }, { status: upstream?.status ?? 502 });
  }
  const session = await upstream.json().catch(() => null) as { token?: string } | null;
  if (!session?.token) {
    return NextResponse.json({ error: "invalid_session" }, { status: 502 });
  }
  const response = new NextResponse(null, { status: 204 });
  response.cookies.set(SESSION_COOKIE, session.token, {
    httpOnly: true,
    sameSite: "strict",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: 60 * 60 * 8
  });
  return response;
}
