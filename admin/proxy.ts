import { NextResponse, type NextRequest } from "next/server";
import { consoleAuthDisabled } from "./lib/console-auth";

const ADMIN_COOKIE = "dontstop_admin_key";
const SESSION_COOKIE = "dontstop_session";

export function proxy(request: NextRequest) {
  if (consoleAuthDisabled()) {
    return NextResponse.next();
  }
  if (request.cookies.get(SESSION_COOKIE)?.value || request.cookies.get(ADMIN_COOKIE)?.value) {
    return NextResponse.next();
  }
  const url = request.nextUrl.clone();
  url.pathname = "/login";
  url.searchParams.set("next", request.nextUrl.pathname);
  return NextResponse.redirect(url);
}

export const config = {
  matcher: ["/((?!login|onboarding|api/auth|_next/static|_next/image|favicon.ico).*)"]
};
