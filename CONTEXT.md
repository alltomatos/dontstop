# CONTEXT — dontstop

> Documento de contexto e linguagem ubíqua. Fonte de verdade complementar a
> [`AGENTS.md`](AGENTS.md); decisões em [`docs/05-ADRS.md`](docs/05-ADRS.md).

## Produto

Gateway corporativo de IA auto-hospedado, single-tenant no P0, com endpoint OpenAI-compatible
(`/v1/chat/completions`, `/v1/models`), catálogo de provedores, múltiplas contas com failover,
cache exato/semântico, pool de proxy Webshare, DLP, quotas/custo e observabilidade. `tenant_id`
existe desde o início para evoluir a multi-tenancy.

## Linguagem ubíqua

| Termo | Definição |
|---|---|
| **Tenant** | Unidade de isolamento; P0 usa `00000000-0000-0000-0000-000000000000`. |
| **Provider** | Provedor de LLM configurado (nativo do catálogo ou custom). |
| **Account** | Credencial de uma conta do provider; pode ter `proxy_mode`. |
| **Proxy endpoint** | IP:porta de saída (`source` = `webshare` ou `manual`), com score/saúde. |
| **Assignment** | Vínculo conta↔proxy; **1 conta por IP por provedor** (ADR-024). |
| **Proxy mode** | `inherit` \| `off` \| `manual` \| `auto` (`preferred`/`required` legados). |
| **Poison** | Ban persistente por `(tenant, provedor, endpoint)` com TTL por classe de falha. |
| **Combo** | Modelo virtual por tenant com membros e failover na mesma requisição. |
| **Endpoint policy** | Política por `/v1/*` (auth, escopo, rate limit, cache, DLP, proxy…). |
| **API key** | Credencial do consumidor (Argon2id), com escopos, quotas e expiração. |

## Invariantes inegociáveis

- **Isolamento de proxy:** 1 conta por IP **por provedor**; reuso entre provedores é permitido.
- **Poison por provedor:** um IP banido por X continua elegível para Y.
- **Fail-closed:** `manual`/`auto`/`required` nunca caem para conexão direta.
- **Rotação transparente:** na mesma requisição, antes do primeiro chunk.
- **Tenancy:** `tenant_id` em todas as tabelas; Redis `t:{tenant}:`; nunca cruzar tenants.
- **Segredos:** credenciais cifradas (AES-256-GCM); nunca retornadas, logadas ou em trace.
- **Catálogo é fonte de verdade:** sem hardcode de modelo/protocolo/política em handler.

## Bounded contexts (data plane Go)

`routing` (seleção/failover), `proxy` (pool/rotação/poison), `cache` (exato/semântico),
`dlp`, `usage` (custo), `endpoint` (registry/políticas), `apikey`, `mcp`, `auth`,
`chat` (orquestração), `admin` (serviços internos), `api` (HTTP), `platform` (config/db/redis/otel).

## Sidecars

- **AgentRouter relay** (`vendor/agentrouter-relay`, MIT): Python + `anthropic` pinado, interno, sem
  porta pública e sem túnel; a chave chega via `x-api-key` do gateway (ADR-025). Disponibilidade/budget
  do AgentRouter são dinâmicos (402/503), nunca hardcoded.

## Onde encontrar

- Arquitetura: [`docs/architecture/ADD.md`](docs/architecture/ADD.md)
- ADRs: [`docs/05-ADRS.md`](docs/05-ADRS.md) (índice em [`docs/adr/`](docs/adr/README.md))
- DER: [`docs/02-DER.md`](docs/02-DER.md) · UML: [`docs/03-UML.md`](docs/03-UML.md)
- Contrato público: [`docs/contracts/openapi-v1.yaml`](docs/contracts/openapi-v1.yaml)
- Admin API: [`docs/reference/ADMIN-API.md`](docs/reference/ADMIN-API.md)
- Roadmap: [`DEVELOPER-ROADMAP.md`](DEVELOPER-ROADMAP.md)
- Governança: [`docs/agents/`](docs/agents/)
