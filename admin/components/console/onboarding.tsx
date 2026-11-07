"use client";

import { useState, useSyncExternalStore } from "react";
import { LOCALES, STORAGE_KEY, type Locale } from "@/lib/i18n";
import { useI18n } from "./i18n-provider";

const ONBOARDING_KEY = "dontstop.onboarding";
const ONBOARDING_EVENT = "dontstop:onboarding";

function subscribe(listener: () => void): () => void {
  window.addEventListener(ONBOARDING_EVENT, listener);
  window.addEventListener("storage", listener);
  return () => {
    window.removeEventListener(ONBOARDING_EVENT, listener);
    window.removeEventListener("storage", listener);
  };
}

function shouldShow(): boolean {
  try {
    return !window.localStorage.getItem(ONBOARDING_KEY) && !window.localStorage.getItem(STORAGE_KEY);
  } catch {
    return false;
  }
}

export function Onboarding() {
  const show = useSyncExternalStore(subscribe, shouldShow, () => false);
  const { locale, setLocale, t } = useI18n();
  const [selected, setSelected] = useState<Locale | null>(null);
  const effective = selected ?? locale;

  if (!show) return null;

  const detected = LOCALES.find(item => item.code === locale)?.name ?? locale;

  function complete() {
    setLocale(effective);
    try {
      window.localStorage.setItem(ONBOARDING_KEY, "done");
    } catch {
      // Storage can be disabled; the current session still continues.
    }
    window.dispatchEvent(new Event(ONBOARDING_EVENT));
  }

  return (
    <div className="modal-backdrop" role="presentation">
      <div className="modal" role="dialog" aria-modal="true" aria-labelledby="onboarding-title">
        <div className="modal-head">
          <div>
            <h2 id="onboarding-title">{t("onboarding.title")}</h2>
            <p className="muted">{t("onboarding.subtitle")}</p>
          </div>
        </div>
        <p className="small muted">{t("onboarding.detected", { locale: detected })}</p>
        <fieldset className="language-grid" style={{ marginTop: 16 }}>
          <legend className="sr-only">{t("onboarding.languageLegend")}</legend>
          {LOCALES.map(item => (
            <label className="language-card" key={item.code}>
              <input
                type="radio"
                name="onboarding-locale"
                value={item.code}
                checked={effective === item.code}
                onChange={() => setSelected(item.code)}
              />
              <span>
                <strong>{item.name}</strong>
                <span>{item.code}</span>
              </span>
            </label>
          ))}
        </fieldset>
        <div className="modal-foot">
          <button className="btn primary" type="button" onClick={complete}>{t("onboarding.confirm")}</button>
        </div>
      </div>
    </div>
  );
}
