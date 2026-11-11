export const SUPPORTED_LOCALES: string[];

export interface GuideConfig {
  source: "local" | "github";
  repo: string;
  ref: string;
  baseUrl: string;
  ttlMs: number;
  timeoutMs: number;
  forceRefreshCooldownMs: number;
}

export interface GuideResult {
  html: string;
  source: string;
  locale: string;
}

export function normalizeLocale(input?: string | null): string;
export function localeCandidates(locale: string): string[];
export function sanitizeSlug(input?: string | null): string | null;
export function readConfig(env?: Record<string, string | undefined>): GuideConfig;
export function rawGuideUrl(config: GuideConfig, locale: string, slug: string): string;
export function cacheKey(config: GuideConfig, locale: string, slug: string): string;
export function createCache(): Map<string, { html: string; etag: string; expiresAt: number; forcedAt: number }>;
export function renderRemoteGuide(markdown: string, options: { locale: string; config: GuideConfig }): string;
export function remoteImageUrl(config: GuideConfig, source: string): string;
export function remoteGuideLink(locale: string, href: string): string;
export function unavailablePage(locale: string): string;
export function resolveGuideContent(options: {
  locale: string;
  slug: string;
  config: GuideConfig;
  cache?: Map<string, { html: string; etag: string; expiresAt: number; forcedAt: number }>;
  fetchFn: typeof fetch;
  readLocal: (locale: string, slug: string) => Promise<string | null>;
  now?: () => number;
  forceRefresh?: boolean;
}): Promise<GuideResult>;
