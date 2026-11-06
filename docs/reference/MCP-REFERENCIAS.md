# Referência — MCP no 9Router e OmniRoute

> Insumo para o Épico E12 do [PRD](../01-PRD.md) e para a página [MCP](../../design-system/pages/mcp.md).
> **Decisão do dontstop:** MCP desligado por padrão, local-only/autenticado, sem túneis.

## 1. 9Router

**Arquivos observados**

- `src/app/(dashboard)/dashboard/mcp/page.js`
- `src/mcp/engine.js`
- `src/mcp/tools/*`
- `src/mcp/transport/{sse,stdio}.js`
- `src/app/api/mcp/native/{sse,message,tools}/route.js`

**Comportamento**

- O servidor MCP nativo fica **sempre exposto** em `/api/mcp/native/sse`; não há `enabled` global na página.
- A página lista ferramentas, snippets de configuração e um marketplace de MCPs.
- Transportes: `stdio` e `SSE`.
- Não há status de processo, PID, uptime, heartbeat nem atividade 24h na página.

**O que aproveitar**

- Catálogo de ferramentas com descrição e configuração copiável.
- SSE nativo no mesmo processo do gateway.

**O que evitar**

- Exposição sempre ligada.
- Ausência de toggle global e de trilha de ativação.

## 2. OmniRoute

**Arquivos observados**

- `src/app/(dashboard)/dashboard/mcp/page.tsx`
- `src/app/api/mcp/status/route.ts`
- `src/app/api/mcp/sse/route.ts`
- `src/app/api/mcp/stream/route.ts`
- `src/app/api/mcp/tools/route.ts`
- `src/app/api/mcp/restart/route.ts`
- `src/app/api/mcp/audit/route.ts`
- `open-sse/mcp-server`

**Comportamento**

- `mcpEnabled` global e `mcpTransport` (`stdio`, `sse`, `streamable-http`).
- Cartões de status: processo, PID, uptime, heartbeat; atividade 24h; top ferramentas; detalhes de runtime; escopos.
- Ações: reiniciar, copiar endpoint/comando; auditoria de ativação e chamadas.
- Ferramentas exigem escopo; desativar remove o anúncio.

**O que aproveitar**

- Toggle global + estado `Online/Offline`.
- Seleção de transporte com endpoint/comando copiável.
- Status, atividade, top ferramentas e auditoria.
- Ferramenta desativada não é anunciada.

**O que evitar**

- Expor HTTP sem auth.
- Qualquer opção de túnel.

## 3. Comparativo

| Capacidade | 9Router | OmniRoute | dontstop (P0) |
|---|---|---|---|
| Toggle global | Não | Sim (`mcpEnabled`) | **Sim, desligado por padrão** |
| Transportes | stdio, SSE | stdio, SSE, Streamable HTTP | **stdio, SSE, Streamable HTTP** |
| Status/PID/uptime | Não | Sim | **Sim** |
| Atividade 24h | Não | Sim | **Sim** |
| Ferramentas + escopos | Lista | Lista + escopos | **Lista + switch + escopo** |
| Auditoria | Parcial | Sim | **Sim** |
| Túnel | Não | Não | **Proibido** |

## 4. Decisões para o dontstop

1. **Desligado por padrão** (`mcp_settings.enabled=false`); ativar é auditado.
2. **Transportes:** `stdio` local; `SSE` e `Streamable HTTP` exigem Bearer/escopo.
3. **Ferramentas:** só anuncia se `enabled=true` e o chamador tiver escopo (`chat:write`, `models:read`, `embeddings:write`, `web:read`, `usage:read`).
4. **Status e atividade:** PID, uptime, heartbeat, chamadas 24h, sucesso e latência.
5. **Sem túneis:** Cloudflare/Tailscale/ngrok proibidos em todas as fases.
6. **Logs sem payload/PII:** guardar ferramenta, escopo, status, latência e hash.
