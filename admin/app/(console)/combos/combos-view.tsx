"use client";

import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { LoaderCircle, Pencil, Plus, Power, Sparkles, Trash2, X } from "lucide-react";
import { Breadcrumbs, EmptyState, PageHeader } from "@/components/console/page";
import { useI18n } from "@/components/console/i18n-provider";
import type { Combo, ComboAIDraft, ComboCatalogModel, ComboMember, ComboStrategy } from "@/lib/combos";

type FormState = {
  id?: string;
  revision?: number;
  slug: string;
  name: string;
  description: string;
  strategy: ComboStrategy;
  failover_enabled: boolean;
  rpm_limit: string;
  tpm_limit: string;
  max_attempts: number;
  members: ComboMember[];
};

const EMPTY_FORM: FormState = {
  slug: "",
  name: "",
  description: "",
  strategy: "priority_round_robin",
  failover_enabled: true,
  rpm_limit: "",
  tpm_limit: "",
  max_attempts: 5,
  members: []
};

function formFromCombo(combo: Combo): FormState {
  return {
    id: combo.id,
    revision: combo.revision,
    slug: combo.slug,
    name: combo.name,
    description: combo.description,
    strategy: combo.strategy,
    failover_enabled: combo.failover_enabled,
    rpm_limit: combo.rpm_limit ? String(combo.rpm_limit) : "",
    tpm_limit: combo.tpm_limit ? String(combo.tpm_limit) : "",
    max_attempts: combo.max_attempts,
    members: (combo.members ?? []).map(member => ({ ...member }))
  };
}

function statusKey(status: string) {
  return status === "active" ? "combos.statusActive" : status === "draft" ? "combos.statusDraft" : "combos.statusInactive";
}

function strategyKey(strategy: string) {
  if (strategy === "priority") return "combos.strategyPriority";
  if (strategy === "round_robin") return "combos.strategyRoundRobin";
  return "combos.strategyPriorityRoundRobin";
}

export function CombosView({
  combos,
  models,
  error = false
}: {
  combos: Combo[];
  models: ComboCatalogModel[];
  error?: boolean;
}) {
  const { t } = useI18n();
  const router = useRouter();
  const [query, setQuery] = useState("");
  const [form, setForm] = useState<FormState | null>(null);
  const [detail, setDetail] = useState<Combo | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Combo | null>(null);
  const [aiOpen, setAiOpen] = useState(false);
  const [aiBusy, setAiBusy] = useState(false);
  const [aiError, setAiError] = useState("");
  const [aiPrompt, setAiPrompt] = useState("");
  const [aiMessages, setAiMessages] = useState<Array<{ role: string; content: string }>>([]);
  const [aiDraft, setAiDraft] = useState<ComboAIDraft | null>(null);
  const [assistantKey, setAssistantKey] = useState("");
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState("");

  const visible = useMemo(() => {
    const term = query.trim().toLowerCase();
    if (!term) return combos;
    return combos.filter(combo =>
      [combo.name, combo.slug, combo.strategy].join(" ").toLowerCase().includes(term)
    );
  }, [combos, query]);

  const usableModels = models.filter(model => model.usable);
  const assistantModels = models;

  function openNew() {
    setForm({ ...EMPTY_FORM, members: [] });
    setFormError("");
  }

  async function openEdit(id: string) {
    const combo = await loadCombo(id);
    if (combo) {
      setForm(formFromCombo(combo));
      setFormError("");
    }
  }

  async function openDetail(id: string) {
    const combo = await loadCombo(id);
    if (combo) setDetail(combo);
  }

  async function loadCombo(id: string): Promise<Combo | null> {
    const response = await fetch(`/api/admin/combos/${encodeURIComponent(id)}`, { cache: "no-store" });
    if (!response.ok) return null;
    return await response.json() as Combo;
  }

  function addMember() {
    if (!form) return;
    const model = usableModels[0];
    if (!model) return;
    if (form.members.some(member => member.provider_id === model.provider_id && member.model_id === model.model_id)) return;
    const next: ComboMember = {
      provider_id: model.provider_id,
      provider_name: model.provider_name,
      model_id: model.model_id,
      model_name: model.display_name || model.model_id,
      priority: 0,
      position: form.members.length,
      weight: 100,
      enabled: true,
      constraints: {}
    };
    setForm({ ...form, members: [...form.members, next] });
  }

  function updateMember(index: number, patch: Partial<ComboMember>) {
    if (!form) return;
    setForm({
      ...form,
      members: form.members.map((member, current) => current === index ? { ...member, ...patch } : member)
    });
  }

  function moveMember(index: number, direction: -1 | 1) {
    if (!form) return;
    const target = index + direction;
    if (target < 0 || target >= form.members.length) return;
    const members = [...form.members];
    [members[index], members[target]] = [members[target], members[index]];
    setForm({ ...form, members: members.map((member, position) => ({ ...member, position })) });
  }

  function validate(): boolean {
    if (!form) return false;
    if (!form.name.trim() || !/^[a-z][a-z0-9-]{1,62}$/.test(form.slug)) {
      setFormError(t("combos.validationError"));
      return false;
    }
    if (form.members.length === 0) {
      setFormError(t("combos.noMembers"));
      return false;
    }
    setFormError("");
    return true;
  }

  async function save(active: boolean) {
    if (!form || !validate() || busy) return;
    setBusy(true);
    try {
      const payload = {
        slug: form.slug,
        name: form.name,
        description: form.description,
        strategy: form.strategy,
        failover_enabled: form.failover_enabled,
        rpm_limit: form.rpm_limit ? Number(form.rpm_limit) : null,
        tpm_limit: form.tpm_limit ? Number(form.tpm_limit) : null,
        max_attempts: form.max_attempts,
        members: form.members.map((member, index) => ({
          provider_id: member.provider_id,
          model_id: member.model_id,
          priority: Number(member.priority) || 0,
          position: index,
          weight: Number(member.weight) || 1,
          enabled: member.enabled !== false,
          constraints: member.constraints ?? {}
        }))
      };
      const response = await fetch(
        form.id ? `/api/admin/combos/${encodeURIComponent(form.id)}?revision=${form.revision}` : "/api/admin/combos",
        {
          method: form.id ? "PATCH" : "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload)
        }
      );
      if (!response.ok) {
        setFormError(t("combos.validationError"));
        return;
      }
      const saved = await response.json() as Combo;
      if (active) {
        const activation = await fetch(`/api/admin/combos/${encodeURIComponent(saved.id)}/activate`, { method: "POST" });
        if (!activation.ok) {
          setFormError(t("combos.noMembers"));
          return;
        }
      }
      setForm(null);
      router.refresh();
    } finally {
      setBusy(false);
    }
  }

  async function setStatus(combo: Combo, active: boolean) {
    if (busy) return;
    setBusy(true);
    try {
      await fetch(`/api/admin/combos/${encodeURIComponent(combo.id)}/${active ? "activate" : "deactivate"}`, { method: "POST" });
      router.refresh();
      if (detail?.id === combo.id) setDetail(null);
    } finally {
      setBusy(false);
    }
  }

  async function remove() {
    if (!deleteTarget || busy) return;
    setBusy(true);
    try {
      await fetch(`/api/admin/combos/${encodeURIComponent(deleteTarget.id)}`, { method: "DELETE" });
      setDeleteTarget(null);
      setDetail(null);
      router.refresh();
    } finally {
      setBusy(false);
    }
  }

  async function generateDraft() {
    if (aiBusy || !aiPrompt.trim()) return;
    setAiBusy(true);
    setAiError("");
    const nextMessages = [...aiMessages, { role: "user", content: aiPrompt.trim() }];
    setAiMessages(nextMessages);
    try {
      const [providerId, modelId] = assistantKey.split("/");
      const response = await fetch("/api/admin/combos/ai/draft", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          description: aiPrompt.trim(),
          provider_id: providerId || undefined,
          model_id: modelId || undefined,
          messages: nextMessages
        })
      });
      if (!response.ok) {
        setAiError(t("combos.aiError"));
        return;
      }
      const draft = await response.json() as ComboAIDraft;
      setAiDraft(draft);
      setAiMessages([...nextMessages, { role: "assistant", content: draft.message }]);
      setAiPrompt("");
    } catch {
      setAiError(t("combos.aiError"));
    } finally {
      setAiBusy(false);
    }
  }

  function applyDraft() {
    if (!aiDraft) return;
    setForm({
      slug: aiDraft.name.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "").slice(0, 60) || "ai-combo",
      name: aiDraft.name,
      description: aiDraft.description,
      strategy: aiDraft.strategy,
      failover_enabled: aiDraft.failover_enabled,
      rpm_limit: "",
      tpm_limit: "",
      max_attempts: 5,
      members: aiDraft.members.map((member, index) => ({ ...member, position: index }))
    });
    setAiOpen(false);
    setAiDraft(null);
    setFormError("");
  }

  return (
    <>
      <Breadcrumbs items={[{ label: t("nav.combos") }]} />
      <PageHeader
        title={t("combos.title")}
        subtitle={t("combos.subtitle")}
        actions={(
          <>
            <button className="btn" type="button" onClick={() => setAiOpen(true)}>
              <Sparkles aria-hidden="true" size={16} /> {t("combos.ai")}
            </button>
            <button className="btn primary" type="button" onClick={openNew}>
              <Plus aria-hidden="true" size={16} /> {t("combos.new")}
            </button>
          </>
        )}
      />

      {error ? <p className="small" role="status" style={{ color: "var(--danger)" }}>{t("common.gatewayError")}</p> : null}

      <div className="card" style={{ marginBottom: 16 }}>
        <div className="toolbar">
          <input className="input" value={query} onChange={event => setQuery(event.target.value)} placeholder={t("combos.search")} aria-label={t("combos.search")} />
          <span className="muted">{visible.length} {t("combos.items")}</span>
        </div>
      </div>

      {visible.length === 0 ? (
        <EmptyState
          title={t("combos.emptyTitle")}
          body={t("combos.emptyBody")}
          action={<button className="btn primary" type="button" onClick={openNew}>{t("combos.new")}</button>}
        />
      ) : (
        <div className="card">
          <table className="table">
            <thead>
              <tr>
                <th>{t("combos.name")}</th>
                <th>{t("common.status")}</th>
                <th>{t("combos.strategy")}</th>
                <th>{t("combos.members")}</th>
                <th className="right">{t("combos.actions")}</th>
              </tr>
            </thead>
            <tbody>
              {visible.map(combo => (
                <tr key={combo.id}>
                  <td>
                    <strong>{combo.name}</strong>
                    <div className="small mono">combo/{combo.slug}</div>
                    <div className="small muted">{combo.source === "ai" ? t("combos.sourceAI") : ""}</div>
                  </td>
                  <td><span className={`chip ${combo.status === "active" ? "ok" : combo.status === "draft" ? "warn" : ""}`}>{t(statusKey(combo.status))}</span></td>
                  <td>{t(strategyKey(combo.strategy))}</td>
                  <td>{combo.member_count ?? combo.members?.length ?? 0} <span className="muted">/ {combo.usable_members ?? 0} {t("combos.usable")}</span></td>
                  <td className="right">
                    <div className="actions" style={{ justifyContent: "flex-end" }}>
                      <button className="btn ghost" type="button" onClick={() => openDetail(combo.id)}>{t("combos.detail")}</button>
                      <button className="btn ghost" type="button" onClick={() => openEdit(combo.id)} aria-label={t("common.edit")}><Pencil size={15} /></button>
                      <button className="btn ghost" type="button" onClick={() => setStatus(combo, combo.status !== "active")} aria-label={combo.status === "active" ? t("combos.deactivate") : t("combos.activate")}><Power size={15} /></button>
                      <button className="btn danger" type="button" onClick={() => setDeleteTarget(combo)} aria-label={t("combos.delete")}><Trash2 size={15} /></button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {form ? (
        <div className="modal-backdrop" role="presentation" onClick={event => { if (event.target === event.currentTarget) setForm(null); }}>
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="combo-title" style={{ width: "min(900px, 100%)", maxHeight: "92vh", overflow: "auto" }}>
            <div className="modal-head">
              <h2 id="combo-title">{form.id ? t("common.edit") : t("combos.new")}</h2>
              <button className="modal-close" type="button" aria-label={t("common.close")} onClick={() => setForm(null)}><X size={16} /></button>
            </div>
            <div style={{ display: "grid", gap: 12, gridTemplateColumns: "1fr 1fr" }}>
              <label className="field"><span>{t("combos.name")}</span><input className="input" value={form.name} onChange={event => setForm({ ...form, name: event.target.value })} /></label>
              <label className="field"><span>{t("combos.slug")}</span><input className="input mono" value={form.slug} disabled={Boolean(form.id)} onChange={event => setForm({ ...form, slug: event.target.value.toLowerCase() })} /></label>
              <label className="field" style={{ gridColumn: "1 / -1" }}><span>{t("combos.description")}</span><input className="input" value={form.description} onChange={event => setForm({ ...form, description: event.target.value })} /></label>
              <label className="field"><span>{t("combos.strategy")}</span>
                <select className="input" value={form.strategy} onChange={event => setForm({ ...form, strategy: event.target.value as ComboStrategy })}>
                  <option value="priority">{t("combos.strategyPriority")}</option>
                  <option value="round_robin">{t("combos.strategyRoundRobin")}</option>
                  <option value="priority_round_robin">{t("combos.strategyPriorityRoundRobin")}</option>
                </select>
              </label>
              <label className="field"><span>{t("combos.maxAttempts")}</span><input className="input" type="number" min={1} max={8} value={form.max_attempts} onChange={event => setForm({ ...form, max_attempts: Number(event.target.value) })} /></label>
              <label className="field"><span>{t("combos.rpm")}</span><input className="input" type="number" min={1} value={form.rpm_limit} onChange={event => setForm({ ...form, rpm_limit: event.target.value })} /></label>
              <label className="field"><span>{t("combos.tpm")}</span><input className="input" type="number" min={1} value={form.tpm_limit} onChange={event => setForm({ ...form, tpm_limit: event.target.value })} /></label>
              <label className="field" style={{ gridColumn: "1 / -1", display: "flex", gap: 8, alignItems: "center" }}>
                <input type="checkbox" checked={form.failover_enabled} onChange={event => setForm({ ...form, failover_enabled: event.target.checked })} />
                <span>{t("combos.failover")}</span>
              </label>
            </div>

            <div className="row" style={{ marginTop: 8 }}>
              <strong>{t("combos.members")}</strong>
              <button className="btn" type="button" onClick={addMember} disabled={!usableModels.length}>{t("combos.addMember")}</button>
            </div>
            <div style={{ display: "flex", flexDirection: "column", gap: 8, marginTop: 8 }}>
              {form.members.map((member, index) => (
                <div className="card" key={`${member.provider_id}/${member.model_id}/${index}`} style={{ padding: 10 }}>
                  <div className="row">
                    <div>
                      <strong>{member.model_name || member.model_id}</strong>
                      <div className="small muted">{member.provider_name || member.provider_id}</div>
                    </div>
                    <div className="actions">
                      <button className="btn ghost" type="button" onClick={() => moveMember(index, -1)} disabled={index === 0} aria-label={t("combos.moveUp")}>↑</button>
                      <button className="btn ghost" type="button" onClick={() => moveMember(index, 1)} disabled={index === form.members.length - 1} aria-label={t("combos.moveDown")}>↓</button>
                      <button className="btn danger" type="button" onClick={() => setForm({ ...form, members: form.members.filter((_, current) => current !== index) })} aria-label={t("combos.remove")}><Trash2 size={14} /></button>
                    </div>
                  </div>
                  <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr 1fr", gap: 8, marginTop: 8 }}>
                    <label className="field"><span>{t("combos.priority")}</span><input className="input" type="number" min={0} max={100} value={member.priority} onChange={event => updateMember(index, { priority: Number(event.target.value) })} /></label>
                    <label className="field"><span>{t("combos.weight")}</span><input className="input" type="number" min={1} max={100} value={member.weight} onChange={event => updateMember(index, { weight: Number(event.target.value) })} /></label>
                    <label className="field"><span>{t("combos.model")}</span>
                      <select className="input" value={`${member.provider_id}/${member.model_id}`} onChange={event => {
                        const [providerId, modelId] = event.target.value.split("/");
                        const model = models.find(item => item.provider_id === providerId && item.model_id === modelId);
                        if (model) updateMember(index, { provider_id: providerId, model_id: modelId, provider_name: model.provider_name, model_name: model.display_name || model.model_id });
                      }}>
                        {models.map(model => <option key={`${model.provider_id}/${model.model_id}`} value={`${model.provider_id}/${model.model_id}`}>{model.provider_name} · {model.display_name || model.model_id}{model.usable ? "" : " (unavailable)"}</option>)}
                      </select>
                    </label>
                  </div>
                </div>
              ))}
              {form.members.length === 0 ? <p className="small muted">{t("combos.noMembers")}</p> : null}
            </div>
            {formError ? <p className="small" role="alert" style={{ color: "var(--danger)" }}>{formError}</p> : null}
            <div className="modal-foot">
              <button className="btn" type="button" onClick={() => setForm(null)}>{t("common.cancel")}</button>
              <button className="btn" type="button" disabled={busy} onClick={() => save(false)}>{busy ? <LoaderCircle className="spin" size={15} /> : null}{t("combos.saveDraft")}</button>
              <button className="btn primary" type="button" disabled={busy} onClick={() => save(true)}>{t("combos.saveActivate")}</button>
            </div>
          </div>
        </div>
      ) : null}

      {detail ? (
        <div className="modal-backdrop" role="presentation" onClick={event => { if (event.target === event.currentTarget) setDetail(null); }}>
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="combo-detail-title" style={{ width: "min(800px, 100%)" }}>
            <div className="modal-head">
              <div><h2 id="combo-detail-title">{detail.name}</h2><div className="small mono">combo/{detail.slug} · rev. {detail.revision}</div></div>
              <button className="modal-close" type="button" aria-label={t("common.close")} onClick={() => setDetail(null)}><X size={16} /></button>
            </div>
            <p className="muted">{detail.description}</p>
            <div className="actions" style={{ marginBottom: 12 }}>
              <button className="btn" type="button" onClick={() => { setForm(formFromCombo(detail)); setDetail(null); }}>{t("common.edit")}</button>
              <button className="btn" type="button" onClick={() => setStatus(detail, detail.status !== "active")}>{detail.status === "active" ? t("combos.deactivate") : t("combos.activate")}</button>
              <button className="btn danger" type="button" onClick={() => setDeleteTarget(detail)}>{t("combos.delete")}</button>
            </div>
            <table className="table">
              <thead><tr><th>#</th><th>{t("combos.model")}</th><th>{t("combos.priority")}</th><th>{t("combos.weight")}</th><th>{t("common.status")}</th></tr></thead>
              <tbody>{(detail.members ?? []).map((member, index) => (
                <tr key={member.id ?? `${member.provider_id}/${member.model_id}/${index}`}>
                  <td>{index + 1}</td>
                  <td><strong>{member.model_name || member.model_id}</strong><div className="small muted">{member.provider_name}</div></td>
                  <td>{member.priority}</td><td>{member.weight}</td>
                  <td>{member.usable ? <span className="chip ok">{t("combos.usable")}</span> : <span className="chip warn">{member.reason || t("combos.unavailable")}</span>}</td>
                </tr>
              ))}</tbody>
            </table>
          </div>
        </div>
      ) : null}

      {deleteTarget ? (
        <div className="modal-backdrop" role="presentation" onClick={event => { if (event.target === event.currentTarget) setDeleteTarget(null); }}>
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="combo-delete-title" style={{ width: "min(520px, 100%)" }}>
            <div className="modal-head"><h2 id="combo-delete-title">{t("combos.deleteTitle")}</h2><button className="modal-close" type="button" onClick={() => setDeleteTarget(null)}><X size={16} /></button></div>
            <p>{t("combos.deleteBody")} <strong>{deleteTarget.name}</strong></p>
            <div className="modal-foot"><button className="btn" type="button" onClick={() => setDeleteTarget(null)}>{t("common.cancel")}</button><button className="btn danger" type="button" disabled={busy} onClick={remove}>{t("combos.delete")}</button></div>
          </div>
        </div>
      ) : null}

      {aiOpen ? (
        <div className="modal-backdrop" role="presentation" onClick={event => { if (event.target === event.currentTarget) setAiOpen(false); }}>
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="combo-ai-title" style={{ width: "min(860px, 100%)", maxHeight: "92vh", overflow: "auto" }}>
            <div className="modal-head"><div><h2 id="combo-ai-title">{t("combos.aiTitle")}</h2><div className="small muted">{t("combos.aiDescription")}</div></div><button className="modal-close" type="button" onClick={() => setAiOpen(false)} aria-label={t("common.close")}><X size={16} /></button></div>
            <label className="field" style={{ marginBottom: 10 }}><span>{t("combos.aiAssistant")}</span>
              <select className="input" value={assistantKey} onChange={event => setAssistantKey(event.target.value)}>
                <option value="">{t("combos.aiDefaultAssistant")}</option>
                {assistantModels.map(model => <option key={`${model.provider_id}/${model.model_id}`} value={`${model.provider_id}/${model.model_id}`}>{model.provider_name} · {model.display_name || model.model_id}</option>)}
              </select>
            </label>
            <div style={{ display: "grid", gap: 8, marginBottom: 10 }}>
              {aiMessages.map((message, index) => <div className="card" key={index} style={{ padding: 10 }}><strong>{message.role}</strong><div>{message.content}</div></div>)}
            </div>
            <label className="field"><span>{t("combos.aiPlaceholder")}</span><textarea className="input" rows={3} value={aiPrompt} onChange={event => setAiPrompt(event.target.value)} /></label>
            {aiError ? <p className="small" role="alert" style={{ color: "var(--danger)" }}>{aiError}</p> : null}
            {aiDraft ? (
              <div className="card" style={{ marginTop: 10 }}>
                <div className="row"><strong>{aiDraft.name}</strong><span className="chip warn">{t("combos.draftReview")}</span></div>
                <p className="small muted">{aiDraft.message}</p>
                <table className="table"><thead><tr><th>{t("combos.model")}</th><th>{t("combos.priority")}</th><th>{t("combos.weight")}</th></tr></thead><tbody>{aiDraft.members.map((member, index) => <tr key={index}><td>{member.model_name || member.model_id}<div className="small muted">{member.provider_name || member.provider_id}</div></td><td>{member.priority}</td><td>{member.weight}</td></tr>)}</tbody></table>
                {aiDraft.warnings.length ? <ul className="small">{aiDraft.warnings.map((warning, index) => <li key={index}>{warning}</li>)}</ul> : null}
                <button className="btn primary" type="button" onClick={applyDraft}>{t("combos.aiApply")}</button>
              </div>
            ) : null}
            <div className="modal-foot">
              <button className="btn" type="button" onClick={() => setAiOpen(false)}>{t("common.cancel")}</button>
              <button className="btn primary" type="button" disabled={aiBusy || !aiPrompt.trim()} onClick={generateDraft}>{aiBusy ? <LoaderCircle className="spin" size={15} /> : <Sparkles size={15} />} {aiBusy ? t("combos.aiGenerating") : t("combos.aiGenerate")}</button>
            </div>
          </div>
        </div>
      ) : null}
    </>
  );
}
