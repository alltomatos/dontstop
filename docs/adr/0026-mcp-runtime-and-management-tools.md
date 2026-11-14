# ADR-026 — MCP runtime no gateway: Go SDK, transportes e ferramentas de gestão

**Status:** Aceito
**Data:** 2026-11-13
**Implementa/estende:** [ADR-016](../05-ADRS.md) — a política opt-in/local-only/sem túnel continua; este ADR registra a arquitetura do runtime e as ferramentas de gestão para agentes.

## Contexto

O MCP do dontstop hoje é **apenas administrativo**: `mcp_settings`/`mcp_tool_settings`, `GET /admin/v1/mcp/status`,
`PATCH /settings`, `PATCH /tools/{id}` e a página `/mcp`. Não existe runtime MCP (sem `/mcp`, `/sse` ou `/v1/mcp`;
sem `initialize`/`tools/list`/`tools/call`), as mudanças de settings/tools não são auditadas e o botão
“Reiniciar” é inerte. O requisito é um servidor MCP funcional, desligado por padrão, com `stdio`, `SSE` e
`Streamable HTTP`, Bearer/escopo, e um agente deve poder **criar/operar combos** (e, em fase seguinte,
outras operações de gestão).

O SDK oficial Go (`github.com/modelcontextprotocol/go-sdk` v1.7+) implementa a spec e os transportes; é uma
dependência **compilada/pinada** (não é instalação de SDK em runtime). O data plane é Go e o MCP deve
reutilizar `auth.Verifier`, `chat.Service`, `provider.Catalog`, `admin.ComboService`/`UsageService` e
`health.Registry`, sempre no `tenant_id` do chamador.

## Decisão

1. **Runtime no gateway** com o SDK Go oficial (pinado). Protocolo negociado pelo SDK; baseline
   initialize-based (2025-06-18/2025-11-25), compatível com Claude Desktop/IDEs.
2. **Transportes:** `Streamable HTTP` em `POST/GET/DELETE /mcp`; `SSE` legado em `GET /sse` + `POST /messages`
   (deprecated, mantido por requisito); `stdio` via subcomando **`gateway mcp`** (in-process; chave em
   `DONTSTOP_MCP_API_KEY`; logs em stderr).
3. **Desligado por padrão.** HTTP exige **Bearer** (API key) via `auth.Verifier`; gate por tenant
   (`mcp_settings.enabled` + transporte configurado) **antes** do handler; desligado/mismatch → **404**,
   sem chamar upstream.
4. **Ferramentas filtradas** por `enabled` (`mcp_tool_settings`) **e** escopo do `Principal`, em ordem
   determinística. Ferramenta não anunciada/desconhecida → JSON-RPC `-32602`; erro de execução →
   `isError:true` com mensagem acionável e sanitizada.
5. **Ferramentas base:** `dontstop_chat`, `dontstop_models`, `dontstop_embeddings` (erro acionável: backend
   ausente), `dontstop_web_search` (idem), `dontstop_usage`, `dontstop_health`.
6. **Ferramentas de gestão (Fase A):** `dontstop_combo_catalog/list/get/create/update/activate/deactivate/delete`
   e `dontstop_provider_list`; escopos **`combos:read`**/ **`combos:write`** (provider/model read usa
   `models:read`). **Nenhum segredo** (credencial de provider/API key) trafega pelo MCP nesta fase.
7. **Fase B (deferida):** connect de provider/add conta, API keys, endpoints/políticas, DLP e proxy, com
   escopos dedicados (`providers:write`, `keys:write`, `endpoints:write`, `dlp:write`) e fluxo de segredo
   **fora do prompt** (referência/secret store; nunca ecoar credencial).
8. **Auditoria** em `audit_outbox` (sem args/PII): `mcp.enable/disable`, `mcp.transport`, `mcp.tool.enable/disable`,
   `mcp.restart`, `mcp.tool.call` (tool/status/tenant/duração) e `mcp.combo.create/update/activate/deactivate/delete`.
9. **Restart:** `POST /admin/v1/mcp/restart` (manage) → audita e reseta cache/sessões novas; `stdio` é
   reiniciado pela IDE.
10. **Observabilidade:** spans `mcp.initialize`/`mcp.tools.list`/`mcp.tools.call` (tenant/tool/status) e
    métricas `mcp.requests`/`mcp.tool.calls`/`mcp.tool.errors`; logs só método/ferramenta/status/tenant.
11. **Segurança:** sem túnel; sem HTTP MCP sem Bearer/escopo; isolamento por tenant (serviços e cache por
    `tenant_id`); nenhum payload/PII em log/trace; anotações `readOnlyHint`/`destructiveHint`/`idempotentHint`/
    `openWorldHint=false` nas ferramentas.

## Alternativas rejeitadas

- **JSON-RPC hand-rolled:** mais risco de divergência de protocolo/transportes; o SDK oficial cobre versões e SSE/streamable.
- **Servidor MCP TS/Python separado:** novo runtime fora do data plane Go, duplicação de auth/serviços e de deploy.
- **Proxy do Admin API por loopback:** amplia superfície, mistura sessão admin com API keys e dificulta o gate por tenant/escopo.
- **MCP sempre ligado:** contraria ADR-016.
- **Credenciais/keys via prompt do agente:** risco de vazamento no modelo/logs; fica para a Fase B com fluxo de segredo.
- **Somente SSE:** legado e sem clientes modernos; streamable HTTP é o default.

## Consequências

- (+) Runtime MCP real, agentic combos, fail-closed por padrão, auditoria e observabilidade; reutiliza os serviços e o tenant.
- (+) SDK oficial maduro (spec 2026-07-28 e anteriores), transports padronizados.
- (?) Nova dependência Go (compilada) e mais testes de contrato/transporte.
- (?) `SSE` é legado; `embeddings`/`web_search` ainda não têm backend (erro acionável até a Fase B).
- (?) Escopos novos (`combos:read/write`) exigem documentação/UI de API keys; rate limit/quotas do MCP ficam para P1.

## Documentos correspondentes

[`architecture/ADD.md`](../architecture/ADD.md) · [`../02-DER.md`](../02-DER.md) · [`../03-UML.md`](../03-UML.md) ·
[`../reference/MCP.md`](../reference/MCP.md) · [`../../design-system/pages/mcp.md`](../../design-system/pages/mcp.md) ·
[`../agents/issue-tracker.md`](../agents/issue-tracker.md).
