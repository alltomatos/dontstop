# Referências — Endpoints e API Manager no 9Router e OmniRoute

> **Escopo:** estudo conceitual. 9Router e OmniRoute n?o s?o depend?ncias de execu??o do dontstop.

> **Objetivo:** entender como os dois gateways auditados tratam a superfície de endpoints e o gerenciamento de API keys, para calibrar o desenho do dontstop.
> **Fontes:** `D:\dev\9router` (v0.5.112) e `D:\dev\OmniRoute` (v3.8.53), código real; mapeamentos em [`../MAPEAMENTO-ARQUITETURAL-9ROUTER.md`](../../MAPEAMENTO-ARQUITETURAL-9ROUTER.md) e [`../MAPEAMENTO-ARQUITETURAL-OMNIROUTE.md`](../../MAPEAMENTO-ARQUITETURAL-OMNIROUTE.md).
> **Conclusão antecipada:** nenhum dos dois tem um **Endpoint Registry** em banco com políticas por endpoint. O 9Router não tem política por endpoint; o OmniRoute tem classificação/authz por rota em código + permissões por chave em categorias de endpoint. O dontstop proposto vai além com `api_endpoints` + `endpoint_policies` + escopos por endpoint/modelo.

---

## 1. 9Router

### 1.1 Endpoints

- **Rotas hardcoded** no App Router: `src/app/api/v1/chat/completions/route.js`, `models/route.js`, `embeddings/route.js`, `messages/route.js`, `responses/route.js`, `images/generations/route.js`, `audio/*`, `videos/*`, `search/route.js`, `web/fetch/route.js`, `systemone/route.js`.
- **Classificação de acesso em código** em `src/dashboardGuard.js`:
  - `PUBLIC_API_PATHS` (`:23`), `PUBLIC_PREFIXES` (`/v1`, `/v1beta`, `/api/v1`, `/api/v1beta`, `/codex`, `/responses`) (`:39`), `ALWAYS_PROTECTED` (`:42`), `PROTECTED_API_PATHS` (`:55`) e `LOCAL_ONLY_PATHS` (`:78`).
  - `isPublicLlmApi()` (`:144`) e `canAccessPublicLlmApi()` (`:164`): acesso remoto a `/v1/*` exige API key; loopback/CLI token passa.
  - `proxy()` (`:214`) aplica local-only, always-protected, API key e dashboard auth.
- **UI de endpoint** (`src/app/(dashboard)/dashboard/endpoint/EndpointPageClient.js`): base URL, “Require API key” e criação/lista de API keys. **Não** edita rate limit, modelos, cache, DLP, proxy ou versão por endpoint. Túneis existem no produto original, mas estão **fora do escopo do dontstop**.
- **Sem versionamento/deprecação** e **sem OpenAPI explorer** por endpoint.
- **Sem rate limit de gateway por chave**: a busca no core só encontra cooldown/limites de upstream e orçamento mensal de usuário. Não há token bucket/sliding window por endpoint.

### 1.2 API management

- **Geração da chave** (`src/shared/utils/apiKey.js:8-37`): `sk-{machineId}-{keyId}-{crc8}`, com `keyId` de 6 caracteres via `Math.random()` e CRC HMAC (`API_KEY_SECRET`, com fallback fixo na linha 3).
- **Persistência**: a chave é gravada em texto plano (`src/lib/db/repos/apiKeysRepo.js:41-48`) e validada por igualdade SQL (`WHERE key = ?`, `:78`).
- **CRUD** (`src/app/api/keys/route.js`, `keys/[id]/route.js`): criar, listar, atualizar `name`/`isActive`/`access` e excluir. Há **ownership** para role `client` (`keys/[id]/route.js:7-18`), mas **não** há expiração, rotação, reveal, IP allowlist, rate limit ou escopo por endpoint.
- **Acesso por chave** (`src/shared/utils/keyAccess.js`, `src/sse/services/keyAccess.js`):
  - `accessRestricted` + `accessAllow` (máx. 200 entradas, 256 chars) — allowlist de **modelos/combos/providers** (search/fetch).
  - Matching exato e case-insensitive sobre identidade **resolvida** (`provider/model`), antes de qualquer lookup de credencial; falha fechada.
  - Combo permitido concede os membros quando alcançados **pelo combo**, não o membro chamado diretamente.
- **Tenant/usuário**: `users` tem `monthlyBudgetUsd`, `currentCycleSpentUsd`, `allowedModels`, `allowedCombos`; `src/sse/handlers/chat.js` aplica quota mensal (429) e whitelist antes do roteamento.
- **UI**: keys na página de endpoint; dashboard do cliente e admin de usuários. Não há uma tela “API Manager” dedicada.

### 1.3 Forças e fraquezas

| Forças | Fraquezas |
|---|---|
| Fail-closed na allowlist de modelos; matching por identidade resolvida | Chave em texto plano e gerador fraco (`Math.random`) |
| Isolamento por usuário/cliente e quota mensal | Sem escopo por endpoint, expiração, rotação, reveal ou rate limit por chave |
| Middleware deny-by-default e classificação explícita | Classificação de rotas hardcoded (mudança exige deploy) |
| | Sem registry/política por endpoint, versionamento ou OpenAPI por rota |

---

## 2. OmniRoute

### 2.1 Endpoints

- **Rotas de código** em `src/app/api/v1/*`, `v1beta/*`, `v0/management/*`, além de aliases (`/v1`, `/codex`, `/responses`) reescritos pelo Next.
- **Classificação/authz em código**:
  - `src/server/authz/classify.ts`: normaliza aliases (`/v1`, `/v1/v1`, `/codex`, `/v1beta`) e classifica `PUBLIC` × `CLIENT_API` × `MANAGEMENT`.
  - `src/server/authz/routeGuard.ts`: 3 tiers — `LOCAL_ONLY`, `ALWAYS_PROTECTED`, `MANAGEMENT`; rotas que spawnam processos são loopback-only.
  - `src/shared/constants/publicApiRoutes.ts`: allowlist pública separando **prefixos** de **rotas exatas** para evitar shadowing (GHSA documentado no arquivo).
  - `src/server/authz/policies/management.ts`: sessão/JWT, CLI token, access token (`read/write/admin`), API key com `manage`/`admin`, CSRF/origem.
  - `src/server/authz/accessScopes.ts`: infere `read`/`write`/`admin` por método e prefixo para access tokens; `manage`/`admin` de API key é full-access.
- **UI `/dashboard/api-endpoints`** (`src/app/(dashboard)/dashboard/api-endpoints/page.tsx` → `endpoint/ApiEndpointsTab.tsx`): é um **catálogo/explorer OpenAPI** com “Try It”, badges `loopbackOnly`, `alwaysProtected`, `internal` e filtro por tier. **Não** edita políticas por endpoint.
- **Sem tabela `api_endpoints`/`endpoint_policies`** (busca no `src/lib/db` não encontrou). A política de endpoint é código + authz.
- **Permissão por chave em categorias de endpoint**:
  - `src/shared/constants/endpointCategories.ts`: ~16 categorias (`chat`, `search`, `embeddings`, `images`, `audio`, `video`, `music`, `rerank`, `models`, `files`, `web-fetch`, `agents`, etc.) mapeadas para prefixos `/v1/*`; `resolveEndpointCategory()` e `resolveCanonicalEndpointPath()` normalizam aliases.
  - `src/shared/utils/apiKeyPolicy.ts` (`validateEndpointAccess`): se a chave tem `allowedEndpoints`, resolve a categoria do path e bloqueia fora da lista antes do handler; lista vazia = todos.
  - `src/lib/db/apiKeys.ts` guarda `allowedEndpoints`; a UI (`GeneralTab`) permite “allow all” ou selecionar categorias.
- **Classificação de modelo por endpoint**: `open-sse/services/modelEndpointPolicy.ts` decide se um modelo é `chat`, `image`, `embedding`, `rerank`, `video` ou `non-chat` a partir de `supportedEndpoints` e de regras por provider (OpenAI, OpenRouter `:batch`). Isso é usado no catálogo/projeção, não como política por rota.
- **Rate limits em camadas**:
  - Por chave: `rateLimits` (limit/window) em `src/shared/utils/apiKeyPolicy.ts` via `checkRateLimit`; TPM/RPM/mensal em `src/lib/db/keyQuota.ts` (sliding window de 60 s em tabelas) e `src/domain/keyQuota.ts` (fail-open em erro de infra).
  - Por conexão/provider: `open-sse/services/rateLimitManager.ts` com Bottleneck adaptativo, aprendendo limites de headers/429 e aplicando overrides por conexão.
  - Redis opcional; sem `REDIS_URL` cai em memória com aviso em produção (`src/shared/utils/rateLimiter.ts:1-15`).

### 2.2 API management

- **Armazenamento de chave** (`src/lib/db/apiKeys.ts:750-826`):
  - segredo cifrado (`enc:v1:`) para reveal/export administrativo;
  - `key_hash` (fingerprint SHA-256 de token de alta entropia) para autenticação;
  - `key_prefix` (12 chars) para identificação;
  - validação por hash (`:1372-1445`), com cache curto, checagem de `is_banned`, `is_active`, `revoked_at` e `expires_at`; SQLite é a verdade.
- **Criação** (`createApiKey`, `:759`): opções `modelAccessMode`, `allowedModels`, `allowedCombos`, `allowedConnections`, `preferredConnections`, `expiresAt`, `scopes`; depois `updateApiKeyPermissions` cobre o restante.
- **Rotação** (`regenerateApiKey`, `:828`): novo segredo, novo hash/prefix, invalida caches (inclusive Redis), audit `apiKey.regenerate`.
- **Reveal** (`/api/keys/[id]/reveal`): admin-only, mascarado por padrão (`isApiKeyRevealEnabled`), auditável.
- **Escopos/gestão**:
  - `manage`/`admin` para management; `mcp:connect`, `read:cache`/`write:cache`, `self:usage`, `self:account-quota`, `bypass-provider-quota` (constantes em `managementScopes.ts`/`selfServiceScopes.ts`).
  - Management auth (`requireManagementAuth`) aceita sessão, CLI loopback, access token `read/write/admin` e API key `manage`/`admin`; nunca aceita credencial na URL.
- **Modelo de permissões por chave** (UI `dashboard/api-manager`):
  - abas General, Models, Combos, Connections, Limits, Behaviour;
  - `modelAccessMode` all/restricted, `allowedModels`, `blockedModels`, padrões de família Claude Code, `allowedCombos` (inclui `ALL_COMBOS_ACCESS_RULE`), `allowedConnections`/`preferredConnections`, `allowedQuotas`, `allowedEndpoints`;
  - `isActive`, `isBanned`, `expiresAt`, `maxSessions`, `throttleDelayMs`, `rateLimits`, `accessSchedule` (dias/horário/timezone), `usageLimitEnabled` + limites diário/semanal em USD, `noLog`, `autoResolve`, `catalogScope`, `forcedModel`, `streamDefaultMode`, `compressionEnabled`, `chaosModeEnabled`.
- **Grupos de chaves** (`src/lib/db/apiKeyGroups.ts`): grupos com regras `allow`/`deny` por padrão de modelo/provider; `checkKeyModelAccess` aplica deny primeiro, depois allow de grupos ativos; chave sem grupo = irrestrita.
- **Quotas**:
  - `keyQuota.ts`: TPM/RPM/mensal USD (mensal lido do cost ledger), contadores em tabelas, janela de 60 s; `domain/keyQuota.ts` fail-open em erro de infraestrutura.
  - `usageLimitEnabled` + `dailyUsageLimitUsd`/`weeklyUsageLimitUsd` (status via `/api/keys/[id]/usage-limits`).
  - `registeredKeys.ts`: chaves `ork_` para provisionamento por provider/account, com orçamento diário/horário e limites de chaves ativas por provider/account.
- **Auditoria**: `regenerate`, criação, permissões e revogação geram eventos de compliance; listagem mascara chaves.

### 2.3 Forças e fraquezas

| Forças | Fraquezas |
|---|---|
| Hash + segredo cifrado + prefixo; validação por hash e cache | Sem Endpoint Registry em banco; política de rota em código |
| Rotação, reveal admin-only, expiração, ban, audit | Categorias de endpoint são **grossas** (prefixos), não por endpoint |
| Permissões ricas (modelos, combos, conexões, grupos, endpoints) | `apiKeys.ts` ~65 KB e UI fragmentada em muitos componentes |
| Rate limits por chave + adaptativo por conexão + quotas | Rate limit em memória se Redis não configurado |
| Access tokens com read/write/admin e management scopes | Complexidade de scopes/tiers e risco de configuração errada |

---

## 3. Comparativo direto

| Capacidade | 9Router | OmniRoute | dontstop (proposto) |
|---|---|---|---|
| Catálogo de endpoints | Rotas hardcoded | Rotas hardcoded + explorer OpenAPI | **`api_endpoints` versionado** |
| Política por endpoint | Não | Authz em código (tiers) | **`endpoint_policies` por endpoint** |
| Restrição por endpoint na chave | Não (só modelo/combo) | Categorias de endpoint (`allowedEndpoints`) | **Escopos por endpoint/modelo** |
| Formato da chave | `sk-...` com CRC; texto plano | `sk-...` + hash + segredo cifrado + prefixo | Hash + prefixo (sem reveal por padrão) |
| Rotação | Não | Sim | Sim (planejado) |
| Expiração/ban | Não | Sim | Sim |
| Rate limit por chave | Não | Sim (rateLimits + TPM/RPM/USD) | Sim (RPM/TPM/USD) |
| Agendamento de acesso | Não | Sim (dias/horário/tz) | Sim (P1) |
| Grupos de chaves | Não | Sim (allow/deny) | Sim (P1) |
| Auditoria de chave | Limitada | Sim | Sim |
| UI de API Manager | Keys na página de endpoint | Dedicada, rica | Mockup dedicado |
| Multi-tenant | Usuários/clientes | Times/grupos (sem `tenant_id` global) | `tenant_id` desde o P0 |

---

## 4. O que adotar no dontstop

1. **Chave**: hash + prefixo (nunca texto plano); se reveal for necessário, segredo cifrado + rota admin auditada (modelo OmniRoute), com decisão explícita.
2. **Rotação/expiração/ban**: regenerar invalida caches e audita; expiração e ban antes do handler.
3. **Endpoint categories**: usar como agrupamento de UX e compatibilidade, mas manter **escopo por endpoint** no registry (mais granular que o OmniRoute).
4. **Permissões por chave**: endpoints, modelos, combos, conexões, quotas e rate limits; fail-closed.
5. **Limites**: RPM/TPM/USD mensal + diário/semanal, throttle, max sessions, access schedule; contadores no Redis/DB com janela deslizante.
6. **Grupos**: allow/deny por padrão de modelo/provider, deny precedence; útil para políticas corporativas.
7. **Auditoria**: criação/rotação/reveal/revogação/mudança de política, com ator/IP.
8. **Canonicalização de path**: normalizar aliases (`/v1`, `/api/v1`, `/codex`) antes de aplicar escopo — lição direta do OmniRoute (`resolveCanonicalEndpointPath`).
9. **Fail-closed** em endpoint off/deprecado/escopo ausente; mensagem de negação não revela a allowlist.

## 5. O que evitar

- Chave em texto plano e `Math.random` (9Router).
- Classificação de rota hardcoded sem registry/política versionada.
- Rate limit apenas em memória em produção.
- Categorias de endpoint como única granularidade.
- UI de “endpoints” que é só documentação, sem governança.
- God files de API key e múltiplos caminhos de autenticação sem contrato único.

## 6. Lacuna comum e diferencial do dontstop

Nenhum dos dois tem:

- endpoint versionado em banco com status (`active/beta/off/deprecated`) e `sunset_at`;
- política por endpoint (auth, rate, modelos, cache, DLP, proxy, body/timeout, CORS) em `endpoint_policies`;
- escopo de chave por **endpoint específico** + modelo + combos;
- editor de endpoint com simulação “esta chave pode chamar este endpoint?”.

O dontstop já foi desenhado com esses elementos (ADR-015, DER 3.7, mockup de Endpoints/API Manager). A referência acima serve para calibrar os campos de API key e os limites que faltavam no desenho.

## 7. Fontes consultadas

- `D:\dev\9router`: `src/app/api/v1/*`, `src/dashboardGuard.js`, `src/shared/utils/apiKey.js`, `src/lib/db/repos/apiKeysRepo.js`, `src/shared/utils/keyAccess.js`, `src/sse/services/keyAccess.js`, `src/app/api/keys/*`, `src/app/(dashboard)/dashboard/endpoint/EndpointPageClient.js`, `src/lib/db/repos/usersRepo.js`, `src/sse/handlers/chat.js`.
- `D:\dev\OmniRoute`: `src/server/authz/{classify,routeGuard,policies/management,accessScopes}.ts`, `src/shared/constants/{publicApiRoutes,endpointCategories,managementScopes,selfServiceScopes}.ts`, `src/shared/utils/{apiKeyPolicy,rateLimiter}.ts`, `src/lib/db/{apiKeys,keyQuota,apiKeyGroups,registeredKeys}.ts`, `src/app/api/keys/*`, `src/app/(dashboard)/dashboard/{api-endpoints,api-manager}/*`, `open-sse/services/{modelEndpointPolicy,rateLimitManager}.ts`.
- Mapeamentos: `MAPEAMENTO-ARQUITETURAL-9ROUTER.md`, `MAPEAMENTO-ARQUITETURAL-OMNIROUTE.md`.
