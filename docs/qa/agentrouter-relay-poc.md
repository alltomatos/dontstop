# AgentRouter — estudo e PoC do relay Python (vendorização)

> Data: 2026-11-13 · Fonte: `D:\dev\allcode\packages\allcode\src\provider\agentrouter\{relay.py,manager.ts,python.ts}`
> Licença do AllCode: **MIT** (copyright 2025 opencode). Nenhuma chave é reproduzida aqui.
> **Status:** implementado como sidecar vendorizado (ADR-025) em `vendor/agentrouter-relay`.

## Como o relay funciona

- `relay.py` é um servidor HTTP **stdlib** (`ThreadingHTTPServer`) em `127.0.0.1:7187`.
- Reemite as requisições pelo **SDK Python `anthropic` síncrono** (`anthropic.Anthropic(api_key, base_url)`).
  Esse é o único "shape" de cliente que passa pelo WAF de fingerprint do AgentRouter; clientes
  genéricos (Node fetch/undici, curl, httpx async e o `net/http` do Go) recebem
  `401 unauthorized client detected`.
- Endpoints: `GET /health` → `{ok:true,version:2}`; `GET /v1/models|/models` → **stub estático** (5
  modelos); `POST /v1/messages|/messages` → sync ou SSE.
- Limpeza de payload: remove `thinking`, `output_config`, blocos `thinking`/`redacted_thinking` e
  `cache_control`; traduz texto para inglês (Google `translate_a/single`, fallback MyMemory) e
  acrescenta instrução de resposta em pt-BR.
- SSE: usa `client.messages.create(stream=True)` de baixo nível, descarta `billing_summary` e
  serializa com `exclude_none`.
- `manager.ts`: localiza um Python com `anthropic` (ou pip), instala o pacote em runtime, cai para um
  **CPython portátil 3.12.7** (python-build-standalone) se necessário; sobe o relay com
  `AGENTROUTER_API_KEY`, espera até 60 s pelo `/health` (version 2) e recupera porta órfã.

## PoC (chave nova, isolada)

Relay do AllCode executado com `anthropic==2.6.0` em porta isolada (**7188**), com a chave nova:

| Etapa | Resultado |
|---|---|
| `/health` | `{"ok":true,"version":2}` (após o warm-up) |
| `POST /v1/messages` `gpt-5.6-sol` (inglês) | ✅ `200`, `text="ok"`, `in=80`, `out=4` |
| `deepseek-v4-flash`, `glm-5.3`, `claude-opus-4-8`, `claude-opus-5` | ❌ `503` "sem canal no grupo default" |
| Adapter nativo Go (sem relay) | ❌ `401 unauthorized_client_error` (WAF) |

Conclusão do PoC: **o relay passa pelo WAF e a chave nova funciona**; a disponibilidade de modelos
depende do grupo/conta — nesta conta só `gpt-5.6-sol` respondeu. O 503 é do AgentRouter, não do relay.

## Modelos e budget (console, 2026-11)

O console do AgentRouter informa: `gpt-6.1sol` recomendado (multiplicador baixo), `gpt-6-astra`
ilimitado 24/7, `claude-opus-5.5` temporariamente offline. PoC pela chave nova:

| Modelo | Resultado |
|---|---|
| `gpt-5.6-sol` | ✅ 200 (`text="ok"`, in=80/out=4) |
| `gpt-6.1sol` | ❌ 503 "sem canal no grupo default" |
| `gpt-6-astra` | ❌ **402 "Budget pool quota has been exhausted"** |
| `deepseek-v4-flash`, `glm-5.3`, `claude-opus-4-8`, `claude-opus-5` | ❌ 503 "sem canal" |

Leitura: **transporte/chave OK**; o que limita é **canal por grupo** (503) e **budget pool** (402).
O dontstop deve mapear `402` → quota (`ErrProviderQuota`) e `503` → indisponível, sem tratar como falha
de proxy/autenticação. O catálogo estático precisa ser atualizado para os nomes atuais
(`gpt-6-astra`, `gpt-6.1sol`, `claude-opus-5.5`) e a disponibilidade é dinâmica.

## Proposta de vendorização (escopo restrito)

1. `vendor/agentrouter-relay/`: `relay.py` (fork), `LICENSE` (MIT/opencode + NOTICE), `requirements.txt`
   (`anthropic==2.6.0` pinado), `Dockerfile` (`python:3.12-slim`, não-root, sem `pip` em runtime),
   `README.md` (proveniência e diff).
2. **Fork**: remover a tradução best-effort e a instrução pt-BR (o adapter `agentrouter` do dontstop já
   traduz para inglês com falha fechada e localiza a resposta). Manter limpeza de payload, normalização
   SSE, `/health` e o stub de modelos. Ler a chave de `AGENTROUTER_API_KEY_FILE` (Docker secret) **ou**
   do header `x-api-key` encaminhado pelo gateway (recomendado: mantém a chave no cofre cifrado do
   dontstop e permite rotação por conta).
3. **Deploy**: serviço `agentrouter-relay` no Swarm, somente na rede interna (`app`), **sem porta
   publicada**, `healthcheck /health`, `replicas: 1`, segredo `agentrouter_api_key`, sem túneis.
4. **Gateway**: a conta `agentrouter` aponta `base_url` para `http://agentrouter-relay:7187`
   (protocolo Anthropic). O catálogo estático continua sendo a fonte de modelos.
5. **Testes**: contrato do adapter com relay fake; live opt-in pelo relay (inglês); `pytest` mínimo do
   fork (clean_payload/SSE/health); teste de que a chave não aparece em log/resposta.

## Conflitos que exigem ADR

- `AGENTS.md` §8/§14 e `AGENTROUTER-ALLCODE-RESEARCH.md` dizem "sem relay de referência em runtime" e
  "o relay Python não é adotado". A vendorização é uma **mudança de arquitetura** → **ADR-025**
  delimitando: somente AgentRouter, relay **vendorizado** (MIT), sidecar interno, sem o app AllCode no
  caminho, sem instalação de SDK em runtime (deps no build), sem túneis; tradução fail-closed fica no
  gateway.
- Alternativa nativa: replicar o fingerprint TLS/HTTP2 do SDK em Go (`uTLS`). Evita Python, mas é
  frágil/não documentado. O PoC mostra que o relay é o caminho de menor risco hoje.

## Segurança

- Chave usada apenas via arquivo local/env do processo do relay; **não** apareceu em logs (verificado).
- Relays do PoC encerrados; portas 7187/7188 livres.
- Recomenda-se rotacionar/remover `D:\apikey-tests.txt` após a decisão.
