"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { LOCALES, type Locale } from "@/lib/i18n";
import { useI18n } from "@/components/console/i18n-provider";

const DEFAULT_BASE_URLS: Record<string, string> = {
  openai: "https://api.openai.com/v1",
  anthropic: "https://api.anthropic.com",
  agentrouter: "https://agentrouter.org"
};

export function OnboardingWizard() {
  const { locale, setLocale, t } = useI18n();
  const router = useRouter();
  const [step, setStep] = useState(1);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [providerName, setProviderName] = useState("");
  const [kind, setKind] = useState("openai");
  const [modelId, setModelId] = useState("");
  const [baseUrl, setBaseUrl] = useState(DEFAULT_BASE_URLS.openai);
  const [credential, setCredential] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function createAccount(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");
    try {
      const response = await fetch("/api/auth/setup", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, email, password })
      });
      if (!response.ok) {
        setError(t("onboarding.error"));
        return;
      }
      setStep(3);
    } catch {
      setError(t("onboarding.error"));
    } finally {
      setLoading(false);
    }
  }

  async function connectProvider(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");
    try {
      const response = await fetch("/api/admin/providers", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: providerName,
          kind,
          origin: "native",
          model_id: modelId,
          model_name: modelId,
          base_url: baseUrl,
          credential
        })
      });
      if (!response.ok) {
        setError(t("onboarding.error"));
        return;
      }
      router.replace("/dashboard");
      router.refresh();
    } catch {
      setError(t("onboarding.error"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="card login-card wizard-card">
      <ol className="wizard-steps" aria-label={t("onboarding.progress")}>
        <li className={step === 1 ? "active" : ""}>1</li>
        <li className={step === 2 ? "active" : ""}>2</li>
        <li className={step === 3 ? "active" : ""}>3</li>
      </ol>

      {step === 1 ? (
        <>
          <h1>{t("onboarding.languageTitle")}</h1>
          <p className="sub">{t("onboarding.languageBody")}</p>
          <fieldset className="wizard-options">
            <legend>{t("common.language")}</legend>
            {LOCALES.map(item => (
              <label key={item.code} className="wizard-option">
                <input
                  type="radio"
                  name="locale"
                  value={item.code}
                  checked={locale === item.code}
                  onChange={() => setLocale(item.code as Locale)}
                />
                <span>{item.name}</span>
              </label>
            ))}
          </fieldset>
          <button className="btn primary" type="button" onClick={() => setStep(2)}>
            {t("onboarding.continue")}
          </button>
        </>
      ) : null}

      {step === 2 ? (
        <form onSubmit={createAccount}>
          <h1>{t("onboarding.accountTitle")}</h1>
          <p className="sub">{t("onboarding.accountBody")}</p>
          <label className="field">
            <span>{t("onboarding.name")}</span>
            <input className="input" value={name} onChange={event => setName(event.target.value)} required />
          </label>
          <label className="field">
            <span>{t("onboarding.email")}</span>
            <input className="input" type="email" value={email} onChange={event => setEmail(event.target.value)} required />
          </label>
          <label className="field">
            <span>{t("onboarding.password")}</span>
            <input className="input" type="password" minLength={8} value={password} onChange={event => setPassword(event.target.value)} required />
          </label>
          {error ? <p className="small" style={{ color: "var(--danger)" }}>{error}</p> : null}
          <button className="btn primary" type="submit" disabled={loading}>
            {loading ? t("common.loading") : t("onboarding.createAccount")}
          </button>
        </form>
      ) : null}

      {step === 3 ? (
        <form onSubmit={connectProvider}>
          <h1>{t("onboarding.providerTitle")}</h1>
          <p className="sub">{t("onboarding.providerBody")}</p>
          <label className="field">
            <span>{t("onboarding.providerName")}</span>
            <input className="input" value={providerName} onChange={event => setProviderName(event.target.value)} required />
          </label>
          <label className="field">
            <span>{t("onboarding.providerKind")}</span>
            <select
              className="input"
              value={kind}
              onChange={event => {
                const next = event.target.value;
                setKind(next);
                setBaseUrl(DEFAULT_BASE_URLS[next] ?? "");
              }}
            >
              <option value="openai">OpenAI-compatible</option>
              <option value="anthropic">Anthropic</option>
              <option value="agentrouter">AgentRouter</option>
            </select>
          </label>
          <label className="field">
            <span>{t("onboarding.modelId")}</span>
            <input className="input" value={modelId} onChange={event => setModelId(event.target.value)} required />
          </label>
          <label className="field">
            <span>{t("onboarding.baseUrl")}</span>
            <input className="input" value={baseUrl} onChange={event => setBaseUrl(event.target.value)} required />
          </label>
          <label className="field">
            <span>{t("onboarding.apiKey")}</span>
            <input className="input" type="password" value={credential} onChange={event => setCredential(event.target.value)} required />
          </label>
          {error ? <p className="small" style={{ color: "var(--danger)" }}>{error}</p> : null}
          <button className="btn primary" type="submit" disabled={loading}>
            {loading ? t("common.loading") : t("onboarding.connectProvider")}
          </button>
        </form>
      ) : null}
    </section>
  );
}
