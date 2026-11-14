"use client";

import { useEffect, useState } from "react";
import { Breadcrumbs, PageHeader } from "@/components/console/page";
import { CopyButton, Toggle } from "@/components/console/ui";
import { useI18n } from "@/components/console/i18n-provider";
import type { MCPActivity, MCPStatus } from "@/lib/admin-api";

export function McpView({ initialStatus, initialActivity }: { initialStatus: MCPStatus | null; initialActivity: MCPActivity | null }) {
  const { t } = useI18n();
  const [enabled, setEnabled] = useState(initialStatus?.enabled ?? false);
  const [transport, setTransport] = useState<MCPStatus["transport"]>(initialStatus?.transport ?? "streamable-http");
  const [tools, setTools] = useState<MCPStatus["tools"]>(initialStatus?.tools ?? []);
  const [toolState, setToolState] = useState<Record<string, boolean>>(() =>
    Object.fromEntries((initialStatus?.tools ?? []).map(tool => [tool.id, tool.enabled]))
  );
  const [activity, setActivity] = useState<MCPActivity | null>(initialActivity);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const response = await fetch("/api/admin/mcp/activity?hours=24", { cache: "no-store" });
        if (!response.ok) return;
        const next = await response.json() as MCPActivity;
        if (!cancelled) setActivity(next);
      } catch {
        // Keep the server-rendered snapshot when the Admin API is unavailable.
      }
    })();
    return () => { cancelled = true; };
  }, []);

  function applyStatus(status: MCPStatus) {
    setEnabled(status.enabled);
    setTransport(status.transport);
    setTools(status.tools);
    setToolState(Object.fromEntries(status.tools.map(tool => [tool.id, tool.enabled])));
  }

  async function persistSettings(next: { enabled?: boolean; transport?: string }) {
    try {
      const response = await fetch("/api/admin/mcp/settings", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(next)
      });
      if (response.ok) {
        applyStatus(await response.json() as MCPStatus);
      }
    } catch {
      // Keep local state when the Admin API is unavailable.
    }
  }

  async function persistTool(toolID: string, value: boolean) {
    try {
      const response = await fetch(`/api/admin/mcp/tools/${encodeURIComponent(toolID)}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ enabled: value })
      });
      if (response.ok) {
        applyStatus(await response.json() as MCPStatus);
      }
    } catch {
      // Keep local state when the Admin API is unavailable.
    }
  }

  async function restart() {
    try {
      const response = await fetch("/api/admin/mcp/restart", { method: "POST" });
      if (response.ok) {
        window.location.reload();
      }
    } catch {
      // Keep the current view when the Admin API is unavailable.
    }
  }

  const endpoint = transport === "stdio" ? "gateway mcp" : transport === "sse" ? "https://gateway.dontstop.corp/sse" : "https://gateway.dontstop.corp/mcp";
  const enabledScopes = Array.from(new Set(tools.filter(tool => toolState[tool.id] ?? true).map(tool => tool.scope).filter(Boolean)));
  const totalCalls = activity?.total_calls ?? 0;
  const successRate = activity?.success_rate ?? 0;
  const avgLatency = activity?.avg_latency_ms ?? 0;

  return (
    <>
      <Breadcrumbs items={[{ label: t("nav.mcp") }]} />
      <PageHeader
        title={t("mcp.server")}
        subtitle={t("mcp.subtitle")}
        actions={<button className="btn" type="button" onClick={() => void restart()}>{t("common.restart")}</button>}
      />

      {!enabled ? (
        <section className="empty-state">
          <div className="empty-icon" aria-hidden="true">+</div>
          <h2>{t("mcp.disabledTitle")}</h2>
          <p>{t("mcp.disabledHint")}</p>
          <button className="btn primary" type="button" onClick={() => { setEnabled(true); void persistSettings({ enabled: true, transport }); }}>{t("mcp.enableServer")}</button>
        </section>
      ) : (
        <div className="stack">
          <section className="card">
            <div className="row">
              <div>
                <h2>{t("mcp.server")}</h2>
                <p className="muted">{t("mcp.summary")}</p>
              </div>
              <div className="actions">
                <span className="chip ok"><span className="dot" />{t("common.online")}</span>
                <Toggle checked={enabled} onChange={value => { setEnabled(value); void persistSettings({ enabled: value }); }} label={t("mcp.disableServer")} />
                <span className="small muted">{enabled ? t("mcp.toggleOn") : t("mcp.toggleOff")}</span>
              </div>
            </div>
            <h3>{t("mcp.howToConnect")}</h3>
            <ol className="muted">
              <li>{t("mcp.step1", { command: "gateway mcp" })}</li>
              <li>{t("mcp.step2")}</li>
              <li>{t("mcp.step3", { tool: "dontstop_chat" })}</li>
            </ol>
          </section>

          <div className="grid-3">
            {[
              ["stdio", t("mcp.stdioHint")],
              ["sse", t("mcp.sseHint")],
              ["streamable-http", t("mcp.streamHint")]
            ].map(([id, hint]) => (
              <button
                className={`card ${transport === id ? "selected" : ""}`}
                key={id}
                type="button"
                onClick={() => { setTransport(id as MCPStatus["transport"]); void persistSettings({ transport: id }); }}
                aria-pressed={transport === id}
              >
                <strong>{id}</strong>
                <div className="small muted">{hint}</div>
              </button>
            ))}
          </div>

          <section className="card">
            <div className="row">
              <h3>{t("mcp.transport")}</h3>
              <span className="chip">{transport}</span>
            </div>
            <div className="list-item">
              <span className="mono">{endpoint}</span>
              <CopyButton value={endpoint} label={t("mcp.transport")} />
            </div>
            <div className="grid-3" style={{ marginTop: 12 }}>
              <div className="list-item"><span>{t("mcp.processStatus")}</span><span className="chip ok">{t("common.online")}</span></div>
              <div className="list-item"><span>PID</span><span className="mono">—</span></div>
              <div className="list-item"><span>{t("mcp.uptime")}</span><span>runtime</span></div>
              <div className="list-item"><span>{t("mcp.heartbeat")}</span><span>live</span></div>
            </div>
          </section>

          <section className="card">
            <h3>{t("mcp.activity")}</h3>
            <div className="kpis" style={{ gridTemplateColumns: "repeat(3, minmax(0, 1fr))" }}>
              <div><div className="label">{t("mcp.totalCalls")}</div><div className="value num">{totalCalls.toLocaleString()}</div></div>
              <div><div className="label">{t("mcp.successRate")}</div><div className="value num">{successRate.toFixed(1)}%</div></div>
              <div><div className="label">{t("mcp.avgLatency")}</div><div className="value num">{Math.round(avgLatency)} ms</div></div>
            </div>
            <div className="grid-2">
              <div>
                <h4>{t("mcp.topTools")}</h4>
                <div className="list">
                  {(activity?.top_tools ?? []).map(item => (
                    <div className="list-item" key={item.tool}><span className="mono">{item.tool}</span><span className="num">{item.count.toLocaleString()}</span></div>
                  ))}
                  {(activity?.top_tools ?? []).length === 0 ? <div className="small muted">—</div> : null}
                </div>
              </div>
              <div>
                <h4>{t("mcp.runtime")}</h4>
                <div className="list">
                  <div className="list-item"><span>{t("mcp.transport")}</span><span>{transport}</span></div>
                  <div className="list-item"><span>{t("mcp.appliedScopes")}</span><span className="mono">{enabledScopes.length ? enabledScopes.join(", ") : "—"}</span></div>
                </div>
              </div>
            </div>
          </section>

          <section className="card">
            <h3>{t("mcp.toolsAndScopes")}</h3>
            <table className="table">
              <thead><tr><th>{t("mcp.tool")}</th><th>{t("mcp.scope")}</th><th>{t("common.status")}</th></tr></thead>
              <tbody>
                {tools.map(tool => {
                  const isEnabled = toolState[tool.id] ?? true;
                  return (
                    <tr key={tool.id}>
                      <td className="mono">{tool.name}</td>
                      <td className="mono">{tool.scope || "—"}</td>
                      <td>
                        <Toggle
                          checked={isEnabled}
                          onChange={value => { setToolState(current => ({ ...current, [tool.id]: value })); void persistTool(tool.id, value); }}
                          label={t(isEnabled ? "mcp.toolDisable" : "mcp.toolEnable", { tool: tool.name })}
                        />
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
            <p className="small muted">{t("mcp.toolDisabledHint")}</p>
          </section>
        </div>
      )}
    </>
  );
}
