# dontstop — Architecture Decision Records (ADRs)

> Fontes: `[REPO]`, `[MAP-9R]`, `[MAP-OMNI]`, `[MAP-ALLCODE]` (ver [README](README.md)).
> Formato: contexto → decisão → alternativas → consequências. Status inicial: **Proposto**.

---

## ADR-001 — Data plane em Go e control plane em TypeScript/Next.js

**Status:** Proposto
**Contexto:** o dontstop precisa sustentar 20.000 streams SSE e 1.000 RPS com overhead p95 ≤ 30 ms, mantendo um console administrativo rico. Os gateways auditados são TypeScript/Node e mostraram bom ecossistema, mas o runtime v1 do AllCode carrega um grafo grande de SDKs e transformações `[MAP-ALLCODE]`; o OmniRoute chega a 1,2M linhas e mantém dois stacks `[MAP-OMNI]`.

**Decisão:** implementar o caminho quente (proxy, streaming, cache, rate limit, failover, DLP) em **Go 1.24+**; implementar o console e as APIs administrativas em **TypeScript + Next.js 16 + React 19**.

**Alternativas:**
- Node/TS puro (Fastify): menor atrito de time, mas 50–100 KB/conexão e GC sob 20k streams.
- Rust (axum): melhor performance, maior custo de entrega e menor ecossistema de adaptadores.
- Elixir/Phoenix: ótima concorrência, fit de time/ecossistema inferior.

**Consequências:**
- (+) Memória previsível, pprof, binário único, goroutines para SSE e pools.
- (−) Duas linguagens/pipelines; necessidade de contratos OpenAPI e geração de tipos.
- **Fallback documentado:** se o time for 100% TypeScript, adotar Fastify + `undici` + worker threads, mantendo Postgres/Redis/Qdrant/NATS e as mesmas metas (aceitando 1,5–2 GiB por réplica).

---

## ADR-002 — PostgreSQL como system of record, com PgBouncer e partições

**Status:** Proposto
**Contexto:** o modelo tem entidades relacionais (tenants, providers, contas, políticas, quotas, auditoria) e precisa de RLS para multi-tenancy futura. O 9Router usou SQLite com bons fallbacks, mas o OmniRoute acumulou 222 migrações e o volume de uso pede analytics separado `[MAP-9R]` `[MAP-OMNI]`.

**Decisão:** **PostgreSQL 17** como banco de verdade, com **PgBouncer** (pool de 50 conexões por réplica), partições mensais para `requests`/`audit_events` e agregação em `usage_daily`. Migrações versionadas e testadas em CI.

**Alternativas:**
- MySQL: sem RLS/JSONB equivalentes; menos extensões.
- CockroachDB: multi-região forte, latência/custo maiores; P2.
- SQLite: excelente para single-node, mas não para HA e concorrência de escrita.

**Consequências:**
- (+) RLS, JSONB, partições, replicação, PITR.
- (−) `requests` detalhado a 1.000 RPS não cabe em OLTP; exige amostragem/agregação no P0 e **ClickHouse no P2** (> 50M linhas/mês).

---

## ADR-003 — Redis para cache exato, rate limit, locks e circuit breaker

**Status:** Proposto
**Contexto:** o caminho quente precisa de decisões em sub-milissegundo, TTL e estado compartilhado entre réplicas. Gateways auditados usaram rate limit em memória (OmniRoute) ou mutex global (9Router), ambos problemáticos em escala `[MAP-OMNI]` `[MAP-9R]`.

**Decisão:** **Redis 8 (ou Valkey 8)** como L2 de cache exato, rate limit (Lua), locks de idempotência, cooldown e estado de circuit breaker. L1 in-process opcional (TTL 5–15 s) com invalidação por pub/sub.

**Alternativas:**
- Memcached: sem Lua/persistência.
- Dragonfly: compatível e eficiente, maturidade menor; P2.
- Só in-process: não compartilha estado entre réplicas.

**Consequências:**
- (+) Latência e operações atômicas; TTL nativo.
- (−) Custo de memória (16–32 GB); Redis vira dependência crítica — exigir HA, réplica e fallback degradado (rate limit local) em caso de indisponibilidade.

---

## ADR-004 — Qdrant para cache semântico

**Status:** Proposto
**Contexto:** o cache semântico precisa de busca ANN por tenant/modelo com latência p95 ≤ 20 ms e isolamento. Gateways auditados não oferecem esse recurso de forma integrada `[MAP-9R]` `[MAP-OMNI]`.

**Decisão:** **Qdrant** com HNSW, payload `{tenant_id, model_key, prompt_hash}`, threshold inicial 0,95 e TTL de 24 h. Embeddings locais (ONNX, 384 dims) no P0; provedor de embeddings como fallback.

**Alternativas:**
- pgvector: menos um store, porém ANN/latência pior em 2M+ vetores; fallback P1.
- Redis Vector: menos componentes, mas memória alta e ANN menos maduro.
- Weaviate/Milvus: operação pesada para o P0.

**Consequências:**
- (+) Busca rápida com filtros, snapshots e HA.
- (−) Mais um componente para operar; risco de falso positivo — mitigado por threshold alto, escopo estrito e feature flag.

---

## ADR-005 — NATS JetStream para eventos e jobs assíncronos

**Status:** Proposto
**Contexto:** uso, auditoria, webhooks e agregação não podem bloquear o caminho quente. O OmniRoute usa logs/agregação em processos e o 9Router faz agregação em JS `[MAP-OMNI]` `[MAP-9R]`.

**Decisão:** **NATS JetStream** com streams `usage`, `audit`, `webhooks`, `dlp` e consumidores idempotentes. Publicação assíncrona após a resposta; batch de 1–5 s para Postgres.

**Alternativas:**
- Kafka: throughput maior, operação pesada.
- Redis Streams: simples, mas acopla ao Redis crítico.
- RabbitMQ: bom roteamento, throughput menor.

**Consequências:**
- (+) Leve, Go-native, at-least-once, replay.
- (−) Consumidores precisam ser idempotentes; monitorar lag e DLQ.

---

## ADR-006 — OpenTelemetry + backend gerenciado (sem Prometheus/Grafana)

**Status:** Aceito (restrição do operador: não usar Prometheus, Grafana, Loki, Tempo ou stack OSS auto-operada)
**Contexto:** SLOs exigem golden signals por provider/modelo e tracing de cache/DLP/failover, mas o operador não quer operar stack de observabilidade própria. O 9Router usa `console.log` extensivamente e o OmniRoute usa pino, sem tracing distribuído nativo `[MAP-9R]` `[MAP-OMNI]`.

**Decisão:** instrumentar com **OpenTelemetry** (traces, métricas, logs) e exportar via OTel Collector para um **backend gerenciado**. Padrão corporativo: **Datadog**; alternativa: **Azure Monitor/Application Insights** em ambientes Microsoft. Amostragem de 10% (100% em erros), redaction de PII e cardinalidade limitada (sem `tenant_id` como label).

**Alternativas:**
- Azure Monitor/Application Insights: ótimo em Azure, menos unificado fora.
- New Relic: APM forte, custo/lock-in.
- Elastic self-hosted: atende air-gapped, mas volta a exigir operação própria (só com decisão explícita).
- Só logs: insuficiente para SLO.

**Consequências:**
- (+) Zero operação de backend; SLOs, alertas e correlação prontos; OTel mantém portabilidade.
- (−) Custo por ingest/host e egress para SaaS; lock-in moderado; air-gapped exige backend alternativo (decisão D11).

---

## ADR-007 — Multi-tenancy por `tenant_id` + RLS (evolução de single-tenant)

**Status:** Proposto
**Contexto:** o P0 é single-tenant, mas o requisito é evoluir sem reescrita. O 9Router adicionou multitenancy depois e o OmniRoute usa políticas e escopos; ambos mostram que o custo de retrofitar é alto `[MAP-9R]` `[MAP-OMNI]`.

**Decisão:** `tenant_id` obrigatório em todas as tabelas de domínio desde o P0, com tenant padrão; RLS habilitada no P1/P2; namespaces de cache (`t:{tenant}:`), payload de vetor e prefixos de fila. Chaves de API e JWT carregam `tenant_id`; políticas e quotas são escopadas.

**Alternativas:**
- Schema por tenant: isolamento maior, migrações e conexões multiplicadas.
- Banco por tenant: isolamento máximo, custo operacional alto; P2 enterprise.
- Sem `tenant_id` até precisar: rejeitado — gera migração destrutiva.

**Consequências:**
- (+) Evolução incremental; testes negativos de vazamento desde cedo.
- (−) Overhead de coluna/índice e disciplina em todas as queries; RLS pode degradar performance se mal indexada.

---

## ADR-008 — Gerenciador de pool Webshare

**Status:** Proposto (requisito de produto)
**Contexto:** o dontstop precisa de egress diversificado e rotação de IP para provedores sensíveis a geo/reputação. O plano Webshare não está documentado no repositório (lacuna).

**Decisão:** pool gerenciado com import/sync via API Webshare, health check a cada 30 s, score (sucesso, latência, banimento), rotação sticky/round-robin/least-latency, limite de concorrência por proxy, quarentena e fallback direto. Credenciais cifradas e nunca logadas.

**Alternativas:**
- Bright Data/Oxylabs: qualidade superior, custo maior; P2.
- Self-hosted: não cobre diversidade de IPs.
- Sem proxy: não atende requisito.

**Consequências:**
- (+) Controle de egress e resiliência.
- (−) Banda pode ser o maior custo (ver 04-ANALISE-STACK §2.2); rotear por proxy só quando necessário.

---

## ADR-009 — DLP em dois estágios

**Status:** Proposto
**Contexto:** DLP no caminho quente precisa de baixa latência e cobertura de PII/segredos, incluindo streaming. O OmniRoute tem guardrails opt-in; o 9Router tem PII limitada `[MAP-OMNI]` `[MAP-9R]`.

**Decisão:** P0 com **regex + validadores** (CPF/CNPJ com dígito, Luhn, e-mail, telefone, JWT, chaves AWS) in-process; P1 com **Presidio sidecar** para NER. Ações `allow/redact/block/audit`; fail-closed para segredos de alta confiança; auditoria sem PII bruta.

**Alternativas:**
- Só Presidio: latência e operação maiores no P0.
- Só regex: não cobre nomes/endereços.
- DLP no cliente: não confiável.

**Consequências:**
- (+) Latência controlada e evolução por camadas.
- (−) Falsos positivos/negativos; exigir modo `audit` antes de `block` e métricas de qualidade.

---

## ADR-010 — Docker Swarm com réplicas e autoscaling externo

**Status:** Aceito (restrição do operador: não usar Kubernetes)
**Contexto:** 20.000 streams e 3.000 RPS de burst exigem escala horizontal e rolling update sem derrubar SSE. O operador determinou **Docker Swarm** como stack de orquestração.

**Decisão:** **Docker Swarm mode** com 3 managers (quorum Raft) e N workers; `stack.yml` versionado; redes overlay `edge`/`app`/`data` (dados com `--opt encrypted`); Docker secrets para credenciais; Traefik como ingress HTTP/2; rolling update `start-first` com `--stop-grace-period 90s` e healthchecks. O Swarm **não tem HPA nativo**: o scaling é manual via `docker service scale` (P0) ou feito por um controlador externo em Go que lê as métricas do backend gerenciado e chama a Docker API (P1).

**Alternativas:**
- Kubernetes: rejeitado por restrição explícita do operador.
- Nomad: boa alternativa, mas adiciona mais um runtime e ecossistema.
- Bare metal: menor custo, mais operação.
- Serverless: cold start e limites de duração para SSE.

**Consequências:**
- (+) Simplicidade operacional, nativo do Docker, secrets e overlay networks; menor curva de adoção.
- (−) Sem autoscaling nativo, sem CRDs/service mesh; estado (Postgres/Redis/Qdrant) exige placement em nós `tier=data`, volumes e backup; multi-região é manual (clusters separados atrás de LB global).

---

## ADR-011 — Segredos com AES-256-GCM e envelope key

**Status:** Proposto
**Contexto:** gateways auditados armazenaram credenciais em texto plano ou com criptografia opt-in; o 9Router não cifrava e o OmniRoute tinha passthrough sem chave `[MAP-9R]` `[MAP-OMNI]`.

**Decisão:** credenciais de providers e proxies cifradas com **AES-256-GCM**, IV aleatório, authTag completo e **envelope key** em KMS/Vault (ou chave mestra em arquivo 0600 no P0). API keys com **Argon2id**; chave só exibida na criação. Rotação versionada (`key_version`).

**Alternativas:**
- Criptografia opt-in: rejeitada por risco de produção.
- Hash reversível: inviável para credenciais de upstream.
- Vault obrigatório no P0: aumenta dependência; pode ser P1.

**Consequências:**
- (+) Redução de impacto de vazamento do banco.
- (−) Gestão de chaves e rotação; indisponibilidade do KMS pode bloquear decrypt — cache local com cuidado.

---

## ADR-012 — Compatibilidade OpenAI como contrato primário

**Status:** Proposto
**Contexto:** o valor do gateway é permitir que CLIs e SDKs existentes apontem para um endpoint único. Os gateways auditados adotam `/v1` compatível `[MAP-9R]` `[MAP-OMNI]`.

**Decisão:** `/v1/chat/completions` e `/v1/models` são o contrato P0, com streaming SSE e semântica OpenAI. `/v1/embeddings` e `/v1/responses` entram no P1. Contratos versionados em OpenAPI e testes com o SDK oficial.

**Alternativas:**
- API própria primeiro: exige adaptação de clientes e reduz adoção.
- gRPC nativo: melhor performance interna, mas não atende o ecossistema.

**Consequências:**
- (+) Adoção imediata por ferramentas existentes.
- (−) Necessidade de tradução fiel de erros, usage e streaming; divergências viram bugs de compatibilidade.

---

## ADR-013 — Lazy loading e cache em camadas

**Status:** Proposto
**Contexto:** o console corporativo tem gráficos, tabelas grandes, editores e logs; o gateway tem adaptadores, catálogo, embeddings e pools de proxy. Carregar tudo no primeiro paint/primeira requisição aumenta TTI, memória e custo. O cache precisa de camadas para absorber rajadas sem multiplicar ops no Redis.

**Decisão:**
- **Frontend:** code splitting por rota, `dynamic import` para gráficos/Monaco/logs, virtualização de listas, skeletons, prefetch no idle e dados com `stale-while-revalidate`. Metas: LCP ≤ 2,5 s, INP ≤ 200 ms, CLS ≤ 0,1, bundle inicial ≤ 250 KB gzip.
- **Backend:** eager apenas do caminho quente (config, auth, rate limit, provider padrão); lazy para adaptadores não padrão, catálogo completo, embeddings, sync de proxy e NER, com timeout/fail-open e telemetria.
- **Cache:** L1 in-process (5–15 s) → L2 Redis (1 h) → semântico Qdrant (≤ 24 h, 0,95) → HTTP cache para estáticos; invalidação por pub/sub e versão de prompt/modelo.

**Alternativas:**
- Carregar tudo eager: mais simples, pior TTI/memória e cold start.
- Só SSR sem cache de cliente: mais requisições e pior percepção.
- Só L2: mais ops no Redis e latência maior em rajadas.

**Consequências:**
- (+) Melhor TTI, memória e custo; cache mais eficiente.
- (−) Mais complexidade de invalidação e necessidade de testes de cold start; lazy loading não pode atrasar o primeiro token.

---

## ADR-014 — Identidade corporativa: OIDC/SAML, MFA e RBAC

**Status:** Proposto
**Contexto:** uso corporativo exige SSO, MFA, papéis e auditoria de acesso. O P0 precisa funcionar com API keys, mas a evolução para o diretório corporativo não pode exigir reescrita.

**Decisão:** P0 com API keys (Argon2id) + JWT httpOnly para o console e RBAC básico. P1 adiciona **OIDC/SAML** e **MFA**; P2 adiciona **SCIM**/sync de diretório. Papéis: `admin`, `operator`, `finops`, `auditor`, `client`. Toda sessão e mudança de papel é auditada; RBAC é aplicado por rota e testado contra IDOR.

**Alternativas:**
- Apenas API keys: simples, mas não atende SSO/MFA corporativo.
- LDAP direto: acopla ao fornecedor e ignora OIDC moderno.
- Sem RBAC: inviável para segregação de funções.

**Consequências:**
- (+) Atende procurement/segurança e segregação de funções.
- (−) Integração com IdP, mapeamento de grupos e testes de autorização; sessão precisa carregar `tenant_id` e papéis.

---

## ADR-015 — Endpoint Registry e escopos de API key

**Status:** Proposto
**Contexto:** a superfície pública (`/v1/*`) precisa de governança por endpoint: quais estão ativos, com qual auth, limites, modelos, cache, DLP e proxy. As chaves precisam de escopo por endpoint/modelo, quota, expiração, rotação e auditoria — sem hardcode em rotas.

**Decisão:**
- Endpoints e políticas vivem no Postgres (`api_endpoints`, `endpoint_policies`) e são cacheados no Redis com TTL curto; mudanças publicam invalidação por pub/sub.
- API keys usam Argon2id, prefixo identificável e escopos em `api_key_scopes` (`endpoint/model/provider`, `read/write`), com quota, expiração, IP allowlist e rotação.
- O gateway consulta endpoint + escopos **antes** do roteamento e falha fechado (401/403/404) sem chamar upstream.
- Criação/rotação/revogação e mudanças de política são auditadas; aprovação em duas etapas no P1.

**Alternativas:**
- Rotas hardcoded com políticas em código: simples, mas exige deploy para qualquer mudança e não escala o catálogo.
- Apenas API keys sem escopo: não atende menor privilégio nem auditoria por endpoint.
- Config estática em arquivo: dificulta autoatendimento e versionamento.

**Consequências:**
- (+) Governança e autoatendimento; mudanças sem deploy; menor privilégio.
- (−) Mais leituras de configuração (mitigadas por cache), necessidade de invalidação consistente e testes de autorização por endpoint.

**Referência de implementação:** [ENDPOINTS-API-MANAGER-REFERENCIAS](reference/ENDPOINTS-API-MANAGER-REFERENCIAS.md) — compara 9Router/OmniRoute e fundamenta categorias, limites e grupos.

## ADR-016 — MCP opt-in, local-only e sem túneis

**Status:** Proposto
**Contexto:** o gateway deve expor ferramentas a agentes/IDEs via MCP, mas sem ampliar a superfície de ataque nem permitir acesso externo não autenticado. 9Router mantém o MCP sempre exposto; OmniRoute oferece toggle global, três transportes e status/atividade.

**Decisão:**
- MCP **desligado por padrão** (`mcp_settings.enabled=false`); ativação, troca de transporte, restart e desativação de ferramenta são auditados.
- Transportes `stdio` (local), `SSE` e `Streamable HTTP`; HTTP exige Bearer/escopo.
- Ferramenta só é anunciada se `enabled=true` e o chamador tiver o escopo correspondente; desativar falha por escopo sem derrubar o servidor.
- Status (PID/uptime/heartbeat), atividade 24h e top ferramentas vêm do runtime; logs não guardam payload/PII.
- **Proibido túnel** (Cloudflare/Tailscale/ngrok) em todas as fases.

**Alternativas:**
- MCP sempre ligado (9Router): simples, mas exposição desnecessária.
- Somente stdio: mais seguro, porém não atende clientes remotos autenticados.
- Túnel gerenciado: conveniência, mas viola a política corporativa e amplia o risco.

**Consequências:**
- (+) Superfície desligada por padrão; menor privilégio por ferramenta; auditoria.
- (−) Mais um estado de configuração e testes de autenticação/escopo por transporte.

**Referência de implementação:** [MCP-REFERENCIAS](reference/MCP-REFERENCIAS.md) — compara 9Router/OmniRoute e fundamenta o toggle, transportes e ferramentas.

## ADR-017 — i18n sem dependência e guia versionado por página

**Status:** Proposto
**Contexto:** o console precisa de seis idiomas, onboarding no primeiro acesso, troca imediata em Settings e um guia por página/modal. O admin ainda é um mockup estático; introduzir uma biblioteca de i18n agora adicionaria dependência e atrito de build sem backend.

**Decisão:**
- Runtime próprio e pequeno (`admin/src/i18n.js`) com catálogos JSON por locale, interpolação `{var}`, `Intl.PluralRules` e `Intl` para datas/números/moeda.
- Fallback em cadeia: locale atual → inglês → chave; nunca quebra a UI.
- Preferência em `localStorage` na chave `dontstop.locale` (BCP-47); onboarding detecta `navigator.languages`; Settings aplica sem reload.
- Guia versionado em `docs/user-guide/<locale>/<slug>.md` com o mesmo slug em todos os idiomas, imagens `assets/<locale>/<slug>.png` e `alt` traduzido.
- Links de ajuda resolvidos por configuração (`repoUrl`, `branch`, `basePath`) + manifest, com fallback locale → inglês → índice.

**Alternativas:**
- `next-intl`/`react-i18next`: bom para o admin Next.js futuro, mas hoje não há app e o mockup não usa React.
- Strings por componente: simples no curto prazo, porém quebra paridade, fallback e testes.
- Wiki externa: não versiona junto do código nem permite link contextual determinístico.

**Consequências:**
- (+) Sem dependência, fallback testável, guia versionado e links determinísticos.
- (−) Migração incremental de strings antigas via `source-keys.js`; ao criar o admin Next.js, avaliar adoção de `next-intl` mantendo os mesmos catálogos/chaves.

## Fontes consultadas

- `[MAP-9R]` — fallback, cache, SQLite, multitenancy, mutex global e segredos em texto plano.
- `[MAP-OMNI]` — 380 provedores, circuit breaker/cooldown, policies, route guard, Redis opcional e dívidas de escala.
- `[MAP-ALLCODE]` — catálogo/plugins, protocolos, adaptadores, rotação sem feedback e instalação em runtime.
- `[REPO]` — confirmação de greenfield; decisões são propostas a validar.
