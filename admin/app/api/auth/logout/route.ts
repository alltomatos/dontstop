import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { ADMIN_COOKIE, SESSION_COOKIE } from "@/lib/admin-auth";

export async function POST() {
  const store = await cookies();
  const token = store.get(SESSION_COOKIE)?.value;
  const base = process.env.DONTSTOP_GATEWAY_URL;
  if (token && base) {
    await fetch(`${base}/admin/v1/auth/logout`, {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store"
    }).catch(() => null);
  }
  const response = new NextResponse(null, { status: 204 });
  for (const name of [SESSION_COOKIE, ADMIN_COOKIE]) {
    response.cookies.set(name, "", {
      httpOnly: true,
      sameSite: "strict",
      secure: process.env.NODE_ENV === "production",
      path: "/",
      maxAge: 0
    });
  }
  return response;
}
