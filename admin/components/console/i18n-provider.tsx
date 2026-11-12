"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useSyncExternalStore,
  type ReactNode
} from "react";
import {
  detectLocale,
  guideUrl,
  LOCALE_CODES,
  plural,
  resolveLocale,
  STORAGE_KEY,
  translate,
  type Locale
} from "@/lib/i18n";

interface I18nValue {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: string, values?: Record<string, string | number>) => string;
  tp: (key: string, count: number, values?: Record<string, string | number>) => string;
  guideUrl: (slug: string, anchor?: string) => string;
}

const I18nContext = createContext<I18nValue | null>(null);
const listeners = new Set<() => void>();

function readLocale(): Locale {
  let requested: string | null = null;
  try {
    requested = new URLSearchParams(window.location.search).get("lang");
  } catch {
    requested = null;
  }
  if (requested) return resolveLocale(requested);
  let stored: string | null = null;
  try {
    stored = window.localStorage.getItem(STORAGE_KEY);
  } catch {
    stored = null;
  }
  if (stored) return resolveLocale(stored);
  const cookie = readLocaleCookie();
  if (cookie) return resolveLocale(cookie);
  return detectLocale(navigator.languages);
}

function readLocaleCookie(): string | null {
  try {
    const prefix = `${STORAGE_KEY}=`;
    const entry = document.cookie.split("; ").find(part => part.startsWith(prefix));
    return entry ? decodeURIComponent(entry.slice(prefix.length)) : null;
  } catch {
    return null;
  }
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  const onStorage = (event: StorageEvent) => {
    if (event.key === STORAGE_KEY) listener();
  };
  window.addEventListener("storage", onStorage);
  return () => {
    listeners.delete(listener);
    window.removeEventListener("storage", onStorage);
  };
}

function notify(): void {
  listeners.forEach(listener => listener());
}

export function I18nProvider({ children, initialLocale = "en" }: { children: ReactNode; initialLocale?: Locale }) {
  const locale = useSyncExternalStore(subscribe, readLocale, () => initialLocale);

  useEffect(() => {
    document.documentElement.lang = locale;
    // Persist the resolved locale so the next server render starts in the same
    // language (avoids an English flash before hydration).
    try {
      document.cookie = `${STORAGE_KEY}=${encodeURIComponent(locale)}; path=/; max-age=31536000; samesite=lax`;
    } catch {
      // Cookies can be disabled; the client still renders the stored locale.
    }
  }, [locale]);

  const setLocale = useCallback((next: Locale) => {
    const resolved = LOCALE_CODES.includes(next) ? next : "en";
    try {
      window.localStorage.setItem(STORAGE_KEY, resolved);
      document.cookie = `${STORAGE_KEY}=${encodeURIComponent(resolved)}; path=/; max-age=31536000; samesite=lax`;
    } catch {
      // Storage can be disabled; the current session still updates.
    }
    document.documentElement.lang = resolved;
    notify();
  }, []);

  const value = useMemo<I18nValue>(() => ({
    locale,
    setLocale,
    t: (key, values) => translate(locale, key, values),
    tp: (key, count, values) => plural(locale, key, count, values),
    guideUrl: (slug, anchor) => guideUrl(locale, slug, anchor)
  }), [locale, setLocale]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nValue {
  const value = useContext(I18nContext);
  if (!value) throw new Error("useI18n must be used inside I18nProvider");
  return value;
}
