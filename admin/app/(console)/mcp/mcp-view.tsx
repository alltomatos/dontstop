"use client";

import { useState } from "react";
import { Breadcrumbs, PageHeader } from "@/components/console/page";
import { CopyButton, Toggle } from "@/components/console/ui";
import { useI18n } from "@/components/console/i18n-provider";
import type { MCPStatus } from "@/lib/admin-api";

const tools = [
  { name: "dontstop_chat", scope: "chat:write" },
  { name: "dontstop_models", scope: "models:read" },
  { name: "dontstop_embeddings", scope: "embeddings:write" },
  { name: "dontstop_web_search", scope: "web:read" },
  { name: "dontstop_usage", scope: "usage:read" },
  { name: "dontstop_health", scope: "models:read" }
];

export function McpView({ initialStatus }: { initialStatus: MCPStatus | null }) {
  const { t } = useI18n();
  const [enabled, setEnabled] = useState(initialStatus?.enabled ?? false);
  const [transport, setTransport] = useState<MCPStatus["transport"]>(initialStatus?.transport ?? "streamable-http");
  const [toolState, setToolState] = useState<Record<string, boolean>>(() =>
    Object.fromEntries(tools.map(tool => [tool.name, initialStatus?.tools.find(item => item.id === tool.name)?.enabled ?? true]))
  );

  async function persistSettings(next: { enabled?: boolean; transport?: string }) {
    try {
      const response = await fetch("/api/admin/mcp/settings", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(next)
      });
      if (response.ok) {
        const status = await response.json() as MCPStatus;
        setEnabled(status.enabled);
        setTransport(status.transport);
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
        const status = await response.json() as MCPStatus;
        setToolState(Object.fromEntries(status.tools.map(tool => [tool.id, tool.enabled])));
      }
    } catch {
      // Keep local state when the Admin API is unavailable.
    }
  }

  const endpoint = transport === "stdio" ? "dontstop mcp" : transport === "sse" ? "https://gateway.dontstop.corp/api/mcp/sse" : "https://gateway.dontstop.corp/api/mcp/stream";

  return (
    <>
      <Breadcrumbs items={[{ label: t("nav.mcp") }]} />
      <PageHeader
        title={t("mcp.server")}
        subtitle={t("mcp.subtitle")}
        actions={<button className="btn" type="button">{t("common.restart")}</button>}
      />

      {!enabled ? (
        <section className="empty-state">
          <div className="empty-icon" aria-hidden="true">+</div>
          <h2>{t("mcp.disabledTitle")}</h2>
          <p>{t("mcp.disabledHint")}</p>
          <button className="btn primary" type="button" onClick={() => { setEnabled(true); void persistSettings({ enabled: true }); }}>{t("mcp.enableServer")}</button>
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
              <li>{t("mcp.step1", { command: "dontstop mcp" })}</li>
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
              <div className="list-item"><span>{t("mcp.uptime")}</span><span>2h14m</span></div>
              <div className="list-item"><span>{t("mcp.heartbeat")}</span><span>12s</span></div>
            </div>
          </section>

          <section className="card">
            <h3>{t("mcp.activity")}</h3>
            <div className="kpis" style={{ gridTemplateColumns: "repeat(3, minmax(0, 1fr))" }}>
              <div><div className="label">{t("mcp.totalCalls")}</div><div className="value num">1.284</div></div>
              <div><div className="label">{t("mcp.successRate")}</div><div className="value num">99,2%</div></div>
              <div><div className="label">{t("mcp.avgLatency")}</div><div className="value num">107 ms</div></div>
            </div>
            <div className="grid-2">
              <div>
                <h4>{t("mcp.topTools")}</h4>
                <div className="list">
                  <div className="list-item"><span className="mono">dontstop_chat</span><span className="num">812</span></div>
                  <div className="list-item"><span className="mono">dontstop_models</span><span className="num">221</span></div>
                </div>
              </div>
              <div>
                <h4>{t("mcp.runtime")}</h4>
                <div className="list">
                  <div className="list-item"><span>{t("mcp.transport")}</span><span>{transport}</span></div>
                  <div className="list-item"><span>{t("mcp.appliedScopes")}</span><span className="mono">chat:write, models:read</span></div>
                </div>
              </div>
            </div>
          </section>

          <section className="card">
            <h3>{t("mcp.toolsAndScopes")}</h3>
            <table className="table">
              <thead><tr><th>{t("mcp.tool")}</th><th>{t("mcp.scope")}</th><th>{t("common.status")}</th></tr></thead>
              <tbody>
                {tools.map(tool => (
                  <tr key={tool.name}>
                    <td className="mono">{tool.name}</td>
                    <td className="mono">{tool.scope}</td>
                    <td>
                      <Toggle
                        checked={toolState[tool.name]}
                        onChange={value => { setToolState(current => ({ ...current, [tool.name]: value })); void persistTool(tool.name, value); }}
                        label={t(toolState[tool.name] ? "mcp.toolDisable" : "mcp.toolEnable", { tool: tool.name })}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <p className="small muted">{t("mcp.toolDisabledHint")}</p>
          </section>
        </div>
      )}
    </>
  );
}
