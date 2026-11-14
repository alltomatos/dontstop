# Evidência — runtime MCP (P0)

**Data:** 2026-11-13
**Escopo:** slices #62–#71 (runtime, stdio, streamable HTTP, SSE, escopos/auditoria, combos, provider list, restart/UI, evidências).
**Ambiente:** Go 1.26.11, gateway local em Postgres 17 + Redis 8 (containers locais), SDK oficial `github.com/modelcontextprotocol/go-sdk` v1.8.0.

## Resultado funcional

| Verificação | Resultado |
|---|---|
| `GET /healthz` | 200 |
| `POST /mcp` `initialize` (Bearer, streamable-http) | 200, `serverInfo` presente |
| `POST /mcp` `tools/list` | 15 ferramentas, incluindo `dontstop_chat` e `dontstop_combo_list` |
| `POST /mcp` `tools/call` `dontstop_combo_list` | `isError=false` (tenant sem combos) |
| `POST /mcp` `tools/call` `dontstop_health` | `isError=false`, `status=ok` |
| `POST /mcp` sem Bearer | **401** |
| `POST /mcp` com Bearer e MCP desligado | **404** |
| `gateway mcp` (stdio) `initialize`/`tools/list`/`tools/call` | IDs 1–4 com resultado, sem `isError` |

> Observação: no stdio as respostas podem chegar fora de ordem (id 4 antes de id 3) por concorrência; todos os ids foram respondidos.

## Auditoria

`audit_outbox` após as chamadas HTTP:

| event_type | count |
|---|---|
| `mcp.enable` | 1 |
| `mcp.transport` | 1 |
| `mcp.tool.call` | 2 |

`mcp.tool.call`: `dontstop_combo_list` (`ok`, 18 ms) e `dontstop_health` (`ok`, 0 ms). Nenhum argumento, payload ou PII foi gravado.

No estado desligado, `mcp.tool.call` permaneceu em **0**.

## Testes e verificações

- `go test ./...` — verde (integração MCP com Postgres dedicado também verde).
- `go vet ./...` — verde.
- `golangci-lint run` — 0 issues.
- `govulncheck ./...` — 0 vulnerabilidades chamadas.
- Admin: `npm test` 68/68, `npm run typecheck`, `npm run lint` e `npm run build` verdes (rotas `/api/admin/mcp/activity` e `/api/admin/mcp/restart` geradas).

## Observações

- O E2E usou um banco dedicado (`dontstop_mcp_e2e`) criado e removido ao final; nenhum dado do stack local foi alterado.
- Chaves/MASTER_KEY usadas no teste não foram registradas neste documento.
- Nada foi commitado ou enviado ao remoto.
