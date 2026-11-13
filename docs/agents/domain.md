# Domain

> Contexto de domínio e linguagem ubíqua: [`CONTEXT.md`](../../CONTEXT.md).

## Bounded contexts (Go)

| Contexto | Pacote | Responsabilidade |
|---|---|---|
| Roteamento | `internal/routing` | candidatos, prioridade/peso, failover, circuit breaker |
| Proxy | `internal/proxy` | pool, seleção, rotação, poison por provedor |
| Chat | `internal/chat` | orquestração de `/v1/chat/completions`, cache, proxy, usage |
| Cache | `internal/cache` | cache exato (Redis) e metadados |
| DLP | `internal/dlp` | detectores e política |
| Uso/custo | `internal/usage` | tokens, custo, outbox |
| Endpoints | `internal/endpoint` | registry e políticas por `/v1/*` |
| API keys | `internal/apikey` | escopos, quotas, rotação |
| MCP | `internal/mcp` | transportes e ferramentas (opt-in) |
| Auth | `internal/auth` | API keys e contexto de tenant |
| Admin | `internal/admin` | serviços internos do console |
| API | `internal/api` | handlers HTTP e router |
| Plataforma | `internal/platform` | config, DB, Redis, segredos, OTel |

## Regras de domínio

- O catálogo versionado é a fonte de verdade de provider/modelo/capability/preço/protocolo.
- Cada conta de provedor tem seu próprio IP **dentro do provedor** (ADR-024).
- Um IP banido por um provedor não é entregue a outra conta desse provedor.
- Credenciais são cifradas e nunca cruzam a fronteira da API.
