"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useI18n } from "@/components/console/i18n-provider";

export function LoginForm({ next }: { next: string }) {
  const { t } = useI18n();
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
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
        body: JSON.stringify({ email, password })
      });
      if (!response.ok) {
        setError(t("auth.error"));
        return;
      }
      router.replace(next);
      router.refresh();
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
        <span>{t("auth.email")}</span>
        <input
          className="input"
          type="email"
          autoComplete="email"
          value={email}
          onChange={event => setEmail(event.target.value)}
          required
        />
      </label>
      <label className="field">
        <span>{t("auth.password")}</span>
        <input
          className="input"
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={event => setPassword(event.target.value)}
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
