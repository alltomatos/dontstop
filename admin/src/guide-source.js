(function (root, factory) {
  const api = factory();
  if (typeof module === "object" && module.exports) module.exports = api;
  if (root) root.DontstopGuideSource = api;
})(typeof globalThis !== "undefined" ? globalThis : this, function () {
  const renderer = require("./guide-renderer.js");
  const { renderMarkdown, renderPage, extractTitle } = renderer;

  const SUPPORTED_LOCALES = ["en", "pt-PT", "pt-BR", "es", "fr", "de"];
  const DEFAULT_REPO = "alltomatos/dontstop";
  const DEFAULT_REF = "main";
  const DEFAULT_BASE_URL = "https://raw.githubusercontent.com";
  const DEFAULT_TTL_SECONDS = 60;
  const DEFAULT_TIMEOUT_MS = 4000;
  const FORCE_REFRESH_COOLDOWN_MS = 10000;
  const SLUG_PATTERN = /^[a-z0-9-]+$/;

  function normalizeLocale(input) {
    const raw = String(input || "").trim().replace(/_/g, "-").toLowerCase();
    if (raw === "pt" || raw === "pt-br") return "pt-BR";
    if (raw === "pt-pt") return "pt-PT";
    if (raw === "en" || raw.startsWith("en-")) return "en";
    if (raw === "es" || raw.startsWith("es-")) return "es";
    if (raw === "fr" || raw.startsWith("fr-")) return "fr";
    if (raw === "de" || raw.startsWith("de-")) return "de";
    return "en";
  }

  function localeCandidates(locale) {
    const normalized = normalizeLocale(locale);
    return normalized === "en" ? ["en"] : [normalized, "en"];
  }

  function sanitizeSlug(input) {
    const value = String(input || "").trim().toLowerCase();
    return SLUG_PATTERN.test(value) ? value : null;
  }

  function readConfig(env) {
    const source = String((env && env.DONTSTOP_GUIDE_SOURCE) || "local").trim().toLowerCase();
    const ttl = Number((env && env.DONTSTOP_GUIDE_CACHE_TTL_SECONDS) || DEFAULT_TTL_SECONDS);
    const timeout = Number((env && env.DONTSTOP_GUIDE_TIMEOUT_MS) || DEFAULT_TIMEOUT_MS);
    return {
      source: source === "github" ? "github" : "local",
      repo: String((env && env.DONTSTOP_GUIDE_REPO) || DEFAULT_REPO).trim(),
      ref: String((env && env.DONTSTOP_GUIDE_REF) || DEFAULT_REF).trim(),
      baseUrl: String((env && env.DONTSTOP_GUIDE_BASE_URL) || DEFAULT_BASE_URL).trim().replace(/\/+$/, ""),
      ttlMs: (Number.isFinite(ttl) && ttl > 0 ? ttl : DEFAULT_TTL_SECONDS) * 1000,
      timeoutMs: Number.isFinite(timeout) && timeout > 0 ? timeout : DEFAULT_TIMEOUT_MS,
      forceRefreshCooldownMs: FORCE_REFRESH_COOLDOWN_MS
    };
  }

  function rawGuideUrl(config, locale, slug) {
    return `${config.baseUrl}/${config.repo}/${config.ref}/docs/user-guide/${locale}/${slug}.md`;
  }

  function cacheKey(config, locale, slug) {
    return `${config.repo}@${config.ref}:${locale}:${slug}`;
  }

  function createCache() {
    return new Map();
  }

  function escapeAttr(value) {
    return String(value).replace(/&/g, "&amp;").replace(/"/g, "&quot;");
  }

  function remoteImageUrl(config, source) {
    const src = String(source || "").trim();
    if (/^(https?:|data:|#)/i.test(src)) return src;
    const stripped = src.replace(/^\.\.\//, "").replace(/^\.\//, "");
    return `${config.baseUrl}/${config.repo}/${config.ref}/docs/user-guide/${stripped}`;
  }

  function remoteGuideLink(locale, href) {
    const value = String(href || "").trim();
    if (!value) return value;
    if (/^(https?:|mailto:|#)/i.test(value)) return value;
    const [target, hash] = value.split("#");
    const clean = target.replace(/^\.\//, "");
    if (!clean.endsWith(".md")) return value;
    const slug = clean.replace(/\.md$/, "").replace(/^\.\.\//, "");
    if (slug === "README" || slug === "README.md" || slug === "") return `/guide/${locale}`;
    return `/guide/${locale}/${slug}${hash ? `#${hash}` : ""}`;
  }

  function renderRemoteGuide(markdown, options) {
    const locale = normalizeLocale(options.locale);
    const config = options.config;
    const title = extractTitle(markdown, "Guia");
    const body = renderMarkdown(markdown, {
      resolveImage: src => remoteImageUrl(config, src),
      resolveLink: href => remoteGuideLink(locale, href)
    });
    return renderPage(title, body, locale);
  }

  function unavailablePage(locale) {
    const normalized = normalizeLocale(locale);
    return renderPage(
      "Guia indisponível",
      "<h1>Guia indisponível</h1>\n<p>Não foi possível carregar o guia agora. Tente novamente em instantes ou abra a página Ajuda do console.</p>",
      normalized
    );
  }

  async function readCached(cache, key, now) {
    const entry = cache.get(key);
    if (!entry) return null;
    return entry.expiresAt > now() ? entry : { ...entry, expired: true };
  }

  async function resolveGuideContent(options) {
    const {
      locale,
      slug,
      config,
      cache,
      fetchFn,
      readLocal,
      now = Date.now,
      forceRefresh = false
    } = options || {};

    const requested = normalizeLocale(locale);
    const safeSlug = sanitizeSlug(slug);
    if (!safeSlug) {
      return { html: unavailablePage(requested), source: "unavailable", locale: requested };
    }
    const candidates = localeCandidates(requested);
    const cacheStore = cache || createCache();

    if (config.source !== "github") {
      for (const candidate of candidates) {
        try {
          const html = await readLocal(candidate, safeSlug);
          if (html) return { html, source: "local", locale: candidate };
        } catch {
          // try the next locale
        }
      }
      return { html: unavailablePage(requested), source: "unavailable", locale: requested };
    }

    let stale = null;
    for (const candidate of candidates) {
      const key = cacheKey(config, candidate, safeSlug);
      const cached = await readCached(cacheStore, key, now);
      if (cached && !cached.expired && !forceRefresh) {
        return { html: cached.html, source: "github-cache", locale: candidate };
      }
      if (cached) stale = { ...cached, locale: candidate };
      if (cached && forceRefresh && now() - (cached.forcedAt || 0) < config.forceRefreshCooldownMs) {
        return { html: cached.html, source: "github-cooldown", locale: candidate };
      }

      const headers = { "User-Agent": "dontstop-console-guide", Accept: "text/plain" };
      if (cached && cached.etag) headers["If-None-Match"] = cached.etag;

      try {
        const response = await fetchFn(rawGuideUrl(config, candidate, safeSlug), {
          headers,
          signal: config.timeoutMs ? AbortSignal.timeout(config.timeoutMs) : undefined
        });
        if (response.status === 304 && cached) {
          cacheStore.set(key, { ...cached, expiresAt: now() + config.ttlMs, forcedAt: now() });
          return { html: cached.html, source: "github-cache", locale: candidate };
        }
        if (response.status === 404) continue;
        if (!response.ok) break;
        const markdown = await response.text();
        if (!markdown.trim()) continue;
        const html = renderRemoteGuide(markdown, { locale: candidate, config });
        cacheStore.set(key, {
          html,
          etag: response.headers.get("etag") || "",
          expiresAt: now() + config.ttlMs,
          forcedAt: now()
        });
        return { html, source: "github", locale: candidate };
      } catch {
        break;
      }
    }

    if (stale) {
      return { html: stale.html, source: "github-stale", locale: stale.locale };
    }
    for (const candidate of candidates) {
      try {
        const html = await readLocal(candidate, safeSlug);
        if (html) return { html, source: "local-fallback", locale: candidate };
      } catch {
        // keep trying
      }
    }
    return { html: unavailablePage(requested), source: "unavailable", locale: requested };
  }

  return {
    SUPPORTED_LOCALES,
    normalizeLocale,
    localeCandidates,
    sanitizeSlug,
    readConfig,
    rawGuideUrl,
    cacheKey,
    createCache,
    renderRemoteGuide,
    remoteImageUrl,
    remoteGuideLink,
    unavailablePage,
    resolveGuideContent,
    escapeAttr
  };
});
