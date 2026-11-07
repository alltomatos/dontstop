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
  guideUrl: (slug: string) => string;
}

const I18nContext = createContext<I18nValue | null>(null);
const listeners = new Set<() => void>();

function readLocale(): Locale {
  let stored: string | null = null;
  try {
    stored = window.localStorage.getItem(STORAGE_KEY);
  } catch {
    stored = null;
  }
  return stored ? resolveLocale(stored) : detectLocale(navigator.languages);
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

export function I18nProvider({ children }: { children: ReactNode }) {
  const locale = useSyncExternalStore(subscribe, readLocale, () => "en" as Locale);

  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);

  const setLocale = useCallback((next: Locale) => {
    const resolved = LOCALE_CODES.includes(next) ? next : "en";
    try {
      window.localStorage.setItem(STORAGE_KEY, resolved);
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
    guideUrl: slug => guideUrl(locale, slug)
  }), [locale, setLocale]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nValue {
  const value = useContext(I18nContext);
  if (!value) throw new Error("useI18n must be used inside I18nProvider");
  return value;
}
