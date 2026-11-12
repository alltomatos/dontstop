# Mapeamento consolidado de provedores — OmniRoute, 9Router e AllCode

> **Natureza do documento:** levantamento e catalogação para alimentar o catálogo versionado do
> dontstop. **Não** é integração, relay, upstream ou dependência de runtime. OmniRoute, 9Router e
> AllCode continuam sendo apenas referências de estudo (AGENTS.md §1.1 e §14).
>
> **Status:** o mapeamento permanece como levantamento. A primeira fatia do catálogo (14 provedores
> comuns aos três projetos, compatíveis com os adapters `openai`/`anthropic`) foi implementada em
> [`PROVIDER-CATALOG.md`](PROVIDER-CATALOG.md) — ver migração `0011_provider_catalog.sql`. Os demais
> itens da seção 8 continuam pendentes de aprovação.

## 0. Escopo e método

### 0.1 Fontes e versões

| Projeto | Versão | Commit | Data | Onde está o catálogo |
|---|---|---|---|---|
| OmniRoute | `3.8.53` (`@omniroute/open-sse` 3.8.53) | `191dce8d53` | 2026-11-06 | `open-sse/config/providers/registry/<id>/index.ts` + `providers/index.ts` |
| 9Router | `0.5.112` (`9router-app`) | `27b05be9` | 2026-11-06 | `open-sse/providers/registry/<id>.js` + `providers/index.js` |
| AllCode | `1.27.6` (monorepo) | `687377a2c` (branch `dev`) | 2026-11-10 | `packages/core/src/plugin/provider/*.ts` + `packages/llm/src/providers/*.ts`; modelos em `models.dev` em runtime |

### 0.2 Como os dados foram extraídos

- **OmniRoute:** execução do registry real com `tsx` (`REGISTRY` exportado por
  `open-sse/config/providers/index.ts`) → 295 entradas, 3.115 modelos, flags por modelo.
- **9Router:** execução de `open-sse/providers/index.js` (cópia isolada com `"type":"module"`) →
  `PROVIDERS`, `PROVIDER_MODELS`, `PROVIDER_OAUTH`, `PROVIDER_MEDIA`, mais preço e capabilities
  por modelo via `pricing.js` / `capabilities.js`.
- **AllCode:** leitura estática dos 37 plugins de provedor e dos 12 módulos de protocolo. O
  catálogo de modelos **não é estático** (vem de `https://models.opencode.ai`, `models-dev.ts:160`).

Nada foi inferido por nome de arquivo: cada linha das tabelas vem de um campo lido na fonte.

### 0.3 Definições usadas

| Termo | Definição neste documento |
|---|---|
| **Provedor de primeira classe** | Tem protocolo, autenticação ou roteamento próprio e exige adapter dedicado (ex.: Bedrock SigV4, Anthropic Messages, Gemini, Responses, Azure, Kiro, Cursor). |
| **OpenAI-compatible genérico** | Fala `POST /chat/completions` no dialeto OpenAI; diferenças são só base URL, API key e alguns headers. Deve ser um único adapter parametrizado por catálogo. |
| **Gateway/agregador** | Revende ou roteia modelos de terceiros (OpenRouter, Vercel AI Gateway, LLM Gateway, Kilo, OpenCode Zen/Go). |
| **Pré-configuração** | Entrada completa no catálogo (slug, protocolo, base URL, headers, capabilities, modelos, metadados). O operador informa **apenas a credencial** (e, quando inevitável, um identificador de conta/região). |
| **Não encontrado** | A informação não existe na fonte analisada. Não foi estimada. |

### 0.4 Limitações (ler antes de usar os números)

1. **IDs de modelo e preços não foram validados contra o fornecedor.** As fontes de 2026-11
   declaram famílias como `gpt-6-astra`, `claude-fable-5`, `gemini-4-argon`. O catálogo do dontstop
   deve tratá-los como *snapshot de referência* até verificação (seção 8.7).
2. **Preço só existe em uma das três bases.** O 9Router tem tabela estática (`pricing.js`, 129 IDs);
   OmniRoute **não** guarda preço no registry; AllCode delega ao `models.dev`. Onde não há preço, a
   tabela diz `nao encontrado`.
3. **Capabilities declaradas ≠ capabilities reais.** O OmniRoute declara `supportsReasoning`,
   `supportsVision`, `toolCalling`; o 9Router aplica defaults e um overlay de catálogo sincronizado.
   O AllCode não declara (deriva do SDK).
4. **Contagens por projeto divergem** por motivos legítimos (escopo, duplicidade de slug, modelos
   dinâmicos). O anexo A consolida por slug canônico.

## 1. Números do inventário

| Métrica | OmniRoute | 9Router | AllCode |
|---|---|---|---|
| Entradas de provedor | 295 (276 diretórios; alguns exportam 2 entradas) | 143 (101 com transporte de chat; 42 só mídia) | 37 plugins |
| Modelos estáticos | 3.115 | 1.181 declarados nos registries | 0 (dinâmico via `models.dev`) |
| Protocolos distintos | 9 (`openai` 271, `claude` 10, `openai-responses` 4, `gemini` 3, `antigravity` 2, `cursor` 2, `kiro` 1, `clova` 1, `custom` 1) | 13 (`openai` 78, `claude` 9, `openai-responses` 3, `gemini` 2, `ollama` 2, `antigravity`, `cursor`, `kiro`, `vertex`, `grok-web`, `perplexity-web`, `commandcode`, `gemini-cli`) | 6 protocolos próprios (`anthropic-messages`, `bedrock-converse`, `gemini`, `openai-chat`, `openai-compatible-chat`, `openai-responses`) |
| Auth | 254 apikey, 23 oauth, 10 optional, 8 none | 26 com OAuth declarado; resto API key/free | API key + OAuth (ChatGPT, Antigravity, Kilo, Kiro, OpenCode, GitLab) |
| Só mídia (imagem/TTS/STT/embed/search) | não separado no registry de chat | 42 entradas | não catalogado |

**Consolidação por slug canônico (anexo A): 362 provedores.**

| Grupo | Qtd | Uso no dontstop |
|---|---|---|
| Primeira classe (`1P`) | 50 | candidatos a adapter dedicado |
| OpenAI-compatible | 192 | um adapter parametrizado |
| Gateway/agregador | 28 | opcional (P1/P2) |
| Web/cookie/CLI reversa | 47 | **fora do escopo** |
| Mídia | 42 | fora do P0 (endpoint de chat) |
| Local/self-hosted | 3 | opcional (P2) |

**Sobreposição:** 70 IDs exatos aparecem em OmniRoute **e** 9Router; 68 hosts de base URL são
compartilhados; 21 provedores aparecem nos três projetos. OmniRoute tem 225 IDs exclusivos e o
9Router 31.

## 2. Classificação adotada

1. **Primeira classe (adapter dedicado)** — protocolo ou auth não-OpenAI. Ex.: `anthropic`,
   `gemini`, `bedrock`, `vertex`, `azure`, `cloudflare-ai`, `github`, `gitlab-duo`, `codex`,
   `xai` (Responses no AllCode), `kiro`, `cursor`, `antigravity`, `clova-studio`, `ollama`.
2. **OpenAI-compatible genérico** — mesmo adapter, catálogo parametrizado. Cobre ~192 slugs.
3. **Gateway/agregador** — mesmo adapter OpenAI, mas o catálogo é do gateway e muda sem aviso
   (exige `passthroughModels` + descoberta).
4. **Web/cookie/CLI reversa** — sem API pública estável, depende de cookie/CLI/fingerprint.
   **Não entra no catálogo do dontstop** (risco de TOS, instabilidade e segurança).
5. **Mídia** — imagem/TTS/STT/embedding/search. Fora do P0 de `/v1/chat/completions`.
6. **Local/self-hosted** — `ollama-local`, `mlx-*`. Exige allowlist de host privado no SSRF guard.

## 3. Provedores de primeira classe (perfil consolidado)

Tabela dos provedores que realmente precisam de adapter próprio e que são candidatos ao catálogo.
Contagem de modelos = catálogo estático declarado nas fontes; quando elas divergem, a faixa mostra
mínimo–máximo. Campos obrigatórios além da credencial estão em **negrito** na coluna Auth.

> **Versão de API:** só Anthropic (`2023-06-01`) e Gemini (`v1beta`) declaram versão explícita.
> Nos demais a versão é implícita no path (`/v1`, `/v2`, `/api/v3`, `/compatible-mode/v1`) ou não é
> declarada — nesse caso, `api_version` no catálogo fica `null`.

| # | Provedor | Slug | Origem | Auth (campos) | Protocolo | Base URL | Modelos | Quirks |
|---|---|---|---|---|---|---|---|---|
| 1 | Anthropic | `anthropic` | O+9+A | API key (`x-api-key`) | Anthropic Messages (`2023-06-01`) | `https://api.anthropic.com/v1/messages` | 14 (ctx 1M) | `Anthropic-Beta` tem três conjuntos distintos entre projetos; Opus 4.7+/Sonnet 5 rejeitam `temperature`/`top_p`/`top_k` |
| 2 | Anthropic (OAuth) | `claude` | O+9 | OAuth conta Claude | Anthropic Messages | `https://api.anthropic.com/v1/messages?beta=true` | 14 | Headers `X-Stainless-*`, `X-App`; fluxo OAuth em `api.anthropic.com/v1/oauth/token` |
| 3 | OpenAI | `openai` | O+9+A | API key (Bearer) | OpenAI Chat | `https://api.openai.com/v1/chat/completions` | 24 | 9Router força `stream`; famílias `gpt-5.x`/`gpt-6.x` divergem entre projetos |
| 4 | OpenAI Codex | `codex` | O+9 | OAuth ChatGPT | OpenAI Responses | `https://chatgpt.com/backend-api/codex/responses` | 52 | Headers `Version`, `OpenAI-Beta`, `originator` |
| 5 | Google Gemini | `gemini` | O+9+A | API key (`x-goog-api-key`) ou OAuth (Bearer) | Gemini `generateContent` `v1beta` | `https://generativelanguage.googleapis.com/v1beta/models` | 10–28 | Dois modos de auth; `:streamGenerateContent?alt=sse` |
| 6 | Google Vertex | `vertex` | O+9+A | API key/ADC + **project** + **location** | Gemini/Vertex | `https://{location}-aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/publishers/google/models` | 34 | Modelos MaaS usam `targetFormat: openai`; o OmniRoute tem fallback para `generativelanguage` (ver §4.13) |
| 7 | AWS Bedrock | `bedrock` | O+9+A | **AWS SigV4** (access key + secret + session token) ou **profile SSO** + **region** | Bedrock (Claude Messages) | `https://bedrock-runtime.{region}.amazonaws.com` | 9 (AllCode: dinâmico) | 9Router usa `format: claude` + assinatura por request; AllCode usa Bedrock Converse; OmniRoute declara `format: openai` |
| 8 | Azure OpenAI | `azure` | 9+A | API key + **resourceName** (+ **deployment**, `api-version`) | Azure OpenAI | `https://{resource}.openai.azure.com/openai/v1` | dinâmico | **Não existe no registry do OmniRoute**; AllCode tem variante Cognitive Services (`cognitiveservices.azure.com/openai`) |
| 9 | Cloudflare AI | `cloudflare-ai` | O+9+A | API key + **accountId** (+ **gatewayId** no AI Gateway) | Workers AI (OpenAI-compat) | `https://api.cloudflare.com/client/v4/accounts/{accountId}/ai/v1` | 9–25 | OmniRoute declara a base **sem** `/ai/v1` e monta no executor |
| 10 | GitHub Copilot | `github` | O+9+A | OAuth | OpenAI-compat | `https://api.githubcopilot.com/chat/completions` | 16 (GHE: 32) | Headers `copilot-integration-id`, `editor-version`, `X-Initiator`, `x-client-machine-id` |
| 11 | GitLab Duo | `gitlab-duo` | O (+9) | OAuth + **instanceUrl** | OpenAI-compat | `https://gitlab.com/api/v4/code_suggestions/completions` | 2 | 9Router usa `/api/v4/chat/completions` — endpoint diferente |
| 12 | xAI | `xai` | O+9+A | API key (Bearer) | OpenAI Chat (AllCode: Responses) | `https://api.x.ai/v1/chat/completions` | 6–9 | OmniRoute tem `xai-oauth`; AllCode força `sdk.responses()` |
| 13 | DeepSeek | `deepseek` | O+9 | API key | OpenAI-compat | `https://api.deepseek.com/chat/completions` | 3–8 | Base sem `/v1` no Omni/9R; AllCode usa `/v1` |
| 14 | Moonshot / Kimi | `kimi` | O+9 | API key | OpenAI-compat (Omni) / Anthropic coding (9R) | `https://api.moonshot.ai/v1/chat/completions` | 6–10 | **Mesmo slug, protocolos diferentes** entre projetos (§4.5) |
| 15 | Z.ai GLM | `glm` | O+9 | API key | OpenAI-compat (`/api/coding/paas/v4`) | `https://api.z.ai/api/coding/paas/v4/chat/completions` | 21 | OmniRoute também tem `zai` em Anthropic Messages |
| 16 | MiniMax | `minimax` | O+9 | API key | OpenAI (Omni) / Anthropic (9R) | `https://api.minimax.io/v1/chat/completions` | 6–13 | 9Router usa `/anthropic/v1/messages` + `x-api-key` |
| 17 | Alibaba Qwen | `alibaba` | O+9+A | API key | OpenAI-compat (DashScope) | `https://dashscope-intl.aliyuncs.com/compatible-mode/v1/chat/completions` | 14 | Variantes: `-cn`, `qwen-cloud`, `-token-plan`, `-coding` |
| 18 | Baidu Qianfan | `baidu` | O+9 | API key | OpenAI-compat | `https://qianfan.baidubce.com/v2/chat/completions` | 16 | OmniRoute também expõe alias `qianfan` |
| 19 | Volcengine / Doubao | `volcengine` | O+9 | API key | OpenAI-compat (Ark v3) | `https://ark.cn-beijing.volces.com/api/v3/chat/completions` | 7–9 | Variantes `coding` e `agent-plan` mudam o path |
| 20 | OpenRouter | `openrouter` | O+9+A | API key | OpenAI-compat | `https://openrouter.ai/api/v1/chat/completions` | passthrough | Headers `HTTP-Referer`, `X-Title`; catálogo muda sem aviso |
| 21 | Perplexity | `perplexity` | O+9+A | API key | OpenAI Chat + Responses (`perplexity-agent`) | `https://api.perplexity.ai/chat/completions` | 4 | Header `X-Pplx-Integration`; modelos `sonar-*` |
| 22 | Cohere | `cohere` | O+9+A | API key | OpenAI-compat (`/compatibility`) | `https://api.cohere.com/compatibility/v1/chat/completions` | 6 | 9Router usa `api.cohere.ai/v1` (host diferente) |
| 23 | Mistral | `mistral` | O+9+A | API key | OpenAI-compat | `https://api.mistral.ai/v1/chat/completions` | 9 | — |
| 24 | Groq | `groq` | O+9+A | API key | OpenAI-compat | `https://api.groq.com/openai/v1/chat/completions` | 7 | — |
| 25 | Cerebras | `cerebras` | O+9+A | API key | OpenAI-compat | `https://api.cerebras.ai/v1/chat/completions` | 3–6 | AllCode injeta `X-Cerebras-3rd-Party-Integration` |
| 26 | Together AI | `together` | O+9+A | API key | OpenAI-compat | `https://api.together.xyz/v1/chat/completions` | 7 | slug AllCode = `togetherai` |
| 27 | Fireworks | `fireworks` | O+9+A | API key | OpenAI-compat | `https://api.fireworks.ai/inference/v1/chat/completions` | 10 | IDs com prefixo `accounts/fireworks/models/` |
| 28 | NVIDIA NIM | `nvidia` | O+9+A | API key | OpenAI-compat | `https://integrate.api.nvidia.com/v1/chat/completions` | 8–11 | AllCode injeta `X-BILLING-INVOKE-ORIGIN` |
| 29 | Ollama | `ollama-cloud` / `ollama-local` | O+9 | API key / **sem auth** | OpenAI-compat / Ollama nativo | `https://ollama.com/v1/chat/completions` / `http://localhost:11434/api/chat` | 12 / 114 | Local exige allowlist de host privado |
| 30 | Vercel AI Gateway | `vercel-ai-gateway` | O+9+A | API key | OpenAI-compat | `https://ai-gateway.vercel.sh/v1/chat/completions` | 5 | slug AllCode = `vercel` |
| 31 | LLM Gateway | `llmgateway` | O+A | API key | OpenAI-compat | `https://api.llmgateway.io/v1/chat/completions` | dinâmico | Headers `HTTP-Referer`, `X-Title`, `X-Source` |
| 32 | OpenCode Zen / Go | `opencode-zen` / `opencode-go` | O+9+A | API key / OAuth (Console) | OpenAI-compat | `https://opencode.ai/zen/v1` · `https://opencode.ai/zen/go/v1` | 37 / 69 | Header `Authorization` cru; catálogo grande e instável |
| 33 | Kilo | `kilocode` / `kilo-gateway` | O+9+A | OAuth / opcional | OpenAI-compat | `https://api.kilo.ai/api/openrouter/...` · `https://api.kilo.ai/api/gateway/...` | 15 / 6 | Dois produtos distintos; slug AllCode = `kilo` |

> Catálogos de modelo completos por provedor: **anexo B** (`PROVIDERS-MODELS-PRICING.md`).
> Inventário dos 362 slugs: **anexo A** (`PROVIDERS-INVENTORY.md`).

## 4. Divergências entre OmniRoute, 9Router e AllCode

Casos em que o "mesmo" provedor aparece com configuração diferente. A coluna **canônico** é a
recomendação para o catálogo do dontstop (não necessariamente a "melhor" tecnicamente — é a que
minimiza retrabalho e mantém rastreabilidade).

| # | Provedor | OmniRoute | 9Router | AllCode | Consequência | Canônico sugerido |
|---|---|---|---|---|---|---|
| 4.1 | `anthropic` | `x-api-key`; `Anthropic-Version: 2023-06-01`; `Anthropic-Beta` derivado de uma lista base **sem** `oauth-2025-04-20` e `claude-code-20250219` (inclui `interleaved-thinking-2025-05-14`, `context-management-2025-06-27`, `prompt-caching-scope-2026-01-05`, `advanced-tool-use-2025-11-20`, `effort-2025-11-24`); `?beta=true` | `anthropic-version` + `Anthropic-Beta: claude-code-20250219,interleaved-thinking-2025-05-14`; auth efetiva `x-api-key` aplicada pelo executor (o registry declara Bearer) | SDK `@ai-sdk/anthropic`; injeta `anthropic-beta: interleaved-thinking-2025-05-14,fine-grained-tool-streaming-2025-05-14` | Três conjuntos de beta diferentes; portar `transport.auth` do 9Router sem ler o executor quebra a auth | `x-api-key` + `anthropic-version: 2023-06-01` + beta **por modelo**, declarado no catálogo |
| 4.2 | `gemini` | `authType: apikey`, `x-goog-api-key`, OAuth adicional | `apiKey → x-goog-api-key`, `oauth → Bearer` (declarado) | SDK usa `x-goog-api-key` | Pouca divergência real; OAuth é o ponto de atenção | API key `x-goog-api-key`; OAuth como capability separada |
| 4.3 | `glm` | `glm` OpenAI (`/api/coding/paas/v4`) **e** `zai` Anthropic (`/api/anthropic/v1/messages`) | `glm` Anthropic | SDK `@ai-sdk/*` (models.dev) | Mesmo fornecedor, dois protocolos e dois slugs | `glm` = OpenAI coding; `zai` = Anthropic (slug separado) |
| 4.4 | `minimax` | OpenAI `/v1/chat/completions` | Anthropic `/anthropic/v1/messages` + `x-api-key` | SDK | Payload/tool-calling incompatíveis entre as escolhas | Adotar OpenAI (mais simples) e manter `minimax-anthropic` como variante |
| 4.5 | `kimi` | `kimi` OpenAI (`api.moonshot.ai`) + `kimi-coding` Anthropic (`api.kimi.com/coding/v1/messages`) | `kimi` **é** o endpoint Anthropic de coding | models.dev | O slug `kimi` significa coisas diferentes | `moonshot` = OpenAI; `kimi-coding` = Anthropic |
| 4.6 | `kiro` | `codewhisperer.us-east-1.amazonaws.com`; 11 modelos | `runtime.us-east-1.kiro.dev` + 2 fallbacks; 10 modelos | `runtime.us-east-1.kiro.dev` | Base URL e catálogo divergem | `runtime.us-east-1.kiro.dev` com fallback para `codewhisperer` |
| 4.7 | `antigravity` | base `daily-cloudcode-pa.googleapis.com`, 19 modelos | sem `baseUrl` no registry, 0 modelos estáticos | 12 modelos com tiers `-low/-medium/-high` | Catálogo incompatível | Modelos do AllCode (tiers) + base do OmniRoute |
| 4.8 | `cohere` | `api.cohere.com/compatibility/v1` | `api.cohere.ai/v1` | SDK | Host e path diferentes | `api.cohere.com/compatibility/v1` |
| 4.9 | `xai` | OpenAI Chat | OpenAI Chat | **Responses API** (`sdk.responses`) | Wire diferente para o mesmo modelo | OpenAI Chat no P0; Responses como capability |
| 4.10 | `gitlab-duo` | `/api/v4/code_suggestions/completions` | `/api/v4/chat/completions` | `instanceUrl` + SDK `gitlab-ai-provider` | Endpoint muda o contrato | `code_suggestions` (Omni) como principal |
| 4.11 | `cloudflare-ai` | base `.../accounts` (executor completa) | `.../accounts/{accountId}/ai/v1/chat/completions` | `.../accounts/{accountId}/ai/v1` | Base declarada incompleta no Omni pode virar bug se o executor não rodar | Base completa com `{accountId}/ai/v1` |
| 4.12 | `bedrock` | `format: openai`, executor dedicado, `authHeader: bearer` | `format: claude` + `auth: aws-sigv4` | Bedrock **Converse** | Três wires distintos para o mesmo upstream | Um adapter Bedrock com SigV4 e tradução para Messages/Converse |
| 4.13 | `vertex` | `format: gemini` + `urlBuilder` que **cai em `generativelanguage.googleapis.com`** | `format: vertex`, base `aiplatform.googleapis.com` | SDK com `project`/`location` e ADC | O "vertex" do OmniRoute não é Vertex de verdade se o executor não sobrescrever | `aiplatform.googleapis.com` + project/location obrigatórios |
| 4.14 | `azure` | **ausente** | `baseUrl: ""` + `hasProviderSpecificData` | `{resource}.openai.azure.com/openai/v1` e variante Cognitive | Só é possível pré-configurar com campos dinâmicos | `{resource}.openai.azure.com/openai/v1` + deployment |
| 4.15 | `openrouter` | 1 modelo (`auto`), `passthrough` | 18 modelos (lista de **embeddings**) | SDK | Catálogo do 9R é impróprio para chat | `auto` + descoberta dinâmica |
| 4.16 | `openai` | 24 modelos (`gpt-6.1-sol`, `gpt-6-astra`…) | 36 modelos (`gpt-5.5`, `gpt-5.4`…) | models.dev | Nomes de família divergem radicalmente | models.dev verificado + aliases |
| 4.17 | `anthropic` (modelos) | 14, inclui `claude-fable-5-1` | 3 legados (`claude-sonnet-4-20250514`) | models.dev | 9Router está desatualizado nesse ponto | Lista do OmniRoute, verificada |
| 4.18 | `openai-compatible` (genérico) | existe com `baseUrl` do operador | não existe como entrada | existe como plugin | Sem isso, integração corporativa fica impossível | Manter como entrada única parametrizada |

**Outras divergências relevantes**

- **Slugs:** `featherless` (9R) vs `featherless-ai` (Omni); `commandcode` (9R) vs `command-code`
  (Omni); `qianfan` (Omni) vs `baidu` (9R); `moonshot` (Omni) vs `kimi` (9R); `xiaomi-tokenplan`
  (9R) vs `xiaomi-mimo-token-plan` (Omni); `uc-web` (9R) vs `uc` (Omni); `volcengine-ark` (9R) vs
  `volcengine` (Omni).
- **Auth declarada ≠ auth efetiva:** no 9Router o registry declara `Authorization: Bearer` como
  default, mas o executor aplica `x-api-key` para `format: claude` (`open-sse/executors/base.js:55`).
  O OmniRoute declara `authHeader` por provedor. **O catálogo do dontstop deve declarar o header
  efetivo**, não o default.
- **Campos extras:** o 9Router declara `hasProviderSpecificData` para `azure`, `bedrock`,
  `bedrock-xai`, `cloudflare-ai`, `aws-polly`, `v1m`, `xiaomi-tokenplan`. O OmniRoute não declara
  nada no registry — os campos vivem na UI/DB. Para pré-configuração, é preciso adotar a lista do
  9Router + os campos do AllCode (`resourceName`, `accountId`, `gatewayId`, `project`, `location`,
  `instanceUrl`).

## 5. Sobreposição e duplicidade

### 5.1 Entre projetos

- **70 IDs idênticos** em OmniRoute e 9Router (ex.: `anthropic`, `openai`, `gemini`, `groq`,
  `mistral`, `deepseek`, `xai`, `perplexity`, `cohere`, `openrouter`, `siliconflow`, `together`,
  `fireworks`, `nvidia`, `sambanova`, `hyperbolic`, `nebius`, `poolside`, `runcrate`, `venice`,
  `tokenrouter`, `tokenharbor`, `vertex`, `vertex-partner`).
- **68 hosts de base URL** compartilhados — bom indicador de que o cadastro deve ser único por
  host+path, não por slug.
- **21 provedores nos três projetos** (ex.: `anthropic`, `openai`, `gemini`, `vertex`, `bedrock`,
  `github`, `groq`, `mistral`, `nvidia`, `deepinfra`, `cloudflare-*`, `kilo`, `venice`, `zenmux`,
  `perplexity`, `openrouter`, `together`, `xai`, `kiro`, `antigravity`, `gitlab`).

### 5.2 Duplicidades internas que **não** devem gerar cadastro repetido

| Projeto | Slugs duplicados/variações | Regra |
|---|---|---|
| OmniRoute | `anthropic` / `claude` | mesmo upstream; `claude` = OAuth |
| OmniRoute | `antigravity` / `agy` | idênticos |
| OmniRoute | `vertex` / `vertex-partner` | mesmo host; partner = modelos MaaS |
| OmniRoute | `cursor` / `cursor-api` | OAuth vs API key |
| OmniRoute | `xai` / `xai-oauth` | API key vs OAuth |
| OmniRoute | `glm` / `glmt` / `zai` / `bigmodel` | `glm`/`glmt` OpenAI coding, `zai` Anthropic, `bigmodel` CN |
| OmniRoute | `moonshot` / `kimi` | idênticos (`api.moonshot.ai`) |
| OmniRoute | `qianfan` / `baidu` | idênticos (`qianfan.baidubce.com`) |
| OmniRoute | `qwen-cloud` / `alibaba` / `alibaba-cn` | DashScope; `-cn` = região |
| OmniRoute | `volcengine` / `doubao` | Ark; `doubao` = modelo |
| OmniRoute | `github` / `ghe-copilot` | `ghe` = GitHub Enterprise |
| OmniRoute | `cline` / `clinepass` | mesmo host, produtos diferentes |
| 9Router | `mimo-free` / `mmf` / `xiaomi-mimo` | mesmo host |
| 9Router | `alicode` / `alims-intl` / `alitp-intl` | Ali: coding, model studio, token plan |
| 9Router | `minimax` / `minimax-cn` / `minimax-code` / `-global` | região/produto |
| 9Router | `opencode` / `opencode-zen` / `opencode-go` | console vs zen vs go |

**Regra de slug canônico adotada:** `<vendor>` + sufixo de região (`-cn`, `-intl`) + sufixo de
produto/capacidade (`-coding`, `-token-plan`, `-oauth`, `-partner`). Sem apelido redundante.

## 6. Modelos e preços

- Catálogo completo por provedor: **anexo B, seção B.1** (flags de contexto, saída máxima,
  reasoning, visão, tools, preço quando existir).
- Tabela canônica de preços: **anexo B, seção B.2** (129 IDs, `$/1M tokens`, do `pricing.js` do
  9Router).
- **Nenhuma das três bases** guarda preço por *conexão*; preço é por modelo. Para o dontstop, o
  catálogo deve referenciar um `pricing_ref` versionado, não copiar preço para cada conta.

Referências úteis já verificadas na extração (preço `in/out` por 1M, `ctx`):

| Modelo | ctx | in | out | Fonte |
|---|---|---|---|---|
| `claude-opus-5` | 1.000.000 | 5 | 25 | 9R `MODEL_PRICING` |
| `claude-sonnet-5` | 1.000.000 | 2 | 10 | 9R |
| `claude-haiku-4.5` | 200.000 | 0,5 | 2,5 | 9R |
| `gpt-5.5` | 1.050.000 | 1,25 | 10 | 9R |
| `gpt-5.3-codex` | 400.000 | 1,75 | 14 | 9R |
| `gemini-3.1-pro` (via `antigravity`) | 1.048.576 | 4 | 18 | 9R caps + pricing |
| `deepseek-v4-flash` | 1.000.000 | 0,14 | 0,28 | 9R |
| `kimi-k2.7-code` | 262.144 | 0,95 | 4 | 9R (`clinepass`) |
| `glm-5.2` | 1.000.000 | 1 | 4 | 9R |
| `MiniMax-M3` | 200.000 | 0,3 | 1,2 | 9R |

> Valores apenas para priorização de esforço. **Não** cadastrar sem validar na página de preços
> do fornecedor (o `pricing.js` não tem data de coleta por linha).

## 7. Requisitos de rede, proxy e região

| Categoria | Provedores | Requisito |
|---|---|---|
| Região obrigatória | `bedrock` (`region`), `vertex` (`location` + `project`), `azure` (`resourceName`), `cloudflare-ai` (`accountId`), `databricks` (workspace), `snowflake` (`{account}`), `sap-ai-core` (deployment) | Campo de configuração, não credencial |
| Egress China (latência/bloqueio) | `glm-cn`, `minimax-cn`, `alibaba-cn`, `baidu`/`qianfan`, `tencent`, `volcengine`/`doubao`, `sensenova`, `stepfun`, `baichuan`, `yi`, `internlm`, `iflytek`, `modelscope`, `qiniu`, `moonshot` (CN) | Testar sem proxy primeiro; proxy regional só se necessário |
| Proxy residencial historicamente usado | `api-airforce`, `aihorde` e todo o grupo **web/cookie** | **Não** portar o grupo web; para os demais, o pool Webshare do dontstop resolve |
| Host privado | `ollama-local`, `mlx-gemma`, `mlx-qwen`, `comfyui`, `sdwebui` | Exige allowlist explícita no SSRF guard |
| Esquemas não-HTTP | `auggie://`, `devin://`, `zcode://`, `codex-app-server://`, `wss://` (`copilot-web`, `m365`) | **Incompatível** com adapter HTTP — fora do catálogo |
| TLS | todos os públicos | TLS 1.3/HSTS no dontstop; sem pinning nas fontes |

### 7.1 Erros, rate limit e cooldown (o que as fontes oferecem)

**9Router — `open-sse/config/errorConfig.js`**

- Mapa status → `type`/`code` no dialeto OpenAI: `400 invalid_request_error/bad_request`,
  `401 authentication_error/invalid_api_key`, `402 billing_error/payment_required`,
  `403 permission_error/insufficient_quota`, `404 model_not_found`,
  `406 model_not_supported`, `429 rate_limit_error/rate_limit_exceeded`,
  `500/502/503/504 server_error`.
- Backoff exponencial: base 2 s, teto 5 min, 15 níveis. Cooldown transitório de 30 s. Teto de
  30 min para cooldown **reportado pelo provedor** (ex.: `resets_at` de 5–6 h do Codex).
- Regras textuais (substring, case-insensitive) avaliadas antes das regras por status.

**OmniRoute — `open-sse/config/providerErrorRules.ts`**

- Regras por provedor avaliadas **antes** das globais, com dois escopos de lock:
  - `opencode`: `429` + `x-ratelimit-remaining-requests: 0` → quota **da conta** → bloquear a
    conexão inteira.
  - `minimax`: `429` + `x-model-quota-remaining: <model>=0` → quota **por modelo** → bloquear só
    aquele modelo.
- `HONORS_RULE_LOCK_SCOPE_PROVIDERS` restringe o escopo declarado a `agentrouter` + família
  `opencode`; nos demais o campo é apenas informativo.

**AllCode — `packages/llm/src/route/executor.ts`**

- Classifica erro em auth, rate limit, quota, transporte e request inválido; 2 retries com backoff
  de 500 ms → 10 s. Não há mapeamento por provedor.

**Não encontrado nas fontes:** schema de erro por fornecedor (as três bases normalizam por status
ou texto) e headers de rate limit por provedor além dos dois casos do OmniRoute.

**Consequência para o dontstop:** o feedback de cooldown descrito no AGENTS.md §8 ("ao receber
429/5xx/auth, marcar a conta em cooldown") pode ser dirigido por status + headers; o catálogo
precisa declarar se o lock é `account` ou `model` (ex.: `opencode` = account, `minimax` = model).

## 8. Recomendação para o catálogo do dontstop

### 8.1 P0 — entrar primeiro (28 entradas)

Labs e infraestrutura que cobrem ~90% do uso corporativo, com contrato estável:

`anthropic`, `claude` (OAuth), `openai`, `codex`, `gemini`, `vertex`, `bedrock`, `azure`,
`cloudflare-ai`, `github`, `gitlab-duo`, `xai`, `deepseek`, `kimi`, `glm`, `minimax`, `alibaba`,
`baidu`, `volcengine`, `openrouter`, `perplexity`, `cohere`, `mistral`, `groq`, `cerebras`,
`together`, `fireworks`, `nvidia`.

**Adapters necessários para o P0: 7**

1. OpenAI Chat (`/chat/completions`)
2. OpenAI Responses (`/responses`)
3. Anthropic Messages
4. Gemini `generateContent`
5. Bedrock (SigV4 + tradução Messages/Converse)
6. Azure OpenAI (path com `deployment` + `api-version`)
7. OpenAI-compatible parametrizado (base URL + headers + auth header/scheme por catálogo)

`github`, `cloudflare-ai`, `gitlab-duo` e `openrouter` rodam no adapter 7 com headers próprios —
não precisam de código dedicado no P0.

### 8.2 P1 — segunda onda (OpenAI-compatible estáveis, ~45)

`xiaomi-mimo`, `xiaomi-mimo-token-plan`, `zai`, `bigmodel`, `moonshot`, `sensenova`, `stepfun`,
`tencent`, `baichuan`, `yi`, `internlm`, `iflytek`, `longcat`, `siliconflow`, `deepinfra`,
`baseten`, `nebius`, `novita`, `hyperbolic`, `sambanova`, `scaleway`, `upstage`, `venice`, `writer`,
`reka`, `ai21`, `liquid`, `plamo`, `publicai`, `sarvam`, `typhoon`, `maritalk`, `inference-net`,
`monsterapi`, `nscale`, `predibase`, `runcrate`, `synthetic`, `wandb`, `bytez`, `modelscope`,
`qiniu`, `morph`, `poolside`, `arcee-ai`.

### 8.3 P2 — condicionais (agregadores, free tier e self-hosted, ~20)

`llmgateway`, `vercel-ai-gateway`, `kilo-gateway`, `kilocode`, `opencode-zen`, `opencode-go`,
`requesty`, `ollama-cloud`, `ollama-local`, `featherless`, `friendliai`, `lambda-ai`, `aihorde`,
`llm7`, `nanogpt`, `unorouter`, `apmix`, `tokenharbor`, `tokenrouter`.

> Agregadores exigem descoberta de catálogo e `passthrough`. Não pré-configurar lista fixa: marcar
> `models.mode: dynamic` e validar no primeiro uso.

### 8.4 Fora do escopo (não cadastrar)

- **47 provedores web/cookie/CLI reversa** (ex.: `chatgpt-web`, `claude-web`, `gemini-web`,
  `grok-web`, `deepseek-web`, `kimi-web`, `t3-web`, `huggingchat`, `lmarena`, `cursor`, `devin-*`,
  `auggie`, `zcode`, `zed`, `copilot-web`, `m365`, `notion-web`, `conol-web`, `promptql`,
  `hyperagent`, `twinmind`, `syntx`, `tinycms-web`, `gitlawb`, `freebuff`, `adapta-web`,
  `blackbox-web`, `chatplayground`, `notrack-web`, `yuanbao-web`, `zai-web`, `duckduckgo-web`,
  `veoaifree-web`, `codebuddy-*`, `workbuddy`, `qoder`, `trae`, `uc`).
  Motivo: sem API pública, dependência de cookie/fingerprint, violação de TOS e risco de banimento
  do IP corporativo.
- **42 provedores de mídia** no P0: ficam para a fase de imagem/áudio/embedding, não para
  `/v1/chat/completions`.
- **`omniroute` como provedor**: é o próprio gateway de referência; cadastrá-lo contradiz o
  AGENTS.md §14.

### 8.5 Pré-configuração: schema-alvo

O JSON abaixo descreve a **entrada de catálogo** (global e versionada), não as tabelas por tenant.
O operador só preenche `credentials`.

```json
{
  "slug": "anthropic",
  "display_name": "Anthropic",
  "protocol": "anthropic_messages",
  "api_version": "2023-06-01",
  "default_base_url": "https://api.anthropic.com",
  "path": "/v1/messages",
  "auth": {
    "type": "api_key",
    "header": "x-api-key",
    "scheme": "raw",
    "credential_fields": [{ "key": "api_key", "label": "API key", "secret": true }]
  },
  "headers": {
    "anthropic-version": "2023-06-01",
    "anthropic-beta": "interleaved-thinking-2025-05-14"
  },
  "capabilities": {
    "stream": true,
    "tools": true,
    "vision": true,
    "reasoning": true,
    "embeddings": false,
    "images": false,
    "audio": false
  },
  "models": { "mode": "static", "ref": "provider_catalog_models/anthropic" },
  "network": { "region_field": null, "proxy_policy": "optional", "private_host": false },
  "quirks": ["rejects_temperature_top_p_top_k_on_opus47_plus"],
  "source_ref": "omniroute@3.8.53:open-sse/config/providers/registry/anthropic/index.ts",
  "catalog_version": "2026-11"
}
```

Campos que **não** são credencial mas são obrigatórios em alguns provedores:

| Provedor | Campo | Origem na fonte |
|---|---|---|
| `azure` | `resource_name`, `deployment`, `api_version` | AllCode `azure.ts:21-26` (OpenAI) e `azure.ts:63-70` (Cognitive); 9R `hasProviderSpecificData` |
| `bedrock` | `region`, `access_key_id`, `secret`, `session_token`, `profile` | 9R `registry/bedrock.js` (`credentialForm: "aws"`) |
| `vertex` | `project`, `location` | AllCode `google-vertex.ts:5-13`; OmniRoute `vertex/index.ts:12` |
| `cloudflare-ai` | `account_id` (+ `gateway_id`) | AllCode `cloudflare-workers-ai.ts:54`; OmniRoute base |
| `gitlab-duo` | `instance_url` | AllCode `gitlab.ts:19` (`GITLAB_INSTANCE_URL`) |
| `snowflake-cortex` | `account` | OmniRoute base `{account}` |
| `databricks` | `workspace_host` | OmniRoute base `adb-xxx.azuredatabricks.net` |
| `ollama-local` | `base_url` | 9R `registry/ollama-local.js` |

**Alinhamento com o schema que já existe no repositório** (`gateway/migrations/0001`–`0008`):

| Tabela | Colunas relevantes | Papel |
|---|---|---|
| `providers` | `name`, `kind`, `origin ∈ {native, custom}`, `enabled` | instância por tenant |
| `provider_accounts` | `base_url`, `credential_cipher`, `credential_key_version`, `priority`, `weight`, `status`, `cooldown_until` | conexão/conta |
| `provider_models` | `model_id`, `display_name`, `capabilities jsonb`, `active`, `input_price_per_million`, `output_price_per_million` | modelo |

`gateway/internal/provider/adapter.go:76-87` hoje resolve apenas `openai`, `anthropic` e
`agentrouter`. Para o P0 da seção 8.1 faltam `gemini`, `openai_responses`, `bedrock`, `azure_openai`
e um `openai_compatible` parametrizado.

**Lacunas entre este mapeamento e o schema atual**

| Necessidade do mapeamento | Onde está hoje | Ação |
|---|---|---|
| protocolo separado do adapter | `providers.kind` | ampliar o `switch` do adapter e/ou coluna `protocol` |
| base URL padrão | `provider_accounts.base_url` (por conta) | catálogo fornece default; conta sobrescreve (região/proxy) |
| versão de API | inexistente | `api_version` no catálogo |
| header/scheme de auth | implícito no adapter | `auth_header` / `auth_scheme` no catálogo |
| headers fixos | implícito no adapter | `headers jsonb` |
| capabilities por modelo | `provider_models.capabilities jsonb` | padronizar chaves: `stream`, `tools`, `vision`, `reasoning`, `context_window`, `max_output_tokens` |
| preço | `input_price_per_million` / `output_price_per_million` | adicionar `cached_price_per_million` e `pricing_source` |
| rede/proxy/região | inexistente | `network jsonb` (`region_field`, `proxy_policy`, `private_host`) |
| quirks e rastreabilidade | inexistente | `quirks jsonb` + `source_ref` |

**Forma sugerida:** catálogo global e versionado, fora das tabelas por tenant:

```
provider_catalog(slug PK, display_name, protocol, api_version, default_base_url, path,
                 auth_type, auth_header, auth_scheme, headers jsonb, capabilities jsonb,
                 network jsonb, quirks jsonb, source_ref, catalog_version)

provider_catalog_models(slug, model_id, context_window, max_output_tokens,
                        supports_stream, supports_tools, supports_vision, supports_reasoning,
                        input_price_per_million, output_price_per_million,
                        cached_price_per_million, pricing_source)
```

As tabelas por tenant (`providers`, `provider_accounts`, `provider_models`) passam a referenciar o
catálogo por `slug`, preservando overrides (base URL regional, peso, prioridade, preço negociado).

### 8.6 Ordem de implementação sugerida

1. Adapter OpenAI Chat + OpenAI-compatible parametrizado (cobre ~192 slugs com um só código).
2. Adapter Anthropic Messages (`anthropic`, `claude`, `zai`, `kimi-coding`, `wafer`, `tabitoken`).
3. Adapter Gemini (`gemini`, `vertex` no modo API key).
4. Adapter OpenAI Responses (`openai`/`codex`, `perplexity-agent`, `chatgpt-web-codex` se liberado).
5. Adapter Azure OpenAI.
6. Adapter Bedrock SigV4.
7. Só então os OAuth proprietários (`github`, `gitlab-duo`, `kilocode`, `kiro`, `antigravity`,
   `cursor`) — nenhum deles é pré-requisito do P0 corporativo.

### 8.7 Checklist antes de codificar (validação de fornecedor)

- [ ] Confirmar IDs de modelo vigentes por provedor (as listas das fontes são snapshot de 2026-11).
- [ ] Confirmar `context window` e `max output` na documentação oficial.
- [ ] Confirmar preço in/out/cached na página de preços oficial.
- [ ] Confirmar formato de erro e `429`/`Retry-After` (para o feedback de cooldown por conta).
- [ ] Confirmar semântica de streaming (SSE, último chunk, `usage` no stream).
- [ ] Confirmar tool calling e formato de `tool_choice` por protocolo.
- [ ] Confirmar requisitos de região/egress dos provedores CN.
- [ ] Decidir política de proxy por provedor (direto, Webshare ou proibido).
- [ ] Aprovar o escopo P0 (28 entradas) antes de criar migração de catálogo.

## 9. Próximos passos (fora deste documento)

Este entregável é **somente levantamento**. Não foram criados adapters, migrações, seeds de
catálogo nem alterações em `gateway/`. Depois da aprovação:

1. Criar migração nova (`0011_*`) com `provider_catalog` + `provider_catalog_models` e seed a partir
   do anexo A; adicionar `cached_price_per_million` e `pricing_source` a `provider_models`.
2. Ampliar `gateway/internal/provider/adapter.go` com os kinds do P0 (8.1), um PR por adapter.
3. Implementar na ordem de 8.6, com teste de contrato gravado por protocolo.
4. Registrar preço com procedência (`pricing_source`), não duplicado por conta.
5. Atualizar `docs/05-ADRS.md` se o protocolo escolhido divergir do previsto.

## Anexos

- [`PROVIDERS-INVENTORY.md`](PROVIDERS-INVENTORY.md) — 362 slugs consolidados com origem, auth,
  protocolo, base URL, contagem de modelos e grupo.
- [`PROVIDERS-MODELS-PRICING.md`](PROVIDERS-MODELS-PRICING.md) — modelos por provedor (ctx, saída,
  flags) e tabela canônica de preços.

### Como reproduzir a extração

```powershell
# OmniRoute: dump do REGISTRY (tsx já está em node_modules)
cd D:\dev\OmniRoute
# script temporário que importa ./open-sse/config/providers/index.ts e grava JSON
.\node_modules\.bin\tsx.cmd .\.tmp-dump.mts

# 9Router: copiar open-sse para um diretório temporário com package.json {"type":"module"}
# e importar ./providers/index.js + ./providers/pricing.js + ./providers/capabilities.js
node dump.mjs

# AllCode: leitura estática
Get-ChildItem packages\core\src\plugin\provider -File
Get-ChildItem packages\llm\src\providers -File
```
