import fs from "node:fs/promises";
import path from "node:path";
import { NextResponse } from "next/server";
import { createCache, readConfig, resolveGuideContent } from "@/src/guide-source";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

// One cache per server process: short TTL keeps GitHub traffic bounded while
// still reflecting committed guide changes on the next load.
const cache = createCache();

async function readLocalGuide(locale: string, slug: string): Promise<string | null> {
  if (!/^[a-z0-9-]+$/i.test(locale) || !/^[a-z0-9-]+$/i.test(slug)) return null;
  try {
    return await fs.readFile(
      path.join(process.cwd(), "public", "guide", locale, `${slug}.html`),
      "utf8"
    );
  } catch {
    return null;
  }
}

export async function GET(
  request: Request,
  { params }: { params: Promise<{ locale: string; slug: string }> }
) {
  const { locale, slug } = await params;
  const config = readConfig(process.env);
  const forceRefresh = new URL(request.url).searchParams.get("refresh") === "1";

  const result = await resolveGuideContent({
    locale,
    slug,
    config,
    cache,
    fetchFn: fetch,
    readLocal: readLocalGuide,
    forceRefresh
  });

  return new NextResponse(result.html, {
    status: 200,
    headers: {
      "Content-Type": "text/html; charset=utf-8",
      "Cache-Control": `public, max-age=0, s-maxage=${Math.round(config.ttlMs / 1000)}, stale-while-revalidate=${Math.round((config.ttlMs / 1000) * 2)}`,
      "X-Dontstop-Guide-Source": result.source,
      "X-Dontstop-Guide-Locale": result.locale
    }
  });
}
