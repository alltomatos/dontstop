const test = require("node:test");
const assert = require("node:assert/strict");
const source = require("../src/guide-source.js");

const githubConfig = source.readConfig({
  DONTSTOP_GUIDE_SOURCE: "github",
  DONTSTOP_GUIDE_REPO: "alltomatos/dontstop",
  DONTSTOP_GUIDE_REF: "main",
  DONTSTOP_GUIDE_CACHE_TTL_SECONDS: "60"
});

function response(status, body = "", headers = {}) {
  return {
    status,
    ok: status >= 200 && status < 300,
    headers: { get: key => headers[key] ?? headers[key.toLowerCase()] ?? null },
    text: async () => body
  };
}

test("source defaults to local and github is opt-in", () => {
  const local = source.readConfig({});
  assert.equal(local.source, "local");
  assert.equal(local.repo, "alltomatos/dontstop");
  assert.equal(local.ref, "main");
  assert.equal(local.ttlMs, 60000);
  assert.equal(githubConfig.source, "github");
});

test("locale fallback prefers the requested locale then english", () => {
  assert.deepEqual(source.localeCandidates("pt-BR"), ["pt-BR", "en"]);
  assert.deepEqual(source.localeCandidates("de"), ["de", "en"]);
  assert.deepEqual(source.localeCandidates("fr-CA"), ["fr", "en"]);
  assert.deepEqual(source.localeCandidates("xx"), ["en"]);
});

test("github urls point to the raw markdown on the configured ref", () => {
  assert.equal(
    source.rawGuideUrl(githubConfig, "pt-BR", "provider-setup"),
    "https://raw.githubusercontent.com/alltomatos/dontstop/main/docs/user-guide/pt-BR/provider-setup.md"
  );
});

test("remote markdown rewrites images and guide links", () => {
  const html = source.renderRemoteGuide(
    "# Título\n\n![Print](../assets/pt-BR/provider-models.png)\n\nVeja [Providers](providers.md#models) e o [índice](../README.md).",
    { locale: "pt-BR", config: githubConfig }
  );
  assert.ok(html.includes('src="https://raw.githubusercontent.com/alltomatos/dontstop/main/docs/user-guide/assets/pt-BR/provider-models.png"'));
  assert.ok(html.includes('href="/guide/pt-BR/providers#models"'));
  assert.ok(html.includes('href="/guide/pt-BR"'));
});

test("local source reads the embedded html", async () => {
  const result = await source.resolveGuideContent({
    locale: "pt-BR",
    slug: "provider-setup",
    config: source.readConfig({}),
    fetchFn: async () => response(500),
    readLocal: async (locale, slug) => `<html>${locale}/${slug}</html>`
  });
  assert.equal(result.source, "local");
  assert.equal(result.locale, "pt-BR");
  assert.match(result.html, /pt-BR\/provider-setup/);
});

test("github source renders and caches the remote guide", async () => {
  let calls = 0;
  let current = 1000;
  const cache = source.createCache();
  const fetchFn = async () => {
    calls += 1;
    return response(200, "# OpenRouter\n\n![x](../assets/en/openrouter.png)");
  };
  const first = await source.resolveGuideContent({
    locale: "en",
    slug: "provider-setup",
    config: githubConfig,
    cache,
    fetchFn,
    readLocal: async () => null,
    now: () => current
  });
  assert.equal(first.source, "github");
  assert.ok(first.html.includes('src="https://raw.githubusercontent.com/alltomatos/dontstop/main/docs/user-guide/assets/en/openrouter.png"'));

  const second = await source.resolveGuideContent({
    locale: "en",
    slug: "provider-setup",
    config: githubConfig,
    cache,
    fetchFn,
    readLocal: async () => null,
    now: () => current
  });
  assert.equal(second.source, "github-cache");
  assert.equal(calls, 1);

  current += 61000;
  await source.resolveGuideContent({
    locale: "en",
    slug: "provider-setup",
    config: githubConfig,
    cache,
    fetchFn,
    readLocal: async () => null,
    now: () => current
  });
  assert.equal(calls, 2);
});

test("force refresh respects the cooldown", async () => {
  let calls = 0;
  const cache = source.createCache();
  const fetchFn = async () => {
    calls += 1;
    return response(200, "# A");
  };
  await source.resolveGuideContent({
    locale: "en", slug: "dashboard", config: githubConfig, cache, fetchFn,
    readLocal: async () => null, now: () => 5000
  });
  const forced = await source.resolveGuideContent({
    locale: "en", slug: "dashboard", config: githubConfig, cache, fetchFn,
    readLocal: async () => null, now: () => 6000, forceRefresh: true
  });
  assert.equal(forced.source, "github-cooldown");
  assert.equal(calls, 1);
});

test("missing locale falls back to english on github", async () => {
  const fetchFn = async url => {
    if (url.includes("/pt-BR/")) return response(404);
    return response(200, "# English guide");
  };
  const result = await source.resolveGuideContent({
    locale: "pt-BR", slug: "dashboard", config: githubConfig, cache: source.createCache(),
    fetchFn, readLocal: async () => null, now: () => 1
  });
  assert.equal(result.source, "github");
  assert.equal(result.locale, "en");
});

test("offline github falls back to the embedded local html", async () => {
  const result = await source.resolveGuideContent({
    locale: "es", slug: "provider-setup", config: githubConfig, cache: source.createCache(),
    fetchFn: async () => { throw new Error("offline"); },
    readLocal: async (locale, slug) => `<html>${locale}/${slug}</html>`,
    now: () => 1
  });
  assert.equal(result.source, "local-fallback");
  assert.equal(result.locale, "es");
  assert.match(result.html, /es\/provider-setup/);
});

test("stale cache is served when github fails", async () => {
  const cache = source.createCache();
  let current = 1;
  const okFetch = async () => response(200, "# Cached");
  await source.resolveGuideContent({
    locale: "en", slug: "cache", config: githubConfig, cache, fetchFn: okFetch,
    readLocal: async () => null, now: () => current
  });
  current += 120000;
  const stale = await source.resolveGuideContent({
    locale: "en", slug: "cache", config: githubConfig, cache,
    fetchFn: async () => { throw new Error("offline"); },
    readLocal: async () => null, now: () => current
  });
  assert.equal(stale.source, "github-stale");
  assert.match(stale.html, /Cached/);
});

test("unknown or unsafe slugs never break the route", async () => {
  assert.equal(source.sanitizeSlug("../etc/passwd"), null);
  assert.equal(source.sanitizeSlug("provider_setup"), null);
  const result = await source.resolveGuideContent({
    locale: "en", slug: "../etc/passwd", config: source.readConfig({}),
    fetchFn: async () => response(200, ""), readLocal: async () => "never"
  });
  assert.equal(result.source, "unavailable");
  assert.ok(result.html.includes("<h1>"));
});
