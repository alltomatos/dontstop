"use client";

import Link from "next/link";
import type { ReactNode } from "react";
import { useI18n } from "./i18n-provider";

export function Breadcrumbs({ items }: { items: Array<{ label: string; href?: string }> }) {
  const { t } = useI18n();
  return (
    <nav className="crumbs" aria-label={t("common.breadcrumb")}>
      {items.map((item, index) => (
        <span className="crumb-item" key={`${item.label}-${index}`}>
          {index > 0 ? <span aria-hidden="true">/</span> : null}
          {item.href ? <Link href={item.href}>{item.label}</Link> : <span>{item.label}</span>}
        </span>
      ))}
    </nav>
  );
}

export function PageHeader({
  title,
  subtitle,
  actions
}: {
  title: string;
  subtitle?: string;
  actions?: ReactNode;
}) {
  return (
    <div className="page-header">
      <div>
        <h1>{title}</h1>
        {subtitle ? <p className="sub">{subtitle}</p> : null}
      </div>
      {actions ? <div className="actions">{actions}</div> : null}
    </div>
  );
}

export function EmptyState({
  title,
  body,
  action
}: {
  title: string;
  body: string;
  action?: ReactNode;
}) {
  return (
    <div className="empty-state">
      <div className="empty-icon" aria-hidden="true">+</div>
      <h2>{title}</h2>
      <p>{body}</p>
      {action}
    </div>
  );
}
