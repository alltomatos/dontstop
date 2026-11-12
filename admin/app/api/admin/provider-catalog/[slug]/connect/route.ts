import { NextResponse } from "next/server";
import { getAdminKey } from "@/lib/admin-auth";

export async function POST(
  request: Request,
  { params }: { params: Promise<{ slug: string }> }
) {
  const base = process.env.DONTSTOP_GATEWAY_URL;
  const key = await getAdminKey();
  if (!base || !key) {
    return NextResponse.json({ error: "admin_api_not_configured" }, { status: 503 });
  }
  const { slug } = await params;
  const response = await fetch(
    `${base.replace(/\/$/, "")}/admin/v1/provider-catalog/${encodeURIComponent(slug)}/connect`,
    {
      method: "POST",
      headers: {
        Authorization: `Bearer ${key}`,
        "Content-Type": "application/json"
      },
      body: await request.text()
    }
  );
  return new NextResponse(await response.text(), {
    status: response.status,
    headers: { "Content-Type": response.headers.get("Content-Type") ?? "application/json" }
  });
}
