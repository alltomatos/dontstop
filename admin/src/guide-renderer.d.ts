export const LOCALE_NAMES: Record<string, string>;
export const STYLES: string;
export function escapeHtml(value: unknown): string;
export function rewriteHref(href: string): string;
export function renderMarkdown(markdown: string, options?: { resolveImage?: (src: string) => string; resolveLink?: (href: string) => string }): string;
export function renderPage(title: string, bodyHtml: string, locale: string): string;
export function extractTitle(markdown: string, fallback: string): string;
