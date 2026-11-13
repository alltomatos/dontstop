# Validação AgentRouter (ponta a ponta)

> Data: 2026-11-13 · Fonte de protocolo: <https://agentrouter.org/docs/opencode.html>
> Mapeamento de WAF/proxy: [`AGENTROUTER-ALLCODE-RESEARCH.md`](../reference/AGENTROUTER-ALLCODE-RESEARCH.md)
> Nenhuma chave é reproduzida neste documento.

## Protocolos documentados

| Gateway | Base URL | Endpoint | Auth |
|---|---|---|---|
| Anthropic (adapter nativo do dontstop) | `https://agentrouter.org` (sem `/v1`) | `POST /v1/messages` | `x-api-key` + `anthropic-version: 2023-06-01` |
| OpenAI-compatível | `https://agentrouter.org/v1` | `POST /v1/chat/completions` | `Authorization: Bearer` |

Modelos (lista estática; o console é a fonte dinâmica): `claude-opus-5`, `claude-opus-4-8`,
`deepseek-v4-flash`, `glm-5.3`, `gpt-5.6-sol`.

## Resultado por etapa

| Etapa | Ação | Resultado |
|---|---|---|
| 1. Listagem de modelos | Catálogo estático do dontstop (`provider_catalog_models`, slug `agentrouter`) | ✅ 5 modelos |
| 1b. `/v1/models` upstream | **Não chamado** — bloqueado pelo WAF por mapeamento | ⏭️ (por design) |
| 2. Auth Anthropic (`POST /v1/messages`, `x-api-key`) | Chave nova (`agentrouter2`) e antiga | ❌ `401 unauthorized_client_error` |
| 3. Auth OpenAI-compat (`POST /v1/chat/completions`, `Bearer`) | Chave nova | ❌ `401 unauthorized_client_error` |
| 4. Chat real via adapter nativo (`TestAgentRouterLive`, prompt em inglês) | `deepseek-v4-flash` | ❌ `provider authentication failed` |
| 5. Filtro de conteúdo (inglês) | Prompts em inglês | ⏭️ não alcançado (WAF barra antes) |

Evidência do corpo (sanitizada):

```json
{"error":{"message":"unauthorized client detected, contact support for assistance ..."},
 "message":"UNAUTHENTICATED","success":false,"type":"unauthorized_client_error"}
```

Headers: `X-Oneapi-Request-Id` (a requisição chega ao relay One API), `Content-Type: application/json`.
Não há página HTML de challenge do Cloudflare — o bloqueio é o WAF de **fingerprint de conexão**
descrito no mapeamento (`AGENTROUTER-ALLCODE-RESEARCH.md`, seção "Transporte e autenticação").

## Conclusão e bloqueio

- **O transporte nativo (Go `net/http`) é bloqueado pelo WAF** com `unauthorized_client_error`,
  independentemente da chave (nova e antiga) e do protocolo. Não é falha de chave comprovada; é o
  comportamento mapeado para clientes genéricos.
- O mapeamento registra que o AllCode contornava com um **relay Python síncrono** do SDK `anthropic`.
  O dontstop **não adota** esse relay, e não foram criados contornos (sem spoof de UA/TLS, sem relay,
  sem túnel). Conforme o mapeamento, **o provedor fica indisponível até haver solução nativa**.
- Listagem de modelos segue pelo **catálogo estático** (correto por design); `/v1/models` não é chamado.

## Correção aplicada

- `internal/provider/agentrouter_live_test.go`: prompt `"Ol?"` (português) → `"Reply with the single
  word: ok"` (inglês), conforme a regra de conteúdo do AgentRouter (`400 content-blocked` para
  não-inglês).

## Segurança

- A chave foi usada apenas via arquivo local + variável de ambiente (`AGENTROUTER_LIVE_KEY`), nunca
  impressa, logada ou versionada.
- Recomenda-se rotacionar/remover `D:\apikey-tests.txt` após a validação.
