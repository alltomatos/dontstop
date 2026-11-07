"use client";

import Link from "next/link";
import { useState } from "react";
import { Breadcrumbs, PageHeader } from "@/components/console/page";
import { CopyButton, Tabs } from "@/components/console/ui";
import { useI18n } from "@/components/console/i18n-provider";

const mainApis = [
  { name: "Chat Completions", path: "/v1/chat/completions" },
  { name: "Responses", path: "/v1/responses", key: "endpoints.responses" },
  { name: "Messages", path: "/v1/messages", key: "endpoints.messages" },
  { name: "Completions", path: "/v1/completions" }
];

const mediaApis = [
  { name: "Embeddings", path: "/v1/embeddings" },
  { name: "Images", path: "/v1/images/generations", key: "endpoints.imageGeneration" },
  { name: "Edits", path: "/v1/images/edits", key: "endpoints.imageEdits" },
  { name: "Audio", path: "/v1/audio/transcriptions", key: "endpoints.audioTranscription" },
  { name: "Video", path: "/v1/videos/generations", key: "endpoints.videoGeneration" }
];

const utilityApis = [
  { name: "Moderations", path: "/v1/moderations", key: "endpoints.moderations" },
  { name: "Batches", path: "/v1/batches", key: "endpoints.batch" },
  { name: "Files", path: "/v1/files", key: "endpoints.files" },
  { name: "Models", path: "/v1/models" }
];

export default function EndpointsPage() {
  const { t, tp } = useI18n();
  const [tab, setTab] = useState("overview");
  const base = "https://gateway.dontstop.corp/v1";
  return (
    <>
      <Breadcrumbs items={[{ label: t("nav.endpoints") }]} />
      <PageHeader
        title={t("endpoints.finalTitle")}
        subtitle={t("endpoints.hint")}
        actions={(
          <>
            <button className="btn" type="button">{t("common.testEndpoint")}</button>
            <button className="btn primary" type="button">{t("common.managePolicies")}</button>
          </>
        )}
      />

      <Tabs
        value={tab}
        onChange={setTab}
        items={[
          { id: "overview", label: t("endpoints.finalTitle") },
          { id: "policies", label: t("endpoints.policies") },
          { id: "aliases", label: t("endpoints.aliases") },
          { id: "health", label: t("common.health") }
        ]}
      />

      {tab === "overview" ? (
        <div className="stack">
          <section className="card">
            <h2>{t("endpoints.apiTitle")}</h2>
            <div className="stack">
              <div className="list-item">
                <span><span className="chip ok"><span className="dot" />{t("endpoints.public")}</span> <span className="mono">{base}</span></span>
                <CopyButton value={base} label={t("endpoints.public")} />
              </div>
              <div className="list-item">
                <span><span className="chip ok"><span className="dot" />{t("endpoints.local")}</span> <span className="mono">http://localhost:20128/v1</span></span>
                <CopyButton value="http://localhost:20128/v1" label={t("endpoints.local")} />
              </div>
            </div>
            <h3>{t("endpoints.localServer")}</h3>
            <div className="actions">
              <span className="chip mono">0.0.0.0:20128</span>
              <span className="chip mono">127.0.0.1:20128</span>
              <span className="chip mono">[::]:20128</span>
              <span className="chip ok"><span className="dot" />{t("common.running")}</span>
            </div>
          </section>

          <section className="card">
            <div className="row">
              <div>
                <h2>{t("endpoints.available")}</h2>
                <p className="muted">{t("endpoints.modelsInEndpoints")}</p>
              </div>
              <span className="chip">{t("endpoints.active")}</span>
            </div>
            <h3>{t("providers.mainApis")}</h3>
            <div className="grid-2">
              {mainApis.map(item => <EndpointTile key={item.path} name={item.key ? t(item.key) : item.name} path={item.path} />)}
            </div>
            <h3>{t("endpoints.mediaMultimodal")}</h3>
            <div className="grid-2">
              {mediaApis.map(item => <EndpointTile key={item.path} name={item.key ? t(item.key) : item.name} path={item.path} />)}
            </div>
            <h3>{t("endpoints.utilities")}</h3>
            <div className="grid-2">
              {utilityApis.map(item => <EndpointTile key={item.path} name={item.key ? t(item.key) : item.name} path={item.path} />)}
            </div>
          </section>
        </div>
      ) : null}

      {tab === "aliases" ? (
        <section className="card">
          <h2>{t("endpoints.aliases")}</h2>
          <div className="stack">
            {[
              [t("endpoints.baseUrl"), `${base}`],
              ["/v1/models", `${base}/models`],
              ["/v1/chat/completions", `${base}/chat/completions`]
            ].map(([label, value]) => (
              <div className="list-item" key={value}>
                <span>{label} <span className="mono muted">{value}</span></span>
                <CopyButton value={value} label={label} />
              </div>
            ))}
          </div>
        </section>
      ) : null}

      {tab === "policies" ? (
        <section className="card">
          <h2>{t("endpoints.policies")}</h2>
          <table className="table">
            <thead><tr><th>{t("common.name")}</th><th>{t("common.status")}</th><th>{t("common.action")}</th></tr></thead>
            <tbody>
              {mainApis.map(item => (
                <tr key={item.path}>
                  <td>{item.key ? t(item.key) : item.name}</td>
                  <td><span className="chip ok">{t("common.enabled")}</span></td>
                  <td className="right"><Link className="btn" href="/api-manager">{t("common.managePolicies")}</Link></td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      ) : null}

      {tab === "health" ? (
        <section className="card">
          <h2>{t("common.health")}</h2>
          <p className="muted">{t("endpoints.hint")}</p>
          <div className="grid-2">
            <div className="card"><div className="label">p95</div><div className="value num">28 ms</div></div>
            <div className="card"><div className="label">{t("common.errors")}</div><div className="value num">0,4%</div></div>
          </div>
          <p className="small muted">{tp("endpoints.modelsCount", 1208)}</p>
        </section>
      ) : null}
    </>
  );
}

function EndpointTile({ name, path }: { name: string; path: string }) {
  return (
    <div className="card">
      <div className="row"><strong>{name}</strong><CopyButton value={path} label={name} /></div>
      <div className="mono muted small">{path}</div>
    </div>
  );
}
