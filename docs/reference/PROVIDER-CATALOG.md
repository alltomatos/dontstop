# Catálogo versionado de provedores — estado da implementação

> Complementa [`PROVIDERS-MAPPING.md`](PROVIDERS-MAPPING.md). O mapeamento é o levantamento; este
> documento registra o que foi implementado, o que ficou de fora e as evidências de validação local.

## 1. O que existe agora

| Componente | Onde |
|---|---|
| Tabelas do catálogo | `gateway/migrations/0011_provider_catalog.sql` |
| Serviço de catálogo | `gateway/internal/admin/catalog.go` |
| Rotas administrativas | `gateway/internal/api/admin_catalog.go`, `router.go` |
| Coluna de rastreabilidade | `providers.catalog_slug` |
| Testes | `admin/catalog_test.go`, `admin/catalog_integration_test.go`, `api/admin_catalog_test.go` |

As tabelas são globais (não por tenant) e versionadas por `catalog_version`:

- `provider_catalog`: `slug`, `display_name`, `protocol`, `api_version`, `default_base_url`,
  `models_path`, `auth_type`, `auth_header`, `auth_scheme`, `headers`, `capabilities`, `network`,
  `quirks`, `source_ref`, `catalog_version`, `enabled`.
- `provider_catalog_models`: `slug`, `model_id`, `display_name`, `context_window`,
  `max_output_tokens`, `supports_stream`, `supports_tools`, `supports_vision`,
  `supports_reasoning`, preços por milhão (input/output/cached) e `pricing_source`.

O catálogo **não** duplica as tabelas por tenant: `providers`, `provider_accounts` e
`provider_models` continuam sendo as tabelas operacionais. Ao conectar por slug, o gateway grava
`providers.kind` derivado do protocolo, `providers.catalog_slug`, `provider_accounts.base_url` do
catálogo e os modelos de `provider_catalog_models` em `provider_models`.

### Protocolo → adapter

| `provider_catalog.protocol` | `providers.kind` | Adapter |
|---|---|---|
| `openai` | `openai` | `internal/provider/openai.go` |
| `anthropic` | `anthropic` | `internal/provider/anthropic.go` |

Protocolos sem adapter no escopo atual (`gemini`, `bedrock`, `openai-responses`, `azure`, OAuth
proprietários) foram deliberadamente **não** cadastrados: cadastrar sem adapter violaria a regra de
não ter entrada de catálogo que não funcione de ponta a ponta.

## 2. Provedores adicionados (16)

Critério: presentes nos **três** projetos de referência (21 no total) **e** compatíveis com os
adapters existentes (API key + wire suportado).

| Slug | Protocolo | Base URL | Modelos | Auth |
|---|---|---|---|---|
| `alibaba` | openai | `https://dashscope-intl.aliyuncs.com/compatible-mode/v1` | 3 | Bearer |
| `anthropic` | anthropic | `https://api.anthropic.com` | 3 | `x-api-key` |
| `cerebras` | openai | `https://api.cerebras.ai/v1` | 3 | Bearer |
| `cohere` | openai | `https://api.cohere.com/compatibility/v1` | 2 | Bearer |
| `groq` | openai | `https://api.groq.com/openai/v1` | 3 | Bearer |
| `mistral` | openai | `https://api.mistral.ai/v1` | 3 | Bearer |
| `nvidia` | openai | `https://integrate.api.nvidia.com/v1` | 3 | Bearer |
| `openai` | openai | `https://api.openai.com/v1` | 4 | Bearer |
| `opencode-go` | openai | `https://opencode.ai/inference/go/openai/v1` | 0 (sync `/models`) | Bearer + `x-opencode-session` |
| `agentrouter` | agentrouter | `https://agentrouter.org` | 5 estáticos | `x-api-key` + `anthropic-version` |
| `openrouter` | openai | `https://openrouter.ai/api/v1` | 1 (`auto`) | Bearer |
| `perplexity` | openai | `https://api.perplexity.ai` | 2 | Bearer |
| `together` | openai | `https://api.together.xyz/v1` | 3 | Bearer |
| `venice` | openai | `https://api.venice.ai/api/v1` | 1 | Bearer |
| `vercel-ai-gateway` | openai | `https://ai-gateway.vercel.sh/v1` | 3 | Bearer |
| `xai` | openai | `https://api.x.ai/v1` | 2 | Bearer |

`perplexity` declara `models_path` nulo porque não expõe `/models`; por isso `GET /providers/{id}/models`
e o import respondem `400` para ele, enquanto o chat funciona normalmente.

Preço só é preenchido quando existe na tabela canônica do 9Router; caso contrário fica `0` com
`pricing_source = 'unverified'`.

## 3. Provedores comuns aos três projetos que ficaram de fora (7)

| Slug | Motivo |
|---|---|
| `antigravity` | OAuth Google + protocolo Code Assist; exige adapter proprietário |
| `bedrock` | AWS SigV4 + tradução Messages/Converse; exige adapter dedicado |
| `gemini` | Protocolo `generateContent`, não implementado |
| `vertex` | Vertex AI com project/location e ADC; exige adapter dedicado |
| `github` | OAuth + headers específicos de Copilot (não é só API key) |
| `kilocode` | OAuth de conta Kilo |
| `kiro` | OAuth AWS Builder ID + AWS EventStream binário |

Também ficaram de fora, por não serem comuns aos três: `deepinfra`, `cloudflare-ai` (exigem
`accountId`), `zenmux`, `gigachat`, entre outros. Nenhum deles foi cadastrado para não criar entrada
de catálogo sem adapter funcional.

## 4. Divergências encontradas e decisões

| Tema | Divergência | Decisão |
|---|---|---|
| Auth do Anthropic | OmniRoute/9Router enviam `x-api-key`; o registry do 9Router declara Bearer por default | Catálogo declara `x-api-key`/`raw` explicitamente |
| Cohere | OmniRoute usa `/compatibility/v1`; 9Router usa `api.cohere.ai/v1` | `/compatibility/v1` (dialeto OpenAI) |
| Perplexity | Não expõe `/models` nas três fontes | `models_path` nulo + erro claro no teste/import |
| OpenRouter | Catálogo do 9Router lista embeddings; OmniRoute usa `auto` + passthrough | `auto` + import explícito de modelos |
| xAI | AllCode força Responses API; Omni/9R usam Chat | Chat no P0; Responses fica para o adapter de Responses |
| Preços | Só o 9Router tem tabela; OmniRoute não tem preço no registry | `pricing_source` por modelo; `unverified` quando ausente |
| Erro 402 | Todas as fontes tratam billing; o gateway mapeava para 502 | Adicionado `ErrProviderQuota` → `402 billing_error` |
| Erro 429 | 9Router classifica rate limit com backoff; OmniRoute tem lock por conta/modelo | Gateway mantém `429 rate_limit_error` e cooldown de 30 s |
| Erro 401/403 | Upstream auth | Gateway responde `502 api_error` (não vaza credencial) |

## 5. Evidências de validação local

Ambiente: PostgreSQL 17 e Redis 8 em containers locais (`dontstop-e2e-pg` em `55433`,
`dontstop-e2e-redis` em `56380`), gateway compilado de `cmd/gateway` em `127.0.0.1:18080`.

| Verificação | Resultado |
|---|---|
| `go build ./...` / `go vet ./...` | OK |
| `go test -p 1 ./...` (com `TEST_DATABASE_URL` e `TEST_REDIS_URL`) | Todas as suítes OK |
| `golangci-lint v2.15.0` (imagem oficial) | 0 issues |
| `admin`: `npm run lint`, `typecheck`, `test`, `validate` | OK (31 testes) |
| `GET /healthz` e `/readyz` | `{"status":"ok"}` / `200` |
| `GET /admin/v1/provider-catalog` | 16 entradas |
| `POST /admin/v1/provider-catalog/openrouter/connect` | `201`, `catalog_slug=openrouter`, 1 conta, 1 modelo |
| `POST /admin/v1/providers/{id}/test` (OpenRouter real) | `ok=true`, `status=200`, 437 modelos |
| Chat real (OpenRouter, `baidu/ernie-5.5-flash-preview:free`) | `200`, conteúdo `ok`, usage 30/18 |
| Streaming real (SSE) | `200`, chunks + `[DONE]` + usage |
| Chat real com modelo pago (`openai/gpt-4o-mini`) | `402` do upstream → gateway `402 billing_error` |
| Mock upstream: chat não-stream | `200`, conteúdo `1, 2, 3` |
| Mock upstream: streaming | `200`, deltas concatenados `1, 2, 3` + `[DONE]` |
| Mock upstream: 429 | Gateway `429 rate_limit_error` |
| Mock upstream: 401 (credencial errada) | Gateway `502 api_error`; teste do provedor `ok=false status=401` |
| Mock upstream: `models_path` | `200`, 3 modelos |
| Sonda sem credencial nos 13 provedores restantes do catálogo | `401` em 10; `nvidia` e `venice` responderam `200` (endpoint de modelos público); `xai` respondeu `400`; `perplexity` respondeu "does not expose a models endpoint" |

Observações da validação real:

- Modelos free do OpenRouter responderam apenas com `reasoning` e `content` vazio quando o
  `max_tokens` era pequeno; o gateway não repassa deltas de reasoning (lacuna registrada).
- OpenRouter retornou `402 Insufficient credits` para modelo pago; antes do ajuste o gateway
  devolvia `502`. O mapeamento foi corrigido e coberto por teste.

## 5.1 Fluxo no console

- **Lista de configurados** (`/providers`): vem do Admin API; provedores sem conta são filtrados. "Adicionar provedor" leva ao catálogo.
- **Catálogo** (`/providers/catalog`): é montado a partir de `GET /admin/v1/provider-catalog` e **remove** os slugs já configurados (por `catalog_slug` ou nome). Não usa mais a lista estática como fonte de itens; ela só fornece ícone/capabilities.
- **Configurar** (`/providers/configure/{slug}`): formulário com **nome da conta** e credencial; o nome do provedor vem do catálogo e é somente leitura. Envia para o proxy `POST /api/admin/provider-catalog/{slug}/connect`, que chama o Admin API. Erros `409`, `404`, `400` e `503` são exibidos ao usuário; não há falha silenciosa.
- **Detalhe por slug** (`/providers/{slug}`): se o slug já estiver configurado, abre o provedor real; se não, mostra a visão do catálogo com o botão **Configurar** (antes a rota caía em erro porque o Admin API só aceitava UUID).
- **Capacidades:** `provider_catalog_models` declara texto, visão, áudio, vídeo, tools, reasoning, streaming e embeddings; a importação copia esses campos para `provider_models.capabilities` com `metadata_source=catalog:provider_catalog_models`. Metadados dinâmicos de `/models` não são sobrescritos pela migração.
- **Modelos:** toda criação (formulário ou catálogo) importa automaticamente os modelos anunciados pelo provedor; falha de importação não bloqueia a criação. O detalhe lista os modelos com **Testar modelo**, **Ativar/Desativar**, **Importar todos os modelos** e **Testar todos os modelos**.
- **Fila de testes:** "Testar todos" cria uma execução no banco (`model_test_runs`/`model_test_run_items`) e um worker do gateway processa um modelo por vez com lease de 5 min (`FOR UPDATE SKIP LOCKED`), sem bloquear a UI. A página faz polling do progresso e reidrata a última execução ao recarregar. Só há uma execução ativa por provedor.
- Modelos inativos saem de `/v1/models` e do roteamento; importações seguintes preservam o estado desligado.
- **Metadados dos modelos:** quando o `/models` do provedor expõe preço, modalidades e parâmetros suportados (caso do OpenRouter), a importação grava esses dados em `provider_models.capabilities` com `metadata_source: "provider:/models"` e nas colunas de preço. Sem metadados na origem, o modelo fica com `pricing_known=false` e o console mostra "sem metadados" — nunca há inferência por nome.
- **Filtros e ações em lote:** a lista de modelos filtra no cliente por preço (grátis/pago/sem preço), capabilities (áudio/visão/tools/reasoning/streaming), status e busca textual, com contagem e "limpar filtros". É possível selecionar todos os filtrados e ativar/desativar em lote via `PATCH /models/bulk`, com confirmação, resultado por item e sem perder os filtros.
- **Admin API:** `GET /admin/v1/providers/{id}` com valor não-UUID retorna `404` (antes vazava erro de cast do Postgres).
- **Guia por provedor:** o detalhe e a tela de configuração apontam para `provider-setup.md#<slug>` em `docs/user-guide/<locale>/`, com instruções específicas de credencial/base URL para cada um dos 14 provedores, nos 6 idiomas.

## 6. Pendências

1. Repassar `reasoning`/`reasoning_details` no streaming (hoje só `content`).
2. Adapters para `gemini`, `bedrock`, `openai-responses`, `azure` e OAuth proprietários; só então
   cadastrar esses provedores.
3. Verificar preço e `context window` de cada modelo na documentação oficial (`pricing_source`
   permanece `unverified` até lá).
4. Decidir se 401 de upstream deve continuar `502` ou virar um erro mais específico.
5. Expor o catálogo no console (`admin/lib/providers.ts` ainda é mock estático).
6. Avaliar lock por modelo (caso MiniMax) além do lock por conta já existente.

## 7. Como reproduzir

```powershell
# infra local
docker run -d --name dontstop-e2e-pg -e POSTGRES_DB=dontstop -e POSTGRES_USER=dontstop `
  -e POSTGRES_PASSWORD=dontstop-e2e -p 55433:5432 postgres:17-alpine
docker run -d --name dontstop-e2e-redis -p 56380:6379 redis:8-alpine

# migrar e subir
$env:DATABASE_URL="postgres://dontstop:dontstop-e2e@127.0.0.1:55433/dontstop?sslmode=disable"
$env:REDIS_URL="redis://127.0.0.1:56380/0"
$env:MASTER_KEY="<base64 de 32 bytes>"
go run ./cmd/migrate
go run ./cmd/gateway
```
