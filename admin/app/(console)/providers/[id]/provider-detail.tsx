"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { ArrowLeft, Download, ExternalLink, LoaderCircle, Pencil, Play, Plus, Power, RefreshCw, Settings2, Trash2, X } from "lucide-react";
import { Breadcrumbs } from "@/components/console/page";
import { OriginChips, StatusChip } from "@/components/console/origin-chips";
import { useI18n } from "@/components/console/i18n-provider";
import type { Provider, ProviderAccount } from "@/lib/providers";
import type { ConfiguredModel, ModelTestRun } from "@/lib/admin-api";
import { PROVIDER_SETUP_GUIDE } from "@/lib/catalog";
import { AccountProxyControl } from "./provider-proxy";
import { configurePath } from "@/src/catalog";
import { eligibleAccounts, pickTestAccount, rememberTestAccount } from "@/src/account-selection";
import {
  INPUT_MODALITIES,
  OUTPUT_MODALITIES,
  bulkPayload,
  filterModels,
  isFree,
  metadataKnown,
  priceKnown,
  selectAllFiltered,
  selectedModels as pickSelectedModels,
  toggleFilterCapability,
  toggleSelection
} from "@/src/model-filters";

function capability(model: ConfiguredModel, key: string): boolean {
  return model.capabilities?.[key] === true;
}

function hasModality(model: ConfiguredModel, direction: "input" | "output", modality: string): boolean {
  const explicit = model.capabilities?.[`${modality}_${direction}`];
  if (typeof explicit === "boolean") return explicit;
  if (direction === "input" && modality === "image") return capability(model, "vision");
  if (direction === "input" && modality === "audio") return capability(model, "audio");
  return false;
}

function contextWindow(model: ConfiguredModel): number {
  const value = Number(model.capabilities?.context_window);
  return Number.isFinite(value) ? value : 0;
}

function formatTokens(value: number): string {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(value % 1_000_000 === 0 ? 0 : 1)}M`;
  if (value >= 1_000) return `${Math.round(value / 1_000)}k`;
  return String(value);
}

function maskCredential(value: string): string {
  const clean = value.trim();
  if (!clean) return "••••";
  const suffix = clean.slice(-4);
  return `${clean.startsWith("sk-") ? "sk-..." : "••••"}${suffix}`;
}

export function ProviderDetail({
  provider,
  models = [],
  latestRun = null
}: {
  provider: Provider;
  models?: ConfiguredModel[];
  latestRun?: ModelTestRun | null;
}) {
  const { t, tp, guideUrl } = useI18n();
  const router = useRouter();
  const [accounts, setAccounts] = useState<ProviderAccount[]>(provider.accounts);
  const [serverAccounts, setServerAccounts] = useState(provider.accounts);
  if (provider.accounts !== serverAccounts) {
    setServerAccounts(provider.accounts);
    setAccounts(provider.accounts);
  }
  const [modelList, setModelList] = useState<ConfiguredModel[]>(models);
  const [modelBusy, setModelBusy] = useState("");
  const [modelMessage, setModelMessage] = useState("");
  const [modelsError, setModelsError] = useState("");
  const [modelResults, setModelResults] = useState<Record<string, string>>({});
  const [syncing, setSyncing] = useState(false);
  const [testRun, setTestRun] = useState<ModelTestRun | null>(latestRun);
  const runActive = testRun?.status === "queued" || testRun?.status === "running";
  const [modelQueryFilter, setModelQueryFilter] = useState("");
  const [priceFilter, setPriceFilter] = useState<"all" | "free" | "paid" | "unknown">("all");
  const [statusFilter, setStatusFilter] = useState<"all" | "active" | "inactive">("all");
  const [capFilters, setCapFilters] = useState<string[]>([]);
  const [selectedTestAccount, setSelectedTestAccount] = useState("");
  const [testingAccountId, setTestingAccountId] = useState("");
  const [accountMessages, setAccountMessages] = useState<Record<string, string>>({});
  const [inputFilters, setInputFilters] = useState<string[]>([]);
  const [outputFilters, setOutputFilters] = useState<string[]>([]);
  const [selectedModelIDs, setSelectedModelIDs] = useState<Set<string>>(new Set());
  const [bulkBusy, setBulkBusy] = useState(false);
  const [bulkMessage, setBulkMessage] = useState("");

  const testableAccounts = useMemo(() => eligibleAccounts(accounts), [accounts]);
  const selectedTestAccountInfo = useMemo(
    () => pickTestAccount(accounts, selectedTestAccount),
    [accounts, selectedTestAccount]
  );

  /* eslint-disable react-hooks/set-state-in-effect */
  useEffect(() => {
    try {
      const raw = window.localStorage.getItem("dontstop.modelTestAccount.v1");
      if (!raw) return;
      const stored = JSON.parse(raw) as Record<string, string>;
      const value = stored[provider.id];
      if (typeof value === "string") setSelectedTestAccount(value);
    } catch {
      // ignore malformed persisted selection
    }
  }, [provider.id]);
  /* eslint-enable react-hooks/set-state-in-effect */

  function chooseTestAccount(accountID: string) {
    setSelectedTestAccount(accountID);
    try {
      const raw = window.localStorage.getItem("dontstop.modelTestAccount.v1");
      const stored = raw ? JSON.parse(raw) as Record<string, string> : {};
      window.localStorage.setItem(
        "dontstop.modelTestAccount.v1",
        JSON.stringify(rememberTestAccount(stored, provider.id, accountID))
      );
    } catch {
      // storage may be unavailable
    }
  }

  function capabilityLabel(value: string): string {
    switch (value) {
      case "text_input": return t("providers.inputShort", { modality: t("providers.modality_text") });
      case "image_input": return t("providers.capVision");
      case "audio_input": return t("providers.capAudio");
      case "video_input": return t("providers.modality_video");
      case "tools": return t("providers.capTools");
      case "reasoning": return t("providers.capReasoning");
      case "embeddings": return t("providers.capEmbeddings");
      case "stream": return t("providers.capStream");
      default: return value;
    }
  }

  const visibleModels = useMemo(
    () => filterModels(modelList, {
      query: modelQueryFilter,
      price: priceFilter,
      status: statusFilter,
      inputModalities: inputFilters,
      outputModalities: outputFilters,
      capabilities: capFilters
    }),
    [modelList, modelQueryFilter, priceFilter, statusFilter, inputFilters, outputFilters, capFilters]
  );
  const allFilteredSelected = visibleModels.length > 0 && visibleModels.every(model => selectedModelIDs.has(model.model_id));
  const someFilteredSelected = visibleModels.some(model => selectedModelIDs.has(model.model_id));
  const hasModelFilters = modelQueryFilter.trim() !== "" || priceFilter !== "all" || statusFilter !== "all" || inputFilters.length > 0 || outputFilters.length > 0 || capFilters.length > 0;

  function clearModelFilters() {
    setModelQueryFilter("");
    setPriceFilter("all");
    setStatusFilter("all");
    setInputFilters([]);
    setOutputFilters([]);
    setCapFilters([]);
  }

  function toggleCapabilityFilter(capability: string) {
    setCapFilters(current => toggleFilterCapability(current, capability));
  }

  /* eslint-disable react-hooks/set-state-in-effect */
  useEffect(() => {
    try {
      const raw = window.localStorage.getItem("dontstop.modelFilters.v1");
      if (!raw) return;
      const stored = JSON.parse(raw) as Record<string, unknown>;
      if (typeof stored.query === "string") setModelQueryFilter(stored.query);
      if (stored.price === "free" || stored.price === "paid" || stored.price === "unknown") setPriceFilter(stored.price);
      if (stored.status === "active" || stored.status === "inactive") setStatusFilter(stored.status);
      if (Array.isArray(stored.input)) setInputFilters(stored.input.filter((item): item is string => typeof item === "string"));
      if (Array.isArray(stored.output)) setOutputFilters(stored.output.filter((item): item is string => typeof item === "string"));
      if (Array.isArray(stored.capabilities)) setCapFilters(stored.capabilities.filter((item): item is string => typeof item === "string"));
    } catch {
      // ignore malformed persisted filters
    }
  }, []);
  /* eslint-enable react-hooks/set-state-in-effect */

  useEffect(() => {
    try {
      window.localStorage.setItem("dontstop.modelFilters.v1", JSON.stringify({
        query: modelQueryFilter,
        price: priceFilter,
        status: statusFilter,
        input: inputFilters,
        output: outputFilters,
        capabilities: capFilters
      }));
    } catch {
      // storage may be unavailable
    }
  }, [modelQueryFilter, priceFilter, statusFilter, inputFilters, outputFilters, capFilters]);

  function toggleModality(setter: (update: (current: string[]) => string[]) => void, modality: string) {
    setter(current => current.includes(modality) ? current.filter(item => item !== modality) : [...current, modality]);
  }

  function toggleSelectAllFiltered() {
    setSelectedModelIDs(current => {
      if (allFilteredSelected) {
        const next = new Set(current);
        for (const model of visibleModels) next.delete(model.model_id);
        return next;
      }
      return selectAllFiltered(visibleModels, current);
    });
  }

  function toggleModelSelection(modelID: string) {
    setSelectedModelIDs(current => toggleSelection(current, modelID));
  }

  async function bulkApply(active: boolean) {
    const chosen = pickSelectedModels(modelList, selectedModelIDs);
    if (chosen.length === 0 || bulkBusy) return;
    const actionLabel = active ? t("providers.bulkActionActivate") : t("providers.bulkActionDeactivate");
    if (!window.confirm(t("providers.bulkConfirm", { count: chosen.length, action: actionLabel }))) return;
    setBulkBusy(true);
    setBulkMessage("");
    setModelsError("");
    try {
      const response = await fetch(`/api/admin/providers/${encodeURIComponent(provider.id)}/models/bulk`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(bulkPayload(chosen.map(model => model.model_id), active))
      });
      const body = await response.json().catch(() => null) as {
        updated?: number;
        results?: Array<{ model_id: string; ok: boolean; error?: string }>;
      } | null;
      if (!response.ok || !Array.isArray(body?.results)) {
        setModelsError(t("providers.bulkFailed"));
        return;
      }
      const okIDs = new Set(body.results.filter(result => result.ok).map(result => result.model_id));
      setModelList(current => current.map(model => (
        okIDs.has(model.model_id) ? { ...model, active } : model
      )));
      const failed = body.results.filter(result => !result.ok);
      if (failed.length > 0) {
        setModelResults(current => {
          const next = { ...current };
          for (const item of failed) next[item.model_id] = item.error || t("providers.modelUpdateFailed");
          return next;
        });
      }
      setBulkMessage(t("providers.bulkDone", { ok: okIDs.size, failed: failed.length }));
      setSelectedModelIDs(new Set());
      router.refresh();
    } catch {
      setModelsError(t("providers.bulkFailed"));
    } finally {
      setBulkBusy(false);
    }
  }
  const [testing, setTesting] = useState(false);
  const [testMessage, setTestMessage] = useState("");
  const [modelsOpen, setModelsOpen] = useState(false);
  const [modelsLoading, setModelsLoading] = useState(false);
  const [importing, setImporting] = useState(false);
  const [remoteModels, setRemoteModels] = useState<Array<{ id: string; name?: string }>>([]);
  const [selectedModels, setSelectedModels] = useState<Set<string>>(new Set());
  const [modelQuery, setModelQuery] = useState("");
  const [modelError, setModelError] = useState("");
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [credential, setCredential] = useState("");
  const [accountError, setAccountError] = useState("");
  const [removingId, setRemovingId] = useState("");
  const [editingAccount, setEditingAccount] = useState<ProviderAccount | null>(null);
  const [editName, setEditName] = useState("");
  const [editBusy, setEditBusy] = useState(false);
  const [editError, setEditError] = useState("");

  async function testAccount(account: ProviderAccount) {
    if (!account.id || testingAccountId) return;
    setTestingAccountId(account.id);
    setAccountMessages(current => ({ ...current, [account.id!]: "" }));
    try {
      const response = await fetch(
        `/api/admin/providers/${encodeURIComponent(provider.id)}/accounts/${encodeURIComponent(account.id)}/test`,
        { method: "POST" }
      );
      const body = await response.json().catch(() => null) as {
        account_name?: string; ok?: boolean; model_count?: number; latency_ms?: number; error?: string;
      } | null;
      const name = body?.account_name || account.name;
      const message = response.ok && body?.ok
        ? t("providers.accountTestSuccess", { account: name, count: body.model_count ?? 0, ms: body.latency_ms ?? 0 })
        : t("providers.accountTestFailed", { account: name, error: body?.error || t("providers.testFailed") });
      setAccountMessages(current => ({ ...current, [account.id!]: message }));
    } catch {
      setAccountMessages(current => ({ ...current, [account.id!]: t("providers.accountTestFailed", { account: account.name, error: t("providers.testFailed") }) }));
    } finally {
      setTestingAccountId("");
    }
  }

  async function testProvider() {
    setTesting(true);
    setTestMessage("");
    try {
      const response = await fetch(`/api/admin/providers/${encodeURIComponent(provider.id)}/test`, { method: "POST" });
      const body = await response.json().catch(() => null) as { ok?: boolean; model_count?: number; latency_ms?: number; error?: string } | null;
      if (response.ok && body?.ok) {
        setTestMessage(t("providers.testSuccess", { count: body.model_count ?? 0, ms: body.latency_ms ?? 0 }));
      } else {
        setTestMessage(t("providers.testFailed"));
      }
    } catch {
      setTestMessage(t("providers.testFailed"));
    } finally {
      setTesting(false);
    }
  }

  async function openModels() {
    setModelsOpen(true);
    setModelsLoading(true);
    setModelError("");
    setModelQuery("");
    setSelectedModels(new Set());
    try {
      const response = await fetch(`/api/admin/providers/${encodeURIComponent(provider.id)}/models`);
      const body = await response.json().catch(() => null) as { data?: Array<{ id: string; name?: string }> } | null;
      if (!response.ok || !Array.isArray(body?.data)) {
        setModelError(t("providers.modelsUnavailable"));
        return;
      }
      setRemoteModels(body.data);
    } catch {
      setModelError(t("providers.modelsUnavailable"));
    } finally {
      setModelsLoading(false);
    }
  }

  async function importModels() {
    setImporting(true);
    setModelError("");
    try {
      const response = await fetch(`/api/admin/providers/${encodeURIComponent(provider.id)}/models/import`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ models: Array.from(selectedModels) })
      });
      if (!response.ok) {
        setModelError(t("providers.modelsImportFailed"));
        return;
      }
      const body = await response.json().catch(() => null) as { imported?: number } | null;
      setModelsOpen(false);
      setTestMessage(tp("providers.modelsImported", body?.imported ?? selectedModels.size));
      router.refresh();
    } catch {
      setModelError(t("providers.modelsImportFailed"));
    } finally {
      setImporting(false);
    }
  }

  const filteredModels = remoteModels.filter(model => !modelQuery.trim() || model.id.toLowerCase().includes(modelQuery.trim().toLowerCase()));
  const allSelected = filteredModels.length > 0 && filteredModels.every(model => selectedModels.has(model.id));

  function toggleModel(id: string) {
    setSelectedModels(current => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  }

  function toggleAllModels() {
    setSelectedModels(current => {
      const next = new Set(current);
      if (allSelected) filteredModels.forEach(model => next.delete(model.id));
      else filteredModels.forEach(model => next.add(model.id));
      return next;
    });
  }

  function openEditAccount(account: ProviderAccount) {
    if (!account.id) return;
    setEditingAccount(account);
    setEditName(account.name);
    setEditError("");
  }

  async function saveAccountName(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!editingAccount?.id || editBusy) return;
    const nextName = editName.trim();
    if (!nextName) return;
    setEditBusy(true);
    setEditError("");
    try {
      const response = await fetch(
        `/api/admin/providers/${encodeURIComponent(provider.id)}/accounts/${encodeURIComponent(editingAccount.id)}`,
        {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ name: nextName })
        }
      );
      if (response.status === 409) {
        setEditError(t("providers.accountNameConflict"));
        return;
      }
      if (!response.ok) {
        setEditError(t("providers.accountUpdateFailed"));
        return;
      }
      setAccounts(current => current.map(account => (
        account.id === editingAccount.id ? { ...account, name: nextName } : account
      )));
      setEditingAccount(null);
      router.refresh();
    } catch {
      setEditError(t("providers.accountUpdateFailed"));
    } finally {
      setEditBusy(false);
    }
  }

  async function removeAccount(account: ProviderAccount) {
    if (!account.id || removingId) return;
    if (!window.confirm(t("providers.removeAccountConfirm", { name: account.name }))) return;
    setAccountError("");
    setRemovingId(account.id);
    try {
      const response = await fetch(
        `/api/admin/providers/${encodeURIComponent(provider.id)}/accounts/${encodeURIComponent(account.id)}`,
        { method: "DELETE" }
      );
      if (!response.ok) {
        setAccountError(t("providers.accountRemoveFailed"));
        return;
      }
      const body = await response.json().catch(() => null) as { provider_deleted?: boolean } | null;
      setAccounts(current => current.filter(item => item.id !== account.id));
      if (body?.provider_deleted) {
        router.push("/providers");
        router.refresh();
        return;
      }
      router.refresh();
    } catch {
      setAccountError(t("providers.accountRemoveFailed"));
    } finally {
      setRemovingId("");
    }
  }

  async function addAccount(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!name.trim() || !credential.trim()) return;
    setAccountError("");
    const fallback = () => {
      setAccounts(current => [
        ...current,
        { name: name.trim(), meta: maskCredential(credential), status: "ok", statusKey: "common.healthy" }
      ]);
      setName("");
      setCredential("");
      setOpen(false);
    };
    try {
      const response = await fetch(`/api/admin/providers/${encodeURIComponent(provider.id)}/accounts`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: name.trim(), credential })
      });
      if (response.status === 503) {
        fallback();
        return;
      }
      if (!response.ok) {
        setAccountError(t("common.error"));
        return;
      }
      const account = await response.json() as { id: string; name: string; status: string; base_url?: string };
      setAccounts(current => [
        ...current,
        {
          name: account.name,
          meta: account.base_url || account.id,
          status: account.status === "cooldown" ? "warn" : "ok",
          statusKey: account.status === "cooldown" ? "common.cooldown" : "common.healthy"
        }
      ]);
      setName("");
      setCredential("");
      setOpen(false);
    } catch {
      fallback();
    }
  }

  function applyRunResults(run: ModelTestRun) {
    if (!run.items?.length) return;
    setModelResults(current => {
      const next = { ...current };
      for (const item of run.items ?? []) {
        if (item.status === "ok") {
          next[item.model_id] = t("providers.modelTestOK", { ms: item.latency_ms });
        } else if (item.status === "failed") {
          next[item.model_id] = item.error || t("providers.modelTestFailed");
        }
      }
      return next;
    });
  }

  useEffect(() => {
    if (!testRun || (testRun.status !== "queued" && testRun.status !== "running")) return;
    let cancelled = false;
    const poll = async () => {
      const response = await fetch(
        `/api/admin/providers/${encodeURIComponent(provider.id)}/models/test-runs/${encodeURIComponent(testRun.id)}`,
        { cache: "no-store" }
      ).catch(() => null);
      if (!response?.ok || cancelled) return;
      const body = await response.json().catch(() => null) as ModelTestRun | null;
      if (!body || cancelled) return;
      setTestRun(body);
      applyRunResults(body);
    };
    const timer = setInterval(poll, 1500);
    void poll();
    return () => {
      cancelled = true;
      clearInterval(timer);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [testRun?.id, testRun?.status, provider.id]);

  async function testAllModels() {
    if (runActive) return;
    setModelsError("");
    setModelMessage("");
    try {
      const response = await fetch(`/api/admin/providers/${encodeURIComponent(provider.id)}/models/test-all`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ models: modelList.map(model => model.model_id) })
      });
      const body = await response.json().catch(() => null) as ModelTestRun | null;
      if (!response.ok || !body) {
        setModelsError(t("providers.testQueueFailed"));
        return;
      }
      setTestRun(body);
      applyRunResults(body);
      router.refresh();
    } catch {
      setModelsError(t("providers.testQueueFailed"));
    }
  }

  async function reloadModels() {
    try {
      const response = await fetch(`/api/admin/providers/${encodeURIComponent(provider.id)}/models/configured`, { cache: "no-store" });
      if (!response.ok) return;
      const body = await response.json() as { data?: ConfiguredModel[] };
      if (Array.isArray(body.data)) setModelList(body.data);
    } catch {
      // keep the current list; the sync message already reports failures
    }
  }

  async function syncModels() {
    if (syncing) return;
    setSyncing(true);
    setModelsError("");
    setModelMessage("");
    try {
      const response = await fetch(`/api/admin/providers/${encodeURIComponent(provider.id)}/models/sync`, { method: "POST" });
      const body = await response.json().catch(() => null) as { imported?: number; total?: number; error?: string } | null;
      if (!response.ok) {
        setModelsError(t("providers.modelsSyncFailed"));
        return;
      }
      if (body?.error) {
        setModelsError(t("providers.modelsSyncFailed"));
        return;
      }
      setModelMessage(t("providers.modelsSynced", { imported: body?.imported ?? 0, total: body?.total ?? 0 }));
      await reloadModels();
      router.refresh();
    } catch {
      setModelsError(t("providers.modelsSyncFailed"));
    } finally {
      setSyncing(false);
    }
  }

  async function toggleModelActive(model: ConfiguredModel) {
    if (modelBusy) return;
    setModelBusy(model.model_id);
    setModelsError("");
    try {
      const response = await fetch(`/api/admin/providers/${encodeURIComponent(provider.id)}/models`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ model_id: model.model_id, active: !model.active })
      });
      if (!response.ok) {
        setModelsError(t("providers.modelUpdateFailed"));
        return;
      }
      setModelList(current => current.map(item => (
        item.model_id === model.model_id ? { ...item, active: !item.active } : item
      )));
      router.refresh();
    } catch {
      setModelsError(t("providers.modelUpdateFailed"));
    } finally {
      setModelBusy("");
    }
  }

  async function testModel(model: ConfiguredModel) {
    if (modelBusy) return;
    if (!selectedTestAccountInfo?.id) {
      setModelResults(current => ({ ...current, [model.model_id]: t("providers.noTestAccounts") }));
      return;
    }
    setModelBusy(model.model_id);
    setModelResults(current => ({ ...current, [model.model_id]: "" }));
    try {
      const response = await fetch(`/api/admin/providers/${encodeURIComponent(provider.id)}/models/test`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ model_id: model.model_id, account_id: selectedTestAccountInfo.id })
      });
      const body = await response.json().catch(() => null) as {
        ok?: boolean;
        latency_ms?: number;
        error?: string;
        accounts?: Array<{ account_name?: string; ok?: boolean; latency_ms?: number; error?: string }>;
      } | null;
      const result = body?.accounts?.[0];
      const name = result?.account_name || selectedTestAccountInfo.name;
      const message = response.ok && result
        ? result.ok
          ? t("providers.modelTestAccountOK", { account: name, ms: result.latency_ms ?? 0 })
          : t("providers.modelTestAccountFailed", { account: name, error: result.error || t("providers.modelTestFailed") })
        : t("providers.modelTestAccountFailed", { account: name, error: body?.error || t("providers.modelTestFailed") });
      setModelResults(current => ({ ...current, [model.model_id]: message }));
    } catch {
      setModelResults(current => ({ ...current, [model.model_id]: t("providers.modelTestAccountFailed", { account: selectedTestAccountInfo.name, error: t("providers.modelTestFailed") }) }));
    } finally {
      setModelBusy("");
    }
  }

  return (
    <>
      <Breadcrumbs
        items={[
          { label: t("nav.providers"), href: "/providers" },
          ...(provider.configured ? [] : [{ label: t("providers.catalog"), href: "/providers/catalog" }]),
          { label: provider.name }
        ]}
      />
      <div className="page-header">
        <div className="provider-detail-title">
          <span className="provider-icon">{provider.icon}</span>
          <div>
            <h1>{provider.name}</h1>
            <p className="sub">{t("providers.detailSubtitle")}</p>
          </div>
          <StatusChip status={provider.status} label={t(provider.statusKey)} />
        </div>
        <div className="actions">
          <Link className="btn" href={provider.configured ? "/providers" : "/providers/catalog"}>
            <ArrowLeft aria-hidden="true" size={16} />
            {provider.configured ? t("providers.backToProviders") : t("providers.backToCatalog")}
          </Link>
          <a
            className="btn"
            href={guideUrl(provider.catalogSlug ? PROVIDER_SETUP_GUIDE : provider.guide, provider.catalogSlug)}
            target="_blank"
            rel="noopener noreferrer"
          >
            {provider.catalogSlug ? t("providers.setupGuide") : t("providers.guideLink")}
            <ExternalLink aria-hidden="true" size={14} />
          </a>
          {provider.signupUrl ? (
            <a className="btn" href={provider.signupUrl} target="_blank" rel="noopener noreferrer">
              {t("providers.signupLink")}
              <ExternalLink aria-hidden="true" size={14} />
            </a>
          ) : null}
      {provider.configured ? (
            <>
              <button className="btn" type="button" onClick={testProvider} disabled={testing || accounts.length === 0}>
                {testing ? <LoaderCircle aria-hidden="true" size={16} className="spin" /> : <Play aria-hidden="true" size={16} />}
                {testing ? t("providers.testing") : t("common.test")}
              </button>
              <button className="btn" type="button" onClick={openModels} disabled={accounts.length === 0}>
                <Download aria-hidden="true" size={16} />
                {t("providers.importModels")}
              </button>
              <button className="btn" type="button" onClick={syncModels} disabled={syncing}>
                {syncing ? <LoaderCircle aria-hidden="true" size={16} className="spin" /> : <RefreshCw aria-hidden="true" size={16} />}
                {syncing ? t("providers.syncingModels") : t("providers.syncModels")}
              </button>
              <button className="btn" type="button" onClick={testAllModels} disabled={runActive || modelList.length === 0}>
                {runActive ? <LoaderCircle aria-hidden="true" size={16} className="spin" /> : <Play aria-hidden="true" size={16} />}
                {runActive ? t("providers.testingAllModels") : t("providers.testAllModels")}
              </button>
              <button className="btn primary" type="button" onClick={() => setOpen(true)}>
                <Plus aria-hidden="true" size={16} />
                {t("providers.addAccount")}
              </button>
            </>
          ) : (
            <Link className="btn primary" href={configurePath(provider.id)}>
              <Settings2 aria-hidden="true" size={16} />
              {t("providers.configure")}
            </Link>
          )}
        </div>
      </div>

      {testMessage ? <p className="small" role="status" style={{ marginBottom: 12 }}>{testMessage}</p> : null}

      <div className="detail-grid">
        <div className="card">
          <div className="row">
            <h2>{t("providers.linkedAccounts")}</h2>
            <span className="chip">{tp("providers.accountsCount", accounts.length)}</span>
          </div>
          {!provider.configured ? (
            <div className="empty-state" style={{ marginTop: 12 }}>
              <div className="empty-icon" aria-hidden="true">+</div>
              <h2>{t("providers.notConfiguredTitle")}</h2>
              <p>{t("providers.notConfiguredBody")}</p>
              <Link className="btn primary" href={configurePath(provider.id)}>
                <Settings2 aria-hidden="true" size={16} />
                {t("providers.configure")}
              </Link>
            </div>
          ) : accounts.length === 0 ? (
            <div className="empty-state" style={{ marginTop: 12 }}>
              <div className="empty-icon" aria-hidden="true">+</div>
              <h2>{t("providers.noAccounts")}</h2>
              <p>{t("providers.noAccountsBody")}</p>
            </div>
          ) : (
            <div className="account-list">
              {accounts.map(account => (
                <div className="account-row" key={account.id ?? `${account.name}-${account.meta}`}>
                  <div>
                    <div className="name">{account.name}</div>
                    <div className="meta">{account.meta}</div>
                    <div className="meta">
                      {t("providers.accountModels", { count: account.modelsAvailable ?? 0 })}
                      {" · "}
                      {t("providers.accountUsableModels", { count: account.modelsUsable ?? 0 })}
                    </div>
                    {accountMessages[account.id ?? ""] ? (
                      <div className="meta" role="status">{accountMessages[account.id ?? ""]}</div>
                    ) : null}
                  </div>
                  <div className="actions">
                    <StatusChip status={account.status} label={t(account.statusKey)} />
                    <button
                      className="btn"
                      type="button"
                      onClick={() => testAccount(account)}
                      disabled={!account.id || account.active !== true || testingAccountId === account.id}
                    >
                      {testingAccountId === account.id ? <LoaderCircle aria-hidden="true" size={16} className="spin" /> : <Play aria-hidden="true" size={16} />}
                      {testingAccountId === account.id ? t("providers.testing") : t("common.test")}
                    </button>
                    <button
                      className="btn"
                      type="button"
                      onClick={() => openEditAccount(account)}
                      disabled={!account.id}
                      aria-label={`${t("providers.editAccount")}: ${account.name}`}
                    >
                      <Pencil aria-hidden="true" size={16} />
                      {t("providers.editAccount")}
                    </button>
                    <button
                      className="btn"
                      type="button"
                      onClick={() => removeAccount(account)}
                      disabled={!account.id || removingId === account.id}
                      aria-label={t("providers.removeAccount")}
                      title={t("providers.removeAccount")}
                    >
                      {removingId === account.id
                        ? <LoaderCircle aria-hidden="true" size={16} className="spin" />
                        : <Trash2 aria-hidden="true" size={16} />}
                      {t("providers.removeAccount")}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <aside className="card">
          <h2>{t("providers.providerSummary")}</h2>
          <div className="list">
            <div className="list-item"><span>{t("common.status")}</span><span>{t(provider.statusKey)}</span></div>
            <div className="list-item"><span>{t("providers.origin.label")}</span><OriginChips origins={provider.origins} /></div>
            <div className="list-item"><span>{t("providers.type")}</span><span>{provider.type}</span></div>
            <div className="list-item"><span>{t("providers.models")}</span><span>{provider.modelsCount ?? 0}</span></div>
            <div className="list-item"><span>{t("providers.usableModels")}</span><span>{provider.modelsUsable ?? provider.modelsCount ?? 0}</span></div>
            <div className="list-item"><span>{t("providers.usableAccounts")}</span><span>{provider.usableAccounts ?? accounts.filter(account => account.active).length}</span></div>
            <div className="list-item">
              <span>{t("providers.capabilities")}</span>
              <span className="caps" style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
                {provider.capabilities.map(capability => <span className="chip" key={capability}>{capability}</span>)}
              </span>
            </div>
          </div>
        </aside>
      </div>

      <div className="card" style={{ marginTop: 16 }}>
        <div className="row">
          <h2>{t("providers.accountSummaryTitle")}</h2>
          <span className="chip">{tp("providers.accountsCount", accounts.length)}</span>
        </div>
        <div className="account-list">
          {accounts.map(account => (
            <div className="account-row" key={`summary-${account.id ?? account.name}`}>
              <div>
                <div className="name">{account.name}</div>
                <div className="meta">
                  {t("providers.accountModels", { count: account.modelsAvailable ?? 0 })}
                  {" · "}
                  {t("providers.accountUsableModels", { count: account.modelsUsable ?? 0 })}
                </div>
                <div className="caps" style={{ display: "flex", gap: 6, flexWrap: "wrap", marginTop: 4 }}>
                  {(account.capabilities ?? []).map(capability => (
                    <span className="chip" key={capability}>{capabilityLabel(capability)}</span>
                  ))}
                </div>
              </div>
              <div className="actions">
                <span className={`chip ${account.health === "active" ? "ok" : account.health === "quarantined" || account.health === "cooldown" ? "warn" : ""}`}>
                  {account.health === "active" ? t("providers.accountActive")
                    : account.health === "cooldown" ? t("providers.accountHealthCooldown")
                    : account.health === "quarantined" ? t("providers.accountHealthQuarantined")
                    : account.health === "disabled" ? t("providers.accountHealthDisabled")
                    : t("providers.accountHealthUnhealthy")}
                </span>
                {account.active ? <span className="chip ok">{t("providers.accountActive")}</span> : null}
              </div>
            </div>
          ))}
        </div>
      </div>

      {provider.configured ? (
        <div className="card" id="models" style={{ marginTop: 16 }}>
          <div className="row">
            <h2>{t("providers.modelsTitle")}</h2>
            <div className="actions">
              {testRun ? (
                <span className="chip" role="status">
                  {testRun.status === "completed"
                    ? t("providers.testQueueDone", { ok: testRun.ok_count, failed: testRun.failed_count })
                    : t("providers.testQueueProgress", { completed: testRun.completed, total: testRun.total })}
                </span>
              ) : null}
              <span className="chip">{tp("providers.modelsCount", modelList.length)}</span>
            </div>
          </div>
          <div className="toolbar" style={{ marginTop: 10 }}>
            <label className="small" style={{ display: "flex", gap: 8, alignItems: "center" }}>
              {t("providers.testAccountLabel")}
              <select
                className="input"
                value={selectedTestAccountInfo?.id ?? ""}
                onChange={event => chooseTestAccount(event.target.value)}
                disabled={testableAccounts.length === 0 || modelBusy !== ""}
                aria-label={t("providers.testAccountLabel")}
              >
                {testableAccounts.map(account => (
                  <option value={account.id} key={account.id}>{account.name}</option>
                ))}
              </select>
            </label>
            {testableAccounts.length === 0 ? (
              <span className="chip warn" role="status">{t("providers.noTestAccounts")}</span>
            ) : null}
          </div>
          {modelMessage ? <p className="small" role="status" style={{ marginTop: 8 }}>{modelMessage}</p> : null}
          {modelsError ? <p className="small" role="alert" style={{ color: "var(--danger)", marginTop: 8 }}>{modelsError}</p> : null}
          {bulkMessage ? <p className="small" role="status" style={{ marginTop: 8 }}>{bulkMessage}</p> : null}
          {modelList.length === 0 ? (
            <div className="empty-state" style={{ marginTop: 12 }}>
              <div className="empty-icon" aria-hidden="true">+</div>
              <h2>{t("providers.modelsEmptyTitle")}</h2>
              <p>{t("providers.modelsEmptyBody")}</p>
              <button className="btn" type="button" onClick={syncModels} disabled={syncing}>
                {syncing ? <LoaderCircle aria-hidden="true" size={16} className="spin" /> : <RefreshCw aria-hidden="true" size={16} />}
                {syncing ? t("providers.syncingModels") : t("providers.syncModels")}
              </button>
            </div>
          ) : (
            <>
              <div className="toolbar" style={{ marginTop: 12 }}>
                <input
                  className="input"
                  value={modelQueryFilter}
                  onChange={event => setModelQueryFilter(event.target.value)}
                  placeholder={t("providers.filterSearchPlaceholder")}
                  aria-label={t("providers.filterSearchPlaceholder")}
                />
                <span className="muted" role="status">
                  {t("providers.filteredCount", { shown: visibleModels.length, total: modelList.length })}
                </span>
                <button className="btn" type="button" onClick={clearModelFilters} disabled={!hasModelFilters}>
                  {t("providers.clearFilters")}
                </button>
              </div>

              <fieldset className="filter-chips" style={{ marginTop: 8, border: 0, padding: 0 }}>
                <legend className="muted small">{t("providers.filterPrice")}</legend>
                {([
                  ["all", "providers.priceAll"],
                  ["free", "providers.priceFree"],
                  ["paid", "providers.pricePaid"],
                  ["unknown", "providers.priceUnknown"]
                ] as Array<["all" | "free" | "paid" | "unknown", string]>).map(([value, key]) => (
                  <button
                    className={`filter-chip ${priceFilter === value ? "active" : ""}`}
                    type="button"
                    key={value}
                    aria-pressed={priceFilter === value}
                    onClick={() => setPriceFilter(value)}
                  >
                    {t(key)}
                  </button>
                ))}
              </fieldset>

              <fieldset className="filter-chips" style={{ marginTop: 8, border: 0, padding: 0 }}>
                <legend className="muted small">{t("providers.filterStatus")}</legend>
                {([
                  ["all", "providers.statusAll"],
                  ["active", "providers.modelActive"],
                  ["inactive", "providers.modelInactive"]
                ] as Array<["all" | "active" | "inactive", string]>).map(([value, key]) => (
                  <button
                    className={`filter-chip ${statusFilter === value ? "active" : ""}`}
                    type="button"
                    key={value}
                    aria-pressed={statusFilter === value}
                    onClick={() => setStatusFilter(value)}
                  >
                    {t(key)}
                  </button>
                ))}
              </fieldset>

              <fieldset className="filter-chips" style={{ marginTop: 8, border: 0, padding: 0 }}>
                <legend className="muted small">{t("providers.filterInput")}</legend>
                {INPUT_MODALITIES.map(modality => (
                  <button
                    className={`filter-chip ${inputFilters.includes(modality) ? "active" : ""}`}
                    type="button"
                    key={modality}
                    aria-pressed={inputFilters.includes(modality)}
                    onClick={() => toggleModality(setInputFilters, modality)}
                  >
                    {t(`providers.modality_${modality}`)}
                  </button>
                ))}
              </fieldset>

              <fieldset className="filter-chips" style={{ marginTop: 8, border: 0, padding: 0 }}>
                <legend className="muted small">{t("providers.filterOutput")}</legend>
                {OUTPUT_MODALITIES.map(modality => (
                  <button
                    className={`filter-chip ${outputFilters.includes(modality) ? "active" : ""}`}
                    type="button"
                    key={modality}
                    aria-pressed={outputFilters.includes(modality)}
                    onClick={() => toggleModality(setOutputFilters, modality)}
                  >
                    {t(`providers.modality_${modality}`)}
                  </button>
                ))}
              </fieldset>

              <fieldset className="filter-chips" style={{ marginTop: 8, border: 0, padding: 0 }}>
                <legend className="muted small">{t("providers.filterCapabilities")}</legend>
                {([
                  ["tools", "providers.capTools"],
                  ["reasoning", "providers.capReasoning"],
                  ["embeddings", "providers.capEmbeddings"],
                  ["stream", "providers.capStream"]
                ] as Array<[string, string]>).map(([value, key]) => (
                  <button
                    className={`filter-chip ${capFilters.includes(value) ? "active" : ""}`}
                    type="button"
                    key={value}
                    aria-pressed={capFilters.includes(value)}
                    onClick={() => toggleCapabilityFilter(value)}
                  >
                    {t(key)}
                  </button>
                ))}
              </fieldset>

              <div className="toolbar" style={{ marginTop: 12 }}>
                <label className="small" style={{ display: "flex", gap: 8, alignItems: "center" }}>
                  <input
                    type="checkbox"
                    checked={allFilteredSelected}
                    ref={element => { if (element) element.indeterminate = someFilteredSelected && !allFilteredSelected; }}
                    onChange={toggleSelectAllFiltered}
                  />
                  {t("providers.selectAllFiltered")}
                </label>
                <span className="muted">{tp("providers.selectedCount", selectedModelIDs.size)}</span>
                <div className="actions">
                  <button className="btn" type="button" onClick={() => bulkApply(true)} disabled={bulkBusy || selectedModelIDs.size === 0}>
                    {bulkBusy ? <LoaderCircle aria-hidden="true" size={16} className="spin" /> : <Power aria-hidden="true" size={16} />}
                    {t("providers.bulkActivate")}
                  </button>
                  <button className="btn" type="button" onClick={() => bulkApply(false)} disabled={bulkBusy || selectedModelIDs.size === 0}>
                    {bulkBusy ? <LoaderCircle aria-hidden="true" size={16} className="spin" /> : <Power aria-hidden="true" size={16} />}
                    {t("providers.bulkDeactivate")}
                  </button>
                </div>
              </div>

              {visibleModels.length === 0 ? (
                <p className="muted" role="status" style={{ marginTop: 12 }}>{t("providers.filterNoResults")}</p>
              ) : (
                <div className="account-list">
                  {visibleModels.map(model => (
                    <div className="account-row" key={model.model_id}>
                      <div style={{ display: "flex", gap: 10, alignItems: "flex-start" }}>
                        <input
                          type="checkbox"
                          checked={selectedModelIDs.has(model.model_id)}
                          onChange={() => toggleModelSelection(model.model_id)}
                          aria-label={`${t("providers.selectModel")} ${model.model_id}`}
                        />
                        <div>
                          <div className="name">{model.display_name || model.model_id}</div>
                          <div className="meta">{model.model_id}</div>
                          <div className="caps" style={{ display: "flex", gap: 6, flexWrap: "wrap", marginTop: 4 }}>
                            {priceKnown(model) ? (
                              <span className="chip">
                                {isFree(model) ? t("providers.priceFree") : t("providers.pricePaid")}
                              </span>
                            ) : (
                              <span className="chip">{t("providers.metadataUnknown")}</span>
                            )}
                            {INPUT_MODALITIES.filter(modality => hasModality(model, "input", modality)).map(modality => (
                              <span className="chip" key={`in-${modality}`} aria-label={`${t("providers.filterInput")}: ${t(`providers.modality_${modality}`)}`}>
                                {t(`providers.modality_${modality}`)}
                              </span>
                            ))}
                            {OUTPUT_MODALITIES.filter(modality => hasModality(model, "output", modality)).map(modality => (
                              <span className="chip" key={`out-${modality}`} aria-label={`${t("providers.filterOutput")}: ${t(`providers.modality_${modality}`)}`}>
                                {t("providers.outputShort", { modality: t(`providers.modality_${modality}`) })}
                              </span>
                            ))}
                            {capability(model, "tools") ? <span className="chip">{t("providers.capTools")}</span> : null}
                            {capability(model, "reasoning") ? <span className="chip">{t("providers.capReasoning")}</span> : null}
                            {capability(model, "embeddings") ? <span className="chip">{t("providers.capEmbeddings")}</span> : null}
                            {capability(model, "stream") ? <span className="chip">{t("providers.capStream")}</span> : null}
                            {contextWindow(model) > 0 ? (
                              <span className="chip">{t("providers.contextTokens", { tokens: formatTokens(contextWindow(model)) })}</span>
                            ) : null}
                            {!metadataKnown(model) ? <span className="chip">{t("providers.capNotInformed")}</span> : null}
                          </div>
                            {modelResults[model.model_id] ? (
                            <div className="meta" role="status">{modelResults[model.model_id]}</div>
                          ) : null}
                        </div>
                      </div>
                      <div className="actions">
                        <span className="chip">
                          {model.active ? t("providers.modelActive") : t("providers.modelInactive")}
                        </span>
                        <button
                          className="btn"
                          type="button"
                          onClick={() => testModel(model)}
                          disabled={modelBusy === model.model_id}
                          aria-label={`${t("providers.modelTest")} ${model.model_id}`}
                        >
                          {modelBusy === model.model_id
                            ? <LoaderCircle aria-hidden="true" size={16} className="spin" />
                            : <Play aria-hidden="true" size={16} />}
                          {t("providers.modelTest")}
                        </button>
                        <button
                          className="btn"
                          type="button"
                          onClick={() => toggleModelActive(model)}
                          disabled={modelBusy === model.model_id}
                          aria-label={`${model.active ? t("providers.deactivateModel") : t("providers.activateModel")} ${model.model_id}`}
                        >
                          {modelBusy === model.model_id
                            ? <LoaderCircle aria-hidden="true" size={16} className="spin" />
                            : <Power aria-hidden="true" size={16} />}
                          {model.active ? t("providers.deactivateModel") : t("providers.activateModel")}
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </>
          )}
        </div>
      ) : null}

      {open ? (
        <div className="modal-backdrop" role="presentation" onClick={event => { if (event.target === event.currentTarget) setOpen(false); }}>
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="account-modal-title">
            <div className="modal-head">
              <div>
                <h2 id="account-modal-title">{t("providers.addAccount")}</h2>
                <p className="muted">{t("providers.accountModalBody")}</p>
              </div>
              <button className="modal-close" type="button" aria-label={t("common.close")} onClick={() => setOpen(false)}>
                <X aria-hidden="true" size={16} />
              </button>
            </div>
            <form onSubmit={addAccount}>
              <div className="form-grid">
                <label className="field">
                  <span>{t("providers.accountName")}</span>
                  <input className="input" required value={name} onChange={event => setName(event.target.value)} placeholder={t("providers.accountNamePlaceholder")} />
                </label>
                <label className="field">
                  <span>{t("providers.accountCredential")}</span>
                  <input className="input" required type="password" value={credential} onChange={event => setCredential(event.target.value)} placeholder={t("providers.accountCredentialPlaceholder")} />
                </label>
              </div>
              {accountError ? <p className="small" style={{ color: "var(--danger)" }}>{accountError}</p> : null}
              <div className="modal-foot">
                <button className="btn" type="button" onClick={() => setOpen(false)}>{t("common.cancel")}</button>
                <button className="btn primary" type="submit">{t("providers.saveAccount")}</button>
              </div>
            </form>
          </div>
        </div>
      ) : null}

      {editingAccount ? (
        <div className="modal-backdrop" role="presentation" onClick={event => { if (event.target === event.currentTarget) setEditingAccount(null); }}>
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="edit-account-title">
            <div className="modal-head">
              <div>
                <h2 id="edit-account-title">{t("providers.editAccountTitle")}</h2>
                <p className="muted">{t("providers.editAccountBody", { provider: provider.name })}</p>
              </div>
              <button className="modal-close" type="button" aria-label={t("common.close")} onClick={() => setEditingAccount(null)}>
                <X aria-hidden="true" size={16} />
              </button>
            </div>
            <form onSubmit={saveAccountName}>
              <label className="field">
                <span>{t("providers.accountName")}</span>
                <input className="input" required value={editName} onChange={event => setEditName(event.target.value)} aria-describedby="edit-account-provider" />
              </label>
              <p className="muted small" id="edit-account-provider">{t("providers.providerNameReadonly", { provider: provider.name })}</p>
              {editError ? <p className="small" role="alert" style={{ color: "var(--danger)" }}>{editError}</p> : null}
              <div className="modal-foot">
                <button className="btn" type="button" onClick={() => setEditingAccount(null)}>{t("common.cancel")}</button>
                <button className="btn primary" type="submit" disabled={editBusy || !editName.trim()}>
                  {editBusy ? <LoaderCircle aria-hidden="true" size={16} className="spin" /> : <Pencil aria-hidden="true" size={16} />}
                  {editBusy ? t("common.loading") : t("providers.saveAccount")}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}

      {modelsOpen ? (
        <div className="modal-backdrop" role="presentation" onClick={event => { if (event.target === event.currentTarget) setModelsOpen(false); }}>
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="models-modal-title">
            <div className="modal-head">
              <div>
                <h2 id="models-modal-title">{t("providers.importModels")}</h2>
                <p className="muted">{t("providers.importModelsBody")}</p>
              </div>
              <button className="modal-close" type="button" aria-label={t("common.close")} onClick={() => setModelsOpen(false)}>
                <X aria-hidden="true" size={16} />
              </button>
            </div>
            <div className="field">
              <span>{t("providers.searchModels")}</span>
              <div style={{ display: "flex", gap: 8 }}>
                <input className="input" value={modelQuery} onChange={event => setModelQuery(event.target.value)} placeholder={t("providers.searchModelsPlaceholder")} />
                <button className="btn" type="button" onClick={toggleAllModels} disabled={filteredModels.length === 0}>
                  {allSelected ? t("providers.clearSelection") : t("providers.selectAll")}
                </button>
              </div>
            </div>
            {modelError ? <p className="small" style={{ color: "var(--danger)" }}>{modelError}</p> : null}
            {modelsLoading ? (
              <p className="muted"><LoaderCircle aria-hidden="true" size={16} className="spin" /> {t("common.loading")}</p>
            ) : filteredModels.length === 0 ? (
              <p className="muted">{t("providers.noRemoteModels")}</p>
            ) : (
              <div className="model-list">
                {filteredModels.map(model => (
                  <label className="model-row" key={model.id}>
                    <input type="checkbox" checked={selectedModels.has(model.id)} onChange={() => toggleModel(model.id)} />
                    <span>
                      <strong>{model.id}</strong>
                      {model.name && model.name !== model.id ? <small>{model.name}</small> : null}
                    </span>
                  </label>
                ))}
              </div>
            )}
            <div className="modal-foot">
              <span className="muted">{tp("providers.selectedModels", selectedModels.size)}</span>
              <button className="btn" type="button" onClick={() => setModelsOpen(false)}>{t("common.cancel")}</button>
              <button className="btn primary" type="button" onClick={importModels} disabled={importing || selectedModels.size === 0}>
                {importing ? <LoaderCircle aria-hidden="true" size={16} className="spin" /> : <Download aria-hidden="true" size={16} />}
                {importing ? t("providers.importingModels") : t("providers.importSelected")}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </>
  );
}
