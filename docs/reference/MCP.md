# MCP - contrato e referência

> **Status:** implementado (ADR-026). Desligado por padrão; opt-in por tenant. Política em
> [`docs/05-ADRS.md`](../05-ADRS.md) (ADR-016) e decisão de runtime em
> [`docs/adr/0026-mcp-runtime-and-management-tools.md`](../adr/0026-mcp-runtime-and-management-tools.md).

## 1. Escopo

Servidor MCP executado **no gateway** (mesmo binário) com ferramentas de chat/modelos/uso/saúde e, na Fase A,
ferramentas de gestão de **combos** e leitura de providers. Não é relay, não é túnel e não expõe segredo de
provider/API key.

## 2. Transportes

| Transporte | Endpoint | Auth | Uso |
|---|---|---|---|
| Streamable HTTP | `POST/GET/DELETE /mcp` | `Authorization: Bearer <api-key>` | default (Claude/IDE remoto) |
| SSE (legado) | `GET /sse` + `POST /messages` | Bearer | clientes antigos (deprecated) |
| stdio | `gateway mcp` | `DONTSTOP_MCP_API_KEY` | IDE local (o processo é iniciado pela IDE) |

> `gateway mcp` reutiliza `DATABASE_URL`, `REDIS_URL` e `MASTER_KEY` do gateway; a chave em
> `DONTSTOP_MCP_API_KEY` define tenant e escopos. Stdio só inicia com `enabled=true` e `transport=stdio`.

`DONTSTOP_BASE_PATH` (ex.: `/dontstop`) é preservado em `/mcp`, `/sse` e `/messages`. Sem Bearer → `401`;
tenant desligado ou transporte divergente → **`404`** (fail-closed, sem chamar upstream).

## 3. Autorização

- `auth.Verifier` resolve `Principal{KeyID, TenantID, Scopes}`; o tenant vem da API key, nunca do payload.
- `mcp_settings.enabled` + `transport` são checados **antes** do handler.
- `tools/list` retorna apenas ferramentas `enabled=true` (`mcp_tool_settings`) **e** com escopo do principal.
- `tools/call` de ferramenta não anunciada/desconhecida → `-32602`; sem escopo → `isError` com mensagem
  acionável (não vaza detalhe interno).

## 4. Ferramentas

### 4.1 Base

| Tool | Escopo | Anotação | Observação |
|---|---|---|---|
| `dontstop_chat` | `chat:write` | `readOnlyHint=false`, `openWorldHint=false` | Chat via roteamento/cache/failover do gateway. |
| `dontstop_models` | `models:read` | `readOnlyHint=true` | Catálogo do tenant. |
| `dontstop_embeddings` | `embeddings:write` | `readOnlyHint=true` | Enquanto não houver backend: `isError` acionável. |
| `dontstop_web_search` | `web:read` | `readOnlyHint=true`, `openWorldHint=true` | Enquanto não houver backend: `isError` acionável. |
| `dontstop_usage` | `usage:read` | `readOnlyHint=true` | Custo/uso agregado do tenant. |
| `dontstop_health` | (autenticado) | `readOnlyHint=true` | Liveness/readiness sem segredo. |

### 4.2 Gestão - Fase A (ADR-026)

| Tool | Escopo | Anotação | Observação |
|---|---|---|---|
| `dontstop_combo_catalog` | `combos:read` | `readOnlyHint=true` | Estratégias e modelos elegíveis para montar combos. |
| `dontstop_combo_list` | `combos:read` | `readOnlyHint=true` | Combos do tenant. |
| `dontstop_combo_get` | `combos:read` | `readOnlyHint=true` | Detalhe/membros/status. |
| `dontstop_combo_create` | `combos:write` | `readOnlyHint=false`, `idempotentHint=false` | Valida nome/estratégia/membros no catálogo. |
| `dontstop_combo_update` | `combos:write` | `readOnlyHint=false`, `idempotentHint=false` | Atualiza nome/estratégia/membros. |
| `dontstop_combo_activate` | `combos:write` | `readOnlyHint=false`, `idempotentHint=true` | Ativa/desativa o combo. |
| `dontstop_combo_deactivate` | `combos:write` | `readOnlyHint=false`, `idempotentHint=true` | Desativa o combo. |
| `dontstop_combo_delete` | `combos:write` | `readOnlyHint=false`, `destructiveHint=true` | Remove o combo. |
| `dontstop_provider_list` | `models:read` | `readOnlyHint=true` | Providers/modelos conectados e capabilities. |

`dontstop_combo_update` recebe `{id, revision, combo:{...mesmo schema de create...}}` (revisão otimista).

**Fase B (deferida):** connect de provider/add conta, API keys, endpoints/políticas, DLP e proxy — escopos
dedicados (`providers:write`, `keys:write`, `endpoints:write`, `dlp:write`) e segredo **fora do prompt**
(secret store/referência; nunca ecoar credencial).

## 5. JSON-RPC (exemplos)

```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"claude","version":"1.0"}}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"dontstop_combo_create","arguments":{"slug":"qa-combo","name":"QA combo","strategy":"priority","members":[{"provider_id":"<uuid>","model_id":"gpt-4o"}]}}}
```

Resposta de chat (resumo): `content` com texto e `structuredContent` com `model`, `usage`, `cache`, `request_id`.
Erro de ferramenta: `{"jsonrpc":"2.0","id":3,"result":{"content":[{"type":"text","text":"..."}],"isError":true}}`.

## 6. Erros

| Situação | Resultado |
|---|---|
| Desligado / transporte divergente | HTTP `404` |
| Sem Bearer / chave inválida | HTTP `401` |
| Sem escopo para a ferramenta | `isError` sanitizado (ou `403` no HTTP) |
| Ferramenta desconhecida/não anunciada | JSON-RPC `-32602` |
| Método inexistente | JSON-RPC `-32601` |
| Falha de execução | `isError:true` com mensagem acionável |

## 7. Conexão

```json
{
  "mcpServers": {
    "dontstop": {
      "type": "http",
      "url": "https://dontstop.example.com/mcp",
      "headers": { "Authorization": "Bearer dsk_..." }
    },
    "dontstop-local": {
      "command": "gateway",
      "args": ["mcp"],
      "env": { "DONTSTOP_MCP_API_KEY": "dsk_..." }
    }
  }
}
```

SSE legado: `"url": "https://dontstop.example.com/sse"` (mesmo header). Nunca commitar a chave; ela é exibida
apenas na criação e pode ser revogada no API Manager.

## 8. Auditoria e observabilidade

- `audit_outbox`: `mcp.enable/disable`, `mcp.transport`, `mcp.tool.enable/disable`, `mcp.restart`,
  `mcp.tool.call` (tool/status/tenant/duração) e `mcp.combo.create/update/activate/deactivate/delete`.
- `POST /admin/v1/mcp/restart` é **soft**: audita e faz novas sessões/tools releiam a configuração; conexões
  SSE já abertas permanecem até reconectar.
- Spans `mcp.initialize`/`mcp.tools.list`/`mcp.tools.call` e métricas `mcp.requests`/`mcp.tool.calls`/`mcp.tool.errors`.
- **Nunca** payload, PII, argumento de ferramenta ou credencial em log/trace.

## 9. Segurança

- Desligado por padrão; sem HTTP MCP sem Bearer/escopo; **proibido túnel**.
- Isolamento por tenant em todos os serviços e no cache (`t:{tenant}:`).
- Ferramentas de gestão exigem `combos:read`/`combos:write`; a Fase A não move segredos.
- Rate limit/quotas específicos do MCP ficam para P1 (herda limites da API key/endpoint).
