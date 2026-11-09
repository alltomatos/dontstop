"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { ArrowLeft, ArrowRight, Check, KeyRound, Search, ShieldCheck, SkipForward } from "lucide-react";
import { LOCALES, type Locale } from "@/lib/i18n";
import { useI18n } from "@/components/console/i18n-provider";
import { PROVIDERS } from "@/lib/providers";
import {
  PROVIDER_SETUP,
  type ProviderAuthType,
  type ProviderSetup
} from "@/lib/provider-setup";

type AuthFilter = "all" | ProviderAuthType;

function authLabel(type: ProviderAuthType, t: (key: string) => string): string {
  return t(`onboarding.auth.${type}`);
}

export function OnboardingWizard() {
  const { locale, setLocale, t } = useI18n();
  const router = useRouter();
  const [step, setStep] = useState(1);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [query, setQuery] = useState("");
  const [authFilter, setAuthFilter] = useState<AuthFilter>("all");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [providerName, setProviderName] = useState("");
  const [kind, setKind] = useState("openai");
  const [modelId, setModelId] = useState("");
  const [baseUrl, setBaseUrl] = useState("");
  const [credential, setCredential] = useState("");
  const [error, setError] = useState("");
  const [setupConflict, setSetupConflict] = useState(false);
  const [accountCreated, setAccountCreated] = useState(false);
  const [loading, setLoading] = useState(false);

  const catalog = useMemo(
    () => Object.values(PROVIDERS).filter(provider => PROVIDER_SETUP[provider.id]),
    []
  );
  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return catalog.filter(provider => {
      const setup = PROVIDER_SETUP[provider.id];
      const matchesQuery = !needle || provider.name.toLowerCase().includes(needle) || provider.id.includes(needle);
      const matchesAuth = authFilter === "all" || setup.authType === authFilter;
      return matchesQuery && matchesAuth;
    });
  }, [authFilter, catalog, query]);

  const selectedProvider = selectedId ? PROVIDERS[selectedId] : null;
  const selectedSetup: ProviderSetup | null = selectedId ? PROVIDER_SETUP[selectedId] : null;

  function selectProvider(id: string) {
    const setup = PROVIDER_SETUP[id];
    const provider = PROVIDERS[id];
    if (!setup || !provider) return;
    setSelectedId(id);
    setProviderName(provider.id);
    setKind(setup.kind ?? "openai");
    setBaseUrl(setup.defaultBaseUrl ?? "");
    setModelId(setup.defaultModel ?? "");
    setCredential("");
    setError("");
  }

  function skipProvider() {
    router.replace("/dashboard");
    router.refresh();
  }

  async function createAccount(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (accountCreated) {
      setStep(3);
      return;
    }
    setLoading(true);
    setError("");
    setSetupConflict(false);
    try {
      const response = await fetch("/api/auth/setup", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, email, password })
      });
      if (response.status === 409) {
        setSetupConflict(true);
        return;
      }
      if (!response.ok) {
        setError(t("onboarding.error"));
        return;
      }
      setAccountCreated(true);
      setStep(3);
    } catch {
      setError(t("onboarding.error"));
    } finally {
      setLoading(false);
    }
  }

  async function connectProvider(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedProvider || !selectedSetup?.supported) return;
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
      skipProvider();
    } catch {
      setError(t("onboarding.error"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="card onboarding-card">
      <header className="onboarding-head">
        <div className="onboarding-brand">
          <span className="brand-mark" aria-hidden="true"><ShieldCheck size={18} /></span>
          <div>
            <span className="eyebrow">dontstop</span>
            <span className="onboarding-step-label">{t("onboarding.step", { current: step, total: 3 })}</span>
          </div>
        </div>
        {step === 3 ? (
          <button className="btn ghost" type="button" onClick={skipProvider}>
            <SkipForward aria-hidden="true" size={16} />
            {t("onboarding.skip")}
          </button>
        ) : null}
      </header>

      <ol className="wizard-steps" aria-label={t("onboarding.progress")}>
        {[1, 2, 3].map(item => (
          <li
            key={item}
            className={step >= item ? "active" : ""}
            aria-current={step === item ? "step" : undefined}
          >
            {step > item ? <Check aria-hidden="true" size={14} /> : item}
          </li>
        ))}
      </ol>

      {step === 1 ? (
        <div className="onboarding-step">
          <div>
            <h1>{t("onboarding.languageTitle")}</h1>
            <p className="sub">{t("onboarding.languageBody")}</p>
          </div>
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
          <div className="onboarding-actions">
            <button className="btn primary" type="button" onClick={() => setStep(2)}>
              {t("onboarding.continue")}
              <ArrowRight aria-hidden="true" size={16} />
            </button>
          </div>
        </div>
      ) : null}

      {step === 2 ? (
        <form className="onboarding-step" onSubmit={createAccount}>
          <div>
            <h1>{t("onboarding.accountTitle")}</h1>
            <p className="sub">{t("onboarding.accountBody")}</p>
          </div>
          <label className="field">
            <span>{t("onboarding.name")}</span>
            <input className="input" autoComplete="name" value={name} onChange={event => setName(event.target.value)} required />
          </label>
          <label className="field">
            <span>{t("onboarding.email")}</span>
            <input className="input" type="email" autoComplete="email" value={email} onChange={event => setEmail(event.target.value)} required />
          </label>
          <label className="field">
            <span>{t("onboarding.password")}</span>
            <input className="input" type="password" autoComplete="new-password" minLength={8} value={password} onChange={event => setPassword(event.target.value)} required />
          </label>
          {setupConflict ? (
            <p className="small" role="alert" style={{ color: "var(--danger)" }}>
              {t("onboarding.setupConflict")}{" "}
              <a href="/login">{t("onboarding.goToSignIn")}</a>
            </p>
          ) : error ? (
            <p className="small" role="alert" style={{ color: "var(--danger)" }}>{error}</p>
          ) : null}
          <div className="onboarding-actions">
            <button className="btn" type="button" onClick={() => setStep(1)}>
              <ArrowLeft aria-hidden="true" size={16} />
              {t("onboarding.back")}
            </button>
            <button className="btn primary" type="submit" disabled={loading}>
              {loading ? t("common.loading") : t("onboarding.createAccount")}
              <ArrowRight aria-hidden="true" size={16} />
            </button>
          </div>
        </form>
      ) : null}

      {step === 3 ? (
        <div className="onboarding-step">
          <div className="onboarding-section-head">
            <div>
              <h1>{t("onboarding.providerTitle")}</h1>
              <p className="sub">{t("onboarding.providerBody")}</p>
            </div>
            <span className="chip">{filtered.length}</span>
          </div>

          <div className="onboarding-provider-layout">
            <div className="onboarding-catalog">
              <div className="onboarding-toolbar">
                <label className="search-inline">
                  <Search aria-hidden="true" size={16} />
                  <input
                    className="input"
                    type="search"
                    value={query}
                    onChange={event => setQuery(event.target.value)}
                    placeholder={t("onboarding.searchProviders")}
                    aria-label={t("onboarding.searchProviders")}
                  />
                </label>
              </div>
              <div className="filter-chips" role="group" aria-label={t("onboarding.authFilter")}>
                {(["all", "api_key", "oauth", "none"] as const).map(filter => (
                  <button
                    key={filter}
                    className={`filter-chip ${authFilter === filter ? "active" : ""}`}
                    type="button"
                    onClick={() => setAuthFilter(filter)}
                  >
                    {filter === "all" ? t("onboarding.allAuth") : authLabel(filter, t)}
                  </button>
                ))}
              </div>
              <div className="provider-catalog" role="group" aria-label={t("onboarding.providerCatalog")}>
                {filtered.map(provider => {
                  const setup = PROVIDER_SETUP[provider.id];
                  const selected = selectedId === provider.id;
                  return (
                    <button
                      key={provider.id}
                      type="button"
                      className={`provider-option ${selected ? "selected" : ""}`}
                      aria-pressed={selected}
                      onClick={() => selectProvider(provider.id)}
                    >
                      <span className="provider-icon">{provider.icon}</span>
                      <span className="provider-option-copy">
                        <strong>{provider.name}</strong>
                        <small>{authLabel(setup.authType, t)}</small>
                      </span>
                      {selected ? <Check className="provider-option-check" aria-hidden="true" size={16} /> : null}
                    </button>
                  );
                })}
                {filtered.length === 0 ? (
                  <p className="muted">{t("onboarding.noProviders")}</p>
                ) : null}
              </div>
            </div>

            <aside className="provider-config">
              {!selectedProvider || !selectedSetup ? (
                <div className="provider-config-empty">
                  <KeyRound aria-hidden="true" size={22} />
                  <h2>{t("onboarding.selectProvider")}</h2>
                  <p className="muted">{t("onboarding.selectProviderBody")}</p>
                </div>
              ) : (
                <>
                  <div className="provider-config-head">
                    <span className="provider-icon">{selectedProvider.icon}</span>
                    <div>
                      <h2>{selectedProvider.name}</h2>
                      <span className="chip">{authLabel(selectedSetup.authType, t)}</span>
                    </div>
                  </div>

                  {selectedSetup.authType === "oauth" ? (
                    <div className="notice">
                      <ShieldCheck aria-hidden="true" size={18} />
                      <div>
                        <strong>{t("onboarding.oauthTitle")}</strong>
                        <p>{t("onboarding.oauthBody")}</p>
                      </div>
                    </div>
                  ) : null}

                  {!selectedSetup.supported ? (
                    <div className="notice">
                      <KeyRound aria-hidden="true" size={18} />
                      <div>
                        <strong>{t("onboarding.unsupportedTitle")}</strong>
                        <p>{t("onboarding.unsupportedBody")}</p>
                      </div>
                    </div>
                  ) : (
                    <form className="provider-form" onSubmit={connectProvider}>
                      <label className="field">
                        <span>{t("onboarding.providerName")}</span>
                        <input className="input" value={providerName} onChange={event => setProviderName(event.target.value)} required />
                      </label>
                      {selectedSetup.requiredFields.includes("base_url") ? (
                        <label className="field">
                          <span>{t("onboarding.baseUrl")}</span>
                          <input className="input" type="url" value={baseUrl} onChange={event => setBaseUrl(event.target.value)} required />
                        </label>
                      ) : null}
                      {selectedSetup.requiredFields.includes("model_id") ? (
                        <label className="field">
                          <span>{t("onboarding.modelId")}</span>
                          <input className="input" value={modelId} onChange={event => setModelId(event.target.value)} required />
                        </label>
                      ) : null}
                      {selectedSetup.requiredFields.includes("credential") ? (
                        <label className="field">
                          <span>{t("onboarding.apiKey")}</span>
                          <input className="input" type="password" autoComplete="off" value={credential} onChange={event => setCredential(event.target.value)} required />
                        </label>
                      ) : null}
                      {error ? <p className="small" role="alert" style={{ color: "var(--danger)" }}>{error}</p> : null}
                      <button className="btn primary" type="submit" disabled={loading}>
                        {loading ? t("common.loading") : t("onboarding.connectProvider")}
                        <ArrowRight aria-hidden="true" size={16} />
                      </button>
                    </form>
                  )}
                </>
              )}
            </aside>
          </div>

          <div className="onboarding-actions">
            <button className="btn" type="button" onClick={() => setStep(2)}>
              <ArrowLeft aria-hidden="true" size={16} />
              {t("onboarding.back")}
            </button>
            <div className="onboarding-actions-right">
              <button className="btn ghost" type="button" onClick={skipProvider}>
                <SkipForward aria-hidden="true" size={16} />
                {t("onboarding.skip")}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </section>
  );
}
