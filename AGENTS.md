# AGENTS.md — dontstop

> Guia de desenvolvimento para agentes e pessoas que vão implementar o dontstop.
> **Estado atual:** repositório greenfield — não há código de aplicação nem commits. Este arquivo descreve o alvo arquitetural e as regras de engenharia.
> **Fonte de verdade:** [`docs/`](docs/README.md). Os mapeamentos `MAPEAMENTO-*.md` são insumos locais e estão no `.gitignore`.

## 1. O que é o dontstop

Gateway corporativo de IA auto-hospedado, single-tenant no P0, com:

- endpoint OpenAI-compatible (`/v1/chat/completions`, `/v1/models`);
- catálogo de provedores e múltiplas contas com failover;
- cache exato e semântico;
- pool de proxy **Webshare**;
- DLP e governança de dados;
- quotas, custo e observabilidade;
- `tenant_id` desde o início para evoluir a multi-tenancy.

## 1.1 Refer?ncias conceituais

**9Router, OmniRoute e AllCode s?o apenas refer?ncias de estudo** de arquitetura, padr?es e formatos. Eles **n?o s?o canais, relays, upstreams, cat?logos, SDKs ou depend?ncias de execu??o** do dontstop. Nenhum adapter, pol?tica ou fluxo de chat pode delegar a esses sistemas em runtime. O cat?logo, os protocolos e os adapters s?o nativos do dontstop.

## 2. Decisões inegociáveis

| Área | Decisão |
|---|---|
| Data plane | **Go 1.24+** (`net/http`, SSE, goroutines, pprof) |
| Control plane | **TypeScript + Next.js 16 + React 19** |
| Banco | **PostgreSQL 17** + **PgBouncer**; RLS no P1 |
| Cache exato/coordenação | **Redis 8** (ou Valkey 8) |
| Cache semântico | **Qdrant** |
| Eventos | **NATS JetStream** |
| Observabilidade | **OpenTelemetry + backend gerenciado (Datadog; alternativa Azure Monitor)** — sem Prometheus/Grafana/Loki/Tempo |
| Orquestração | **Docker Swarm** — **Kubernetes está proibido** |
| Proxy | **Webshare** (import/sync/health/rotação/quarentena) |
| Segredos | AES-256-GCM + envelope key (KMS/Vault); API keys com Argon2id |
| API | OpenAI-compatible primeiro; OpenAPI versionado |
| UI/UX | `design-system/MASTER.md` + mockups; WCAG 2.2 AA; corporate navy/slate |
| Lazy loading/cache | Route-level + init sob demanda + L1/L2/semântico (ADR-013) |
| Identidade corporativa | OIDC/SAML + MFA + RBAC (ADR-014) |
| MCP | Desligado por padrão; `stdio`/`SSE`/`Streamable HTTP`; local-only, sem túneis |

Detalhes e alternativas: [`docs/05-ADRS.md`](docs/05-ADRS.md).

## 3. Documentos de referência

| Documento | Uso |
|---|---|
| [`docs/00-SUMARIO-EXECUTIVO.md`](docs/00-SUMARIO-EXECUTIVO.md) | Decisões, metas e riscos |
| [`docs/01-PRD.md`](docs/01-PRD.md) | Requisitos funcionais/não funcionais e aceite |
| [`docs/02-DER.md`](docs/02-DER.md) | Entidades, índices, RLS e retenção |
| [`docs/03-UML.md`](docs/03-UML.md) | Componentes, sequências, classes e deploy Swarm |
| [`docs/04-ANALISE-STACK.md`](docs/04-ANALISE-STACK.md) | Capacidade e justificativa da stack |
| [`docs/05-ADRS.md`](docs/05-ADRS.md) | Decisões arquiteturais |
| [`docs/06-ROADMAP-RISCOS.md`](docs/06-ROADMAP-RISCOS.md) | Fases, riscos e pendências |
| [`docs/BACKLOG.md`](docs/BACKLOG.md) | Slices verticais prontos para Issues |
| [`docs/architecture/ADD.md`](docs/architecture/ADD.md) | Documento consolidado de arquitetura |
| [`docs/07-I18N-E-USER-GUIDE.md`](docs/07-I18N-E-USER-GUIDE.md) | Idiomas, preferência, fallback, guia e links contextuais |
| [`docs/user-guide/README.md`](docs/user-guide/README.md) | Guia versionado por página/modal em seis idiomas |
| [`docs/reference/ENDPOINTS-API-MANAGER-REFERENCIAS.md`](docs/reference/ENDPOINTS-API-MANAGER-REFERENCIAS.md) | Estudo conceitual de endpoints/API manager; n?o ? depend?ncia |
| [`docs/reference/MCP-REFERENCIAS.md`](docs/reference/MCP-REFERENCIAS.md) | Estudo conceitual de MCP; não é dependência |
| [`design-system/MASTER.md`](design-system/MASTER.md) | Tokens, componentes e regras visuais |
| [`design-system/mockups/console.html`](design-system/mockups/console.html) | Mockup navegável do console |

## 4. Estrutura-alvo do repositório

```
dontstop/
├─ gateway/                 # Go: API, roteamento, cache, DLP, proxy, providers
│  ├─ cmd/gateway/
│  ├─ internal/
│  │  ├─ api/               # handlers /v1
│  │  ├─ auth/              # API keys, tenant context
│  │  ├─ endpoint/          # registry de endpoints + políticas
│  │  ├─ apikey/            # API Manager: escopos, quotas, rotação
│  │  ├─ mcp/               # servidor MCP, transportes e ferramentas
│  │  ├─ provider/          # registry, adapters, protocolos
│  │  ├─ routing/           # combos, failover, circuit breaker
│  │  ├─ cache/             # exact + semantic
│  │  ├─ dlp/               # detectors, policy
│  │  ├─ proxy/             # Webshare pool
│  │  ├─ usage/             # custo e métricas
│  │  └─ platform/          # config, logging, otel, db, redis
│  └─ migrations/           # SQL versionado
├─ admin/                   # Next.js: console e Admin API
├─ deploy/
│  ├─ stack.yml             # Docker Swarm
│  ├─ traefik/
│  └─ observability/        # OTel Collector (backend gerenciado externo)
├─ docs/
├─ design-system/           # MASTER, pages e mockups
├─ tests/                   # integração, contrato, carga
│  ├─ contract/
│  ├─ load/
│  └─ chaos/
├─ scripts/
├─ .env.example
├─ AGENTS.md
└─ README.md
```

> A estrutura é proposta; ajuste com ADR se necessário. Não crie arquivos no root fora de configuração/documentação.

## 5. Fluxo de desenvolvimento

1. **Nunca commitar direto em `main`/`master`.** Use branch `feat/`, `fix/`, `refactor/`, `docs/`, `test/`, `chore/`.
2. **Conventional Commits** (`feat(provider): ...`, `fix(cache): ...`).
3. PR pequeno, com testes e evidência de observabilidade; merge só com CI verde.
4. Mudança de arquitetura exige **ADR** em `docs/adr/` (ou atualização de `docs/05-ADRS.md`).
5. Toda mudança de comportamento atualiza o documento correspondente em `docs/`.

## 6. Comandos (quando o código existir)

```bash
# Go
go test ./...
go vet ./...
golangci-lint run
govulncheck ./...

# Admin
cd admin
npm run lint
npm test
npm run build

# Guia do usuário (raiz do repo)
node scripts/generate-user-guide.js
node scripts/capture-user-guide.js

# Docker Swarm (produção/homologação)
docker stack deploy -c deploy/stack.yml dontstop
docker service ls
docker service logs -f dontstop_gateway
docker service scale dontstop_gateway=8
docker service update --image registry/dontstop-gateway:<tag> dontstop_gateway

# Carga/chaos
k6 run tests/load/chat.js
```

> Ainda não existem scripts reais; não invente comandos. Ao criar, registre aqui.

## 7. Regras de engenharia

### Código
- **SRP:** arquivos com mais de ~250 linhas devem ser divididos por responsabilidade.
- **Sem pseudocódigo:** nada de `// ...`, `TODO` de implementação ou placeholders em código entregue.
- **DI explícita:** composição no `main`/bootstrap; proibido service locator ou singleton arbitrário.
- **Erros:** nunca engolir erro em stream; usar contexto/cancelamento; sanitizar mensagens (sem stack/PII).
- **Sem `eval`/`new Function`;** validação de entrada obrigatória.

### Banco e concorrência
- Sem SQL cru em handlers; use repositórios/módulos.
- Migrações versionadas e idempotentes; toda query com `tenant_id`.
- Índices para os acessos quentes; **proibido N+1**; pool dimensionado + PgBouncer.
- Transações curtas; batch para uso/auditoria.

### Testes
- Toda mudança de produção exige teste.
- Unidade para regra; integração para DB/Redis; contrato para protocolos de provider; carga para NFRs; caos para failover.
- Testes de segurança negativa em rotas críticas (IDOR, SSRF, tenant leak, DLP bypass).
- Cobertura mínima: 70% nas linhas críticas do gateway (PRD).

### Lazy loading e performance de UI
- Rotas com code splitting, `loading.tsx`/Suspense e skeleton.
- Charts, Monaco, logs e tabelas grandes só sob demanda; listas > 50 linhas virtualizadas.
- Backend: eager apenas do crítico (config, auth, rate limit, provider padrão); lazy com timeout/fail-open.
- Metas: LCP ≤ 2,5 s; INP ≤ 200 ms; CLS ≤ 0,1; bundle inicial ≤ 250 KB gzip.
- Nenhum módulo opcional pode atrasar o primeiro token.

### i18n e guia do usuário
- Locales suportados: `en`, `pt-PT`, `pt-BR`, `es`, `fr`, `de`; fallback obrigatório para `en`.
- Preferência em `localStorage` na chave `dontstop.locale` (string BCP-47); onboarding só no primeiro acesso; Settings aplica sem reload.
- Strings de UI sempre em `admin/locales/*.json` (ou `data-i18n`), nunca hardcoded em código novo; use `{vars}` e `Intl` para plural/data/moeda.
- Guia versionado em `docs/user-guide/<locale>/<slug>.md`, mesmo slug em todos os idiomas; imagens em `assets/<locale>/<slug>.png`; `alt` traduzido.
- Link de ajuda: `admin/src/guide-links.js` (`GUIDE_CONFIG`) + `manifest.json`; fallback locale → inglês → índice; abrir em nova aba com `noopener noreferrer`.
- Ao criar/renomear/remover página, modal ou funcionalidade, atualize inventário, docs, manifest, imagens e índices.

### Segurança (OWASP)
- Sanitização de entrada, sem SQLi/XSS/CSRF; cookies httpOnly/SameSite.
- API keys com Argon2id; credenciais com AES-256-GCM; redaction em logs/traces.
- SSRF guard e allowlist de upstream; TLS 1.3/HSTS.
- Nunca commitar `.env`, chaves, tokens ou dumps.

## 8. Regras específicas de provedores

- **Sem gateways de refer?ncia em runtime:** 9Router, OmniRoute e AllCode s?o apenas estudo. Proibido relay, upstream, import de SDK ou `base_url` apontando para esses sistemas.

- **Catálogo é fonte de verdade:** provider, modelo, capabilities, preço e protocolo vêm do catálogo versionado; nada hardcoded em handler.
- **Um adaptador por protocolo/provider:** OpenAI, Anthropic, Gemini, Azure, Bedrock e OpenAI-compatible. Protocolos proprietários exigem adaptador explícito e testes gravados.
- **Sem instalação de SDK em runtime:** dependências são empacotadas e versionadas; proibido `npm install` dinâmico.
- **Credenciais:** sempre cifradas; rotação com `key_version`; nunca logadas.
- **Failover:** conta → modelo → provider; circuit breaker e cooldown no Redis; lockout por modelo.
- **Rotação com feedback:** ao receber 429/5xx/auth, marcar a conta em cooldown (não apenas escolher a próxima).
- **Proxy:** adquirir por tentativa; sticky por conta; score por sucesso/latência; quarentena; fallback direto.
- **UI de provedores:** visão em grade ou lista (toggle persistido), filtros por auth/capability e ações em lote; detalhe em drawer.

## 8.1 Endpoints e API Manager

- **Catálogo é fonte de verdade:** `/v1/*` é definido em `api_endpoints` versionado; nada de rota hardcoded com política embutida.
- **Política por endpoint:** auth, escopos, rate limit, modelos, cache, DLP, proxy, body/timeout, CORS e logging vêm de `endpoint_policies`.
- **Config em cache:** endpoint + políticas no Redis com TTL curto e invalidação por pub/sub; Postgres é a verdade.
- **API key:** hash Argon2id + prefixo, escopo mínimo por endpoint/modelo **ativo de provedor conectado**, quota, expiração **ou “sem expiração”** (`expires_at = null`, se a política permitir), IP allowlist, rotação, max sessions e access schedule.
- **Limites por chave:** RPM/TPM/USD mensal + diário/semanal; contadores com janela deslizante; nunca só em memória em produção.
- **Categorias:** `endpointCategories` agrupa `/v1/*` para UX e compatibilidade, mas o escopo efetivo é por endpoint no registry.
- **Página Endpoint final:** URLs pública/local, servidor local, catálogo de endpoints por categoria, aliases para CLIs e system prompt opcional; **sem túneis**. A URL pública vem de `DONTSTOP_PUBLIC_BASE_URL` ou de headers `Forwarded`/`X-Forwarded-*` somente com `DONTSTOP_TRUST_PROXY=on`; `DONTSTOP_BASE_PATH` preserva o prefixo do proxy e o endpoint é normalizado para um único `/v1`. Nunca derivar o endpoint copiável de `window.location.origin`.
- **Path canônico:** normalizar `/v1`, `/api/v1`, `/v1/v1` e aliases (`/codex`, `/responses`) antes de aplicar escopo.
- **Grupos (P1):** allow/deny por padrão de modelo/provider; deny vence; chave sem grupo é irrestrita.
- **Segredo:** exibido apenas na criação; se houver reveal, rota admin auditada e desabilitada por padrão.
- **Falha fechada:** endpoint off/deprecado, escopo ausente, chave revogada/expirada/banida retornam 401/403/404 antes de chamar upstream.
- **Auditoria:** criação/rotação/reveal/revogação de chave e mudança de política sempre auditadas; aprovação em duas etapas no P1.

## 8.2 MCP

- **Desligado por padrão** (`mcp_settings.enabled=false`); ativação, troca de transporte, desativação de ferramenta e restart são auditados.
- Transportes: `stdio` (local, a IDE inicia o processo), `SSE` e `Streamable HTTP`; exibir comando/URL copiável por transporte.
- Ferramenta só é anunciada se `enabled=true` e o chamador tiver o escopo (`chat:write`, `models:read`, `embeddings:write`, `web:read`, `usage:read`).
- HTTP MCP exige Bearer/escopo; **nunca** expor sem auth. **Proibido túnel** (Cloudflare/Tailscale/ngrok).
- Status (PID/uptime/heartbeat), atividade 24h e top ferramentas vêm do runtime; logs não guardam payload/PII.
- Desativar ferramenta remove o anúncio e chamadas existentes falham por escopo; não quebrar o servidor inteiro.

## 9. Regras de cache

- Camadas: **L1 in-process** (5–15 s) → **L2 Redis** (1 h) → **semântico Qdrant** (≤ 24 h, 0,95) → **HTTP cache** para estáticos.
- Chave exata: `sha256(tenant|scope|model|normalized_messages|params|prompt_version)`.
- **Nunca** cruzar tenants; prefixo `t:{tenant}:`.
- Semântico: só respostas determinísticas (temperatura ≤ 0,2), threshold ≥ 0,95, TTL ≤ 24 h, feature flag.
- Invalidação por modelo/provider/tenant/versão de prompt; TTL padrão 1 h; L1 invalidado por pub/sub.
- Hit não pode pular DLP se a política exigir reavaliação; cache negativo com TTL curto.

## 10. Regras de DLP

- Ações: `allow`, `redact`, `block`, `audit`.
- P0: regex + validadores (CPF/CNPJ com dígito, Luhn, e-mail, telefone, JWT, chaves AWS).
- P1: Presidio sidecar para NER.
- **Nunca** logar PII bruta; guardar detector, regra, ação e hash.
- Streaming: janelas com sobreposição; `block` antes do primeiro chunk.
- Fail-closed para segredos de alta confiança; `audit` antes de `block` em produção.
- Auditoria exportável para SIEM; nunca logar payload bruto; guardar regra, detector, ação e hash.

## 11. Regras de multi-tenancy e identidade corporativa

- `tenant_id` obrigatório em todas as tabelas e no contexto.
- RLS no P1; toda query testada com tenant diferente.
- Redis: `t:{tenant}:`; Qdrant: payload `tenant_id` (P2: coleção por tenant).
- Quotas/políticas/auditoria sempre escopadas.
- Tenant padrão no P0: `00000000-0000-0000-0000-000000000000`.
- P0: proprietário do console com senha Argon2id + sessão opaca + API keys; JWT/RBAC básico; P1: OIDC/SAML + MFA; P2: SCIM. Toda sessão/mudança de papel é auditada.

## 12. Docker Swarm (não Kubernetes)

- 3 managers + workers; labels `tier=edge|app|data`.
- `stack.yml` versionado; sem `docker-compose` em produção.
- Segredos via **Docker secrets**; nunca `environment`.
- Redes overlay `edge`/`app`/`data` (`data` com `--opt encrypted`).
- Rolling update `start-first`, `--update-failure-action rollback`, `--stop-grace-period 90s`.
- **Sem HPA nativo:** scale via `docker service scale` (P0) ou controlador externo (P1). Não introduza Kubernetes, CRDs ou operators.
- **Sem Prometheus/Grafana/Loki/Tempo:** OTel Collector exporta OTLP para backend gerenciado (Datadog/Azure Monitor).
- Dados em nós `tier=data` com volumes e backup; prefira serviços gerenciados.
- Toda alteração de deploy atualiza `deploy/stack.yml` e este documento.

## 13. Definição de pronto (DoD)

- [ ] Requisito do PRD atendido e testado.
- [ ] Testes unitários + integração (e contrato, se provider) verdes.
- [ ] Sem segredo/PII em log; redaction testada.
- [ ] Métricas/tracing do novo caminho.
- [ ] Migração e índice (se DB) revisados.
- [ ] Documento em `docs/` atualizado.
- [ ] `golangci-lint`, `go vet`, `govulncheck`, lint do admin e build verdes.
- [ ] Revisão de segurança (OWASP/tenant/SSRF/DLP).
- [ ] UI: mockup/spec em `design-system/` atualizado, Lighthouse (LCP/INP/CLS) e axe sem regressão.
- [ ] Lazy loading: nenhum chunk opcional no primeiro paint; teste de cold start do caminho quente.

## 14. O que NÃO fazer

- N?o usar 9Router, OmniRoute ou AllCode como canal, relay, upstream, cat?logo ou depend?ncia de execu??o; s?o apenas refer?ncias conceituais.

- Não criar Kubernetes, Helm, CRDs, operators ou manifests K8s.
- Não adicionar Prometheus, Grafana, Loki, Tempo ou stack OSS de observabilidade; usar OTel + backend gerenciado.
- Não implementar túneis (Cloudflare Tunnel, Tailscale Funnel, ngrok) em nenhuma fase.
- Não habilitar MCP por padrão nem expor HTTP MCP sem Bearer/escopo.
- Não carregar módulos opcionais no primeiro paint nem no caminho quente sem necessidade.
- Não hardcodar políticas de endpoint em rotas; usar o Endpoint Registry.
- Não exibir nem logar API key completa após a criação.
- Não commitar mapeamentos (`MAPEAMENTO-*.md`), `.env`, segredos ou dados de clientes.
- Não usar SQLite como banco principal.
- Não instalar SDK de provider em runtime.
- Não compartilhar cache entre tenants.
- Não silenciar erro em stream nem retornar stack trace.
- Não criar arquivos de teste/script fora de `tests/` e `scripts/`.

## 15. Fontes consultadas

- `[REPO]` — estado greenfield de `D:\dev\dontstop`.
- `docs/` — PRD, DER, UML, análise de stack, ADRs, roadmap e backlog.
- `design-system/` — tokens, specs de página e mockup do console (gerado com `/ui-ux-pro-max`).
- `[MAP-9R]`, `[MAP-OMNI]`, `[MAP-ALLCODE]` — mapeamentos locais (gitignored).
