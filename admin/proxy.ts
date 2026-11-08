import { NextResponse, type NextRequest } from "next/server";

const ADMIN_COOKIE = "dontstop_admin_key";

export function proxy(request: NextRequest) {
  if (process.env.DONTSTOP_CONSOLE_AUTH !== "required") {
    return NextResponse.next();
  }
  if (request.cookies.get(ADMIN_COOKIE)?.value) {
    return NextResponse.next();
  }
  const url = request.nextUrl.clone();
  url.pathname = "/login";
  url.searchParams.set("next", request.nextUrl.pathname);
  return NextResponse.redirect(url);
}

export const config = {
  matcher: ["/((?!login|api/auth|_next/static|_next/image|favicon.ico).*)"]
};
