import { NextResponse } from "next/server";
import { SESSION_COOKIE } from "@/lib/admin-auth";

export async function POST(request: Request) {
  const body = await request.json().catch(() => null) as { email?: string; password?: string } | null;
  const email = body?.email?.trim();
  const password = body?.password ?? "";
  const base = process.env.DONTSTOP_GATEWAY_URL;
  if (!email || !password || !base) {
    return NextResponse.json({ error: "invalid_request" }, { status: 400 });
  }
  const upstream = await fetch(`${base}/admin/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
    cache: "no-store"
  }).catch(() => null);
  if (upstream?.status === 429) {
    return NextResponse.json(
      { error: "rate_limited" },
      { status: 429, headers: { "Retry-After": upstream.headers.get("Retry-After") ?? "300" } }
    );
  }
  if (!upstream?.ok) {
    return NextResponse.json({ error: "unauthorized" }, { status: 401 });
  }
  const session = await upstream.json().catch(() => null) as { token?: string } | null;
  if (!session?.token) {
    return NextResponse.json({ error: "invalid_session" }, { status: 502 });
  }
  const response = new NextResponse(null, { status: 204 });
  response.cookies.set(SESSION_COOKIE, session.token, {
    httpOnly: true,
    sameSite: "strict",
    secure: process.env.NODE_ENV === "production" && process.env.DONTSTOP_INSECURE_COOKIES !== "on",
    path: "/",
    maxAge: 60 * 60 * 8
  });
  return response;
}
