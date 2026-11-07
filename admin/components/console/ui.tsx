"use client";

import { useState } from "react";
import { Check, Copy } from "lucide-react";
import { useI18n } from "./i18n-provider";

export function CopyButton({ value, label }: { value: string; label?: string }) {
  const { t } = useI18n();
  const [copied, setCopied] = useState(false);
  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
    } catch {
      // Clipboard can be unavailable; the value remains selectable.
    }
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1600);
  }
  return (
    <button className="btn" type="button" onClick={copy} aria-label={`${t("common.copy")}: ${label ?? value}`}>
      {copied ? <Check aria-hidden="true" size={14} /> : <Copy aria-hidden="true" size={14} />}
      {copied ? t("common.copied") : t("common.copy")}
    </button>
  );
}

export function Toggle({
  checked,
  onChange,
  label
}: {
  checked: boolean;
  onChange: (value: boolean) => void;
  label: string;
}) {
  return (
    <button
      className={`switch ${checked ? "" : "off"}`}
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      onClick={() => onChange(!checked)}
    >
      <span />
    </button>
  );
}

export function Tabs({
  items,
  value,
  onChange
}: {
  items: Array<{ id: string; label: string }>;
  value: string;
  onChange: (id: string) => void;
}) {
  return (
    <div className="tabs" role="tablist">
      {items.map(item => (
        <button
          className="tab"
          key={item.id}
          type="button"
          role="tab"
          aria-selected={value === item.id}
          onClick={() => onChange(item.id)}
        >
          {item.label}
        </button>
      ))}
    </div>
  );
}

export function BarChart({ values, alt = false }: { values: number[]; alt?: boolean }) {
  return (
    <div className="chart-bars" aria-hidden="true">
      {values.map((value, index) => (
        <span className={`bar ${alt ? "alt" : ""}`} key={index} style={{ height: `${value}%` }} />
      ))}
    </div>
  );
}

export function Progress({ value, tone = "ok" }: { value: number; tone?: "ok" | "warn" | "bad" }) {
  return (
    <span className="progress" aria-label={`${value}%`}>
      <span className={`progress-value ${tone}`} style={{ width: `${Math.min(100, value)}%` }} />
    </span>
  );
}
