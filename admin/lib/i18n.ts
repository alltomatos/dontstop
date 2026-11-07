import en from "../locales/en.json";
import ptBR from "../locales/pt-BR.json";
import ptPT from "../locales/pt-PT.json";
import es from "../locales/es.json";
import fr from "../locales/fr.json";
import de from "../locales/de.json";

export const LOCALE_CODES = ["en", "pt-PT", "pt-BR", "es", "fr", "de"] as const;
export type Locale = (typeof LOCALE_CODES)[number];

export const LOCALES: Array<{ code: Locale; name: string }> = [
  { code: "en", name: "English" },
  { code: "pt-PT", name: "Português (Portugal)" },
  { code: "pt-BR", name: "Português (Brasil)" },
  { code: "es", name: "Español" },
  { code: "fr", name: "Français" },
  { code: "de", name: "Deutsch" }
];

const catalogs: Record<Locale, Record<string, string>> = {
  en,
  "pt-PT": ptPT,
  "pt-BR": ptBR,
  es,
  fr,
  de
};

export const STORAGE_KEY = "dontstop.locale";

export function resolveLocale(input?: string | null): Locale {
  if (!input) return "en";
  const raw = input.trim().replace(/_/g, "-").toLowerCase();
  if (raw === "pt" || raw === "pt-br") return "pt-BR";
  if (raw === "pt-pt") return "pt-PT";
  if (raw === "en" || raw.startsWith("en-")) return "en";
  if (raw === "es" || raw.startsWith("es-")) return "es";
  if (raw === "fr" || raw.startsWith("fr-")) return "fr";
  if (raw === "de" || raw.startsWith("de-")) return "de";
  return "en";
}

export function detectLocale(languages?: readonly string[]): Locale {
  for (const language of languages ?? []) {
    const resolved = resolveLocale(language);
    if (resolved !== "en") return resolved;
  }
  return "en";
}

export function translate(locale: Locale, key: string, values?: Record<string, string | number>): string {
  const value = catalogs[locale]?.[key] ?? catalogs.en[key] ?? key;
  if (!values) return value;
  return value.replace(/\{(\w+)\}/g, (match, name) =>
    Object.prototype.hasOwnProperty.call(values, name) ? String(values[name]) : match
  );
}

export function plural(
  locale: Locale,
  key: string,
  count: number,
  values?: Record<string, string | number>
): string {
  const catalog = catalogs[locale] ?? catalogs.en;
  const zeroKey = `${key}.zero`;
  const category = count === 0 && catalog[zeroKey] ? "zero" : new Intl.PluralRules(locale).select(count);
  const value = catalog[`${key}.${category}`] ?? catalog[`${key}.other`] ?? key;
  return value.replace(/\{(\w+)\}/g, (match, name) =>
    Object.prototype.hasOwnProperty.call(values ?? {}, name)
      ? String((values ?? {})[name])
      : name === "count"
        ? String(count)
        : match
  );
}

export function guideUrl(locale: Locale, slug: string): string {
  return `https://github.com/alltomatos/dontstop/blob/main/docs/user-guide/${locale}/${slug}.md`;
}

export function formatNumber(locale: Locale, value: number, options?: Intl.NumberFormatOptions): string {
  return new Intl.NumberFormat(locale, options).format(value);
}
