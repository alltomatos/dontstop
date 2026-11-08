"use client";

import { useState } from "react";
import { useI18n } from "@/components/console/i18n-provider";

export function LoginForm({ next }: { next: string }) {
  const { t } = useI18n();
  const [key, setKey] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");
    try {
      const response = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ key })
      });
      if (!response.ok) {
        setError(t("auth.error"));
        return;
      }
      window.location.href = next;
    } catch {
      setError(t("auth.error"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <form className="card login-card" onSubmit={submit}>
      <h1>{t("auth.title")}</h1>
      <p className="sub">{t("auth.subtitle")}</p>
      <label className="field">
        <span>{t("auth.apiKey")}</span>
        <input
          className="input"
          type="password"
          autoComplete="off"
          value={key}
          onChange={event => setKey(event.target.value)}
          required
        />
      </label>
      {error ? <p className="small" style={{ color: "var(--danger)" }}>{error}</p> : null}
      <button className="btn primary" type="submit" disabled={loading}>
        {loading ? t("common.loading") : t("auth.signIn")}
      </button>
    </form>
  );
}
