# QA — Validação E2E local (containers)

> Data: 2026-11-13 · Branch `docs/provider-setup-guide` · commits `18f1cc0`, `1c5bf76`.
> Evidência sem segredos/PII.

## Build e deploy

- Imagens locais: `dontstop-gateway:local-20261113-agentrouter`,
  `dontstop-admin:local-20261113-agentrouter`,
  `dontstop-agentrouter-relay:local-20261113-agentrouter`.
- Stack `dontstop`: gateway `:8080`, admin `:3000`, relay interno, Traefik `0/0`.
- `/readyz` 200 · `/login` 200 · relay `/health` 200.
- Incidente: secret NATS `unused` derrubou o gateway; contornado em runtime removendo
  `NATS_URL_FILE` e definindo `NATS_URL=`. O stack base exige NATS válido (ou secret vazio).

## Resultados

| Fluxo | Resultado |
|---|---|
| `/v1/models` (API key) | 200 · 8 modelos |
| Chat non-stream (`gpt-5.6-sol` via relay) | 200 · `"ok"` · in=13/out=4 |
| Chat stream | 200 · 2 eventos `data:` · `[DONE]` |
| Auth | sem chave 401; `models:read` no chat 403 |
| Failover cross-provider (AgentRouter 402 → OpenRouter) | 200 · `combo/qa-cross-provider` |
| Cache exato | `X-Donstop-Cache: hit` · hits 0→2 |
| DLP (Admin) | list 200 · create 201 (`cpf`/`block`) |
| Proxy | manual 201 + assign 200; WebShare 100/100; pool 101 |
| Admin API | 9/9 200 |
| Console | login/dashboard/combos/proxies OK |
| AgentRouter relay | `POST /v1/messages 200`; sem chave nos logs |

## Pendências

- DLP **não** está cablado no data plane (apenas Admin CRUD).
- AgentRouter: `gpt-6.1sol` 503, `gpt-6-astra` 402 (budget), `claude-opus-5.5` offline.
- Failover no mesmo provider/conta é bloqueado pelo cooldown após 402; cross-provider validado.
- Headers `X-Donstop-Combo*` vazios.
- MCP: apenas gestão de settings/tools no Admin; **runtime MCP ausente**.
