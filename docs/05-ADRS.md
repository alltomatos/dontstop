# dontstop — Architecture Decision Records (ADRs)

> **Refer?ncias conceituais:** 9Router, OmniRoute e AllCode s?o apenas estudo; n?o s?o canais, relays, upstreams, cat?logos, SDKs nem depend?ncias de execu??o.

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

## ADR-018 — Envelope P0-α e primeiro slice vertical do gateway

**Status:** Aceito
**Contexto:** a arquitetura-alvo suporta 20.000 streams SSE e 1.000 RPS (ADR-001/ADR-010), mas o repositório ainda não tem data plane. Entregar esse envelope de uma vez aumenta o risco de integração entre auth, providers, cache, streaming e usage. O primeiro ciclo precisa de uma meta mensurável, conservadora e compatível com a evolução para o alvo.

**Decisão:**
- O primeiro slice vertical (**P0-α**) entrega `/v1/chat/completions` (JSON + SSE) e `/v1/models` com **2 provedores** (OpenAI e Anthropic), API keys, failover multi-conta, cache exato Redis e usage/custo no Postgres.
- Envelope de entrega: **50 RPS sustentado**, **150 RPS de burst**, **500 streams SSE simultâneos**, **3 réplicas** do gateway, **1 região** e **single-tenant**.
- Metas: overhead do gateway **p95 ≤ 30 ms**, latência fim-a-fim **p95 ≤ 800 ms sem o tempo do provedor**, sucesso **≥ 99,5%** e disponibilidade **99,9%** mensal.
- Teste de carga em **2× o envelope** (100 RPS/300 burst) antes do GA do slice; o alvo de 1.000 RPS/20.000 streams permanece como meta de evolução, não como critério do P0-α.
- Guardrails obrigatórios: timeouts de conexão/primeiro token/total, retry somente antes do primeiro token, circuit breaker + cooldown, rate limit por chave/modelo, `Idempotency-Key` para POST não streaming, redaction de PII, payload off por padrão e outbox para usage/audit.
- O contrato público é versionado em `docs/contracts/openapi-v1.yaml`; mudanças incompatíveis exigem nova versão e ADR.

**Alternativas:**
- Entregar direto o alvo de 1.000 RPS/20.000 streams: maior risco de regressão e difícil atribuir gargalos entre auth, cache, provider e streaming.
- Slice somente `/v1/models`: reduz risco, mas não valida o caminho quente nem streaming.
- Slice sem cache/usage: simplifica, porém não valida custo, isolamento e invalidação, que são requisitos centrais do produto.

**Consequências:**
- (+) Primeiro fluxo ponta a ponta verificável, com SLOs e capacidade explícitos; base para escalar sem reescrever.
- (−) O P0-α não atende ainda o envelope corporativo completo; o roadmap deve manter os testes de 1.000 RPS/20.000 streams como gate de P1.
- (−) O envelope exige disciplina de capacidade e observabilidade desde o primeiro deploy; sem isso, a meta de 99,9% não é verificável.

## ADR-019 - Origem nativa dos provedores e adapta??o AgentRouter

**Status:** Aceito
**Contexto:** o console precisa separar provedores configurados do cat?logo e deixar expl?cita a origem do provedor. 9Router, OmniRoute e AllCode s?o apenas refer?ncias conceituais de arquitetura, padr?es e formatos; n?o s?o canais, relays, upstreams nem depend?ncias de execu??o do dontstop. O AgentRouter rejeita texto n?o ingl?s no filtro de conte?do e pode aplicar WAF ao transporte.

**Decis?o:**
- `providers.origin` ? obrigat?rio e restrito a `native` (cat?logo embutido do dontstop) ou `custom` (provedor cadastrado pelo operador); a API `/v1/models` exp?e `x-dontstop-origin`.
- Nenhum adapter, cat?logo, pol?tica ou fluxo de chat pode delegar a 9Router, OmniRoute, AllCode, relay de outro gateway ou SDK de gateway externo. Toda integra??o usa `net/http` e implementa??o nativa do dontstop.
- `kind` identifica o protocolo/adaptador nativo (`openai`, `anthropic`, `agentrouter`); `provider_accounts.base_url` aponta exclusivamente para o endpoint real do provedor.
- O kind `agentrouter` fala Anthropic Messages (`/v1/messages`, `x-api-key`, `anthropic-version`) diretamente com o provedor. Se o WAF impedir o transporte nativo, o provedor fica indispon?vel at? existir transporte nativo validado; n?o se usa relay de terceiros como contorno.
- Todo prompt ? traduzido para ingl?s antes do envio. Se a tradu??o falhar, a chamada ao AgentRouter ? bloqueada (`ErrTranslation`, HTTP 502 `translation_error`) e o motivo ? logado sem o texto do prompt.
- A resposta ? traduzida de volta ao idioma detectado no ?ltimo prompt de usu?rio; em streaming, a resposta ? acumulada e traduzida antes do primeiro chunk.
- O prompt original n?o ? mutado; ele permanece na requisi??o do servi?o para resposta, cache e rastreabilidade. Logs n?o guardam o texto bruto.

**Alternativas:**
- Fallback silencioso para o texto original: viola o requisito de nunca enviar idioma n?o traduzido.
- Traduzir chunk a chunk no streaming: pior qualidade e risco de vazamento de idioma.
- Usar relay ou gateway de refer?ncia para contornar WAF: rejeitado; cria depend?ncia de execu??o externa e viola o cat?logo/adapters nativos.

**Consequ?ncias:**
- (+) Falha expl?cita, origem audit?vel e UI consistente; nenhuma depend?ncia de gateway externo em runtime.
- (?) O AgentRouter pode ficar indispon?vel enquanto o transporte nativo n?o passar no WAF; streaming ganha lat?ncia de bufferiza??o.


## ADR-020 - Origem configurável do guia do usuário (local × GitHub)

**Contexto:** o guia era renderizado para `admin/public/guide/**` e embutido na imagem do admin; qualquer correção de texto exigia rebuild e redeploy do container.

**Decisão:** servir o guia por uma rota server-side (`GET /guide/{locale}/{slug}`) que escolhe a origem em runtime via `DONTSTOP_GUIDE_SOURCE`:
- `local` (padrão): HTML embutido em `admin/public/guide`, mantendo o fluxo de desenvolvimento atual;
- `github`: Markdown buscado em `raw.githubusercontent.com` (`DONTSTOP_GUIDE_REPO`/`DONTSTOP_GUIDE_REF`), renderizado no servidor.

A URL pública, as âncoras, as imagens e a abertura em nova aba não mudam. Em produção (`deploy/stack-vps.yml`) a fonte é `github`; `docker-compose.yml`/`.env.example` mantêm `local` como padrão. O navegador nunca acessa o GitHub diretamente e nenhum token é usado.

**Fallback:** locale solicitado → `en`; falha do GitHub → cache expirado → HTML local embutido (locale → `en`) → página amigável. A rota sempre responde `200`.

**Cache:** em memória por processo, TTL curto (`DONTSTOP_GUIDE_CACHE_TTL_SECONDS`, padrão 60 s), requisições condicionais (`If-None-Match`) e `?refresh=1` com cooldown de 10 s. O cache é por réplica; o atraso máximo é o TTL.

**Alternativas rejeitadas:**
- Fetch direto no cliente: expõe CORS, dificulta cache/fallback e permite vazar configuração.
- Rewrite via middleware: `public` tem precedência e o ambiente de runtime não é confiável no edge.
- Embutir o conteúdo a cada mudança: exige rebuild, que é exatamente o problema.

**Consequências:**
- (+) Mudanças commitadas no guia passam a valer sem novo container; conteúdo local permanece como fallback offline.
- (+) Nenhum segredo no cliente; tráfego limitado por cache/TTL.
- (?) Cada réplica pode servir conteúdo com atraso de até um TTL; o GitHub precisa estar acessível para atualizações.
- (?) O `guide:build` continua necessário para atualizar a cópia embutida (modo local e fallback).

**Documento correspondente:** `docs/reference/GUIDE-SOURCE.md`.

## ADR-021 - Proxy Webshare dedicado e sticky por conta de provedor

**Contexto:** contas de provedor compartilham o IP de saída do gateway, o que aumenta risco de bloqueio, mistura de reputação e dificulta diagnosticar falhas de rede. O Webshare oferece lista de proxies direct (IP:porta + credenciais) e um endpoint backbone com sessão sticky/rotação via username. O dontstop precisa de IP dedicado por conta, com fallback direto e sem acoplar o data plane a gateways de referência.

**Decisão:**
- Usar a **API v2/v3 do Webshare** como fonte de proxies, com API key configurada pelo console, cifrada em `webshare_settings` (AES-256-GCM, `key_version`) e nunca retornada/logada.
- Preferir **proxies direct**, persistidos em `proxy_endpoints` com credenciais cifradas (AES-256-GCM) e vínculo por conta em `proxy_assignments` (sticky de 30 min, renovável). Cada conta tem um único IP por vez; um mesmo IP pode atender **N contas do mesmo tenant** (limite configurável, padrão 3), sempre exposto na UI.
- Expor `proxy_mode` por conta: `off`, `preferred` (fallback direto) e `required` (falha fechada).
- Selecionar por **score** persistido (sucesso e falha) e **quarentena** de 5 min, com estado em Redis sob `t:{tenant}:proxy:*`; nunca apenas em memória.
- Rotacionar em falha de transporte, `407`, `429` e `5xx`; `401/403` do provedor **não** rotacionam proxy (são credencial), salvo `407` do proxy.
- Sincronizar a lista por worker periódico e sob demanda (`WEBSHARE_SYNC_INTERVAL`, padrão 1 h), com advisory lock no Postgres para não sobrepor réplicas, diff idempotente e erros sanitizados em `last_sync_*`.
- Manter isolamento por tenant em todas as tabelas e chaves Redis. Compartilhar um IP entre contas do mesmo tenant é permitido e visível; **nunca** entre tenants diferentes.

**Alternativas rejeitadas:**
- **Backbone com `-rotate` por request:** o IP muda no meio da conversa, quebra sticky e piora reputação/limites; fica apenas como opção futura por conta.
- **Rotação/score em memória (padrão 9Router):** não sobrevive a restart nem funciona com múltiplas réplicas.
- **Interceptar `globalThis.fetch` (padrão OmniRoute):** acopla transporte global, dificulta testes e vaza decisão para fora do adapter.
- **Relay types que reescrevem `base_url` (vercel/cloudflare/deno):** viola o catálogo/adapters nativos e a regra de não usar gateways de referência em runtime.
- **Proxy URL livre por tenant:** risco de SSRF e de credenciais em texto; só endpoints vindos do Webshare.

**Consequências:**
- (+) IP dedicado e estável por conta; fallback direto controlado; credenciais cifradas; observabilidade por tentativa.
- (+) Rotação e quarentena consistentes entre réplicas via Redis/DB.
- (?) Dependência operacional do Webshare (token, plano, limites de 60/min na lista e 30/min em download); indisponibilidade cai para direto em `preferred`.
- (?) Mais uma tabela e um worker no data plane; exige migração, testes de integração e caos.
- (?) Custo e limites de banda/concorrência do plano precisam ser monitorados.

**Documento correspondente:** `docs/reference/WEBSHARE-PROXY-PLAN.md`.
## ADR-022 - Failover padrão entre contas do mesmo provedor

**Status:** Aceito
**Contexto:** uma conta pode esgotar rate limit, quota ou crédito mesmo com o provedor saudável. Deixar a requisição falhar, ou trocar de provedor antes de tentar as demais contas, desperdiça capacidade já contratada e piora a latência percebida.

**Decisão:**
- O roteamento é **conta → modelo → provedor** para o modelo solicitado: todas as contas elegíveis do provedor primário são tentadas antes de outro provedor.
- Gatilhos de failover: 429, 5xx, 401/403 de credencial, quota/crédito esgotado (402/429 com `insufficient_quota`, `billing_hard_limit_reached`) e timeout/erro de transporte.
- Erros do payload do cliente (400/413/422) são **terminais**: retornam sem consumir outra conta e sem abrir circuit breaker.
- Cada falha grava cooldown no Redis em `t:{tenant}:cb:{account}:open`, com TTL específico do motivo e backoff exponencial; sucesso limpa o estado. O lockout é por conta (o modelo continua elegível em outra conta).
- Em streaming, o failover só ocorre antes do primeiro chunk; depois dele o stream é encerrado com erro sanitizado, sem reinício silencioso.
- Cada tentativa abre evento OTel no span `routing.failover` com conta, provedor, motivo, tentativa e latência; o span final registra conta original, conta selecionada, tentativas e failovers. Credenciais e PII nunca entram em atributos.

**Alternativas rejeitadas:**
- Trocar direto para outro provedor: ignora contas saudáveis do mesmo provedor e contraria a ordem de failover.
- Fazer failover em 400 do cliente: multiplica custo/latência e mascara erro de payload.
- Retry ilimitado: permite tempestade entre contas; o limite de tentativas e o breaker controlam o blast radius.

**Consequências:**
- (+) Melhor disponibilidade e uso do pool de contas; comportamento padrão sem flag.
- (+) Cooldown e tracing consistentes entre réplicas.
- (?) O limite de tentativas por requisição precisa equilibrar latência e diversidade de contas.

## ADR-023 - Combos nativos com failover na mesma requisição

**Status:** Proposto
**Contexto:** o PRD já previa rotas/combos, mas o AllCode e os gateways de referência implementam
combos de formas divergentes. O AllCode usa JSONC, dois resolvedores, `report` não ligado no caminho
V1 e failover apenas na próxima requisição. Copiar esse comportamento criaria dívida técnica e não
atenderia ao failover conta → modelo → provider já aceito no dontstop. O combo precisa ser um modelo
virtual nativo, multi-tenant, observável e consistente entre réplicas.

**Decisão:**
- Combo é uma entidade PostgreSQL por tenant (`combos` + `combo_members`), com `slug` imutável,
  `revision`, `status` (`draft`, `active`, `inactive`), estratégia, limites e membros
  `provider_id + model_id` validados contra o catálogo ativo.
- O identificador público é `combo/<slug>`. O combo ativo aparece em `/v1/models` e é consumido por
  `/v1/chat/completions`; o response mantém o modelo solicitado e expõe headers
  `X-Donstop-Combo*`.
- Estratégias P0: `priority`, `round_robin` e `priority_round_robin`, com peso apenas dentro do
  mesmo tier. Cursores e rate limit vivem no Redis por tenant/combo; não em memória.
- Failover ocorre **na mesma requisição**, antes do primeiro chunk: todas as contas do membro,
  depois o próximo membro, com `max_attempts` limitado. `400/413/422` são terminais.
- O combo usa as contas e o circuit breaker existentes; não cria credenciais, contas ou providers.
- A criação assistida por IA é conversacional, usa ferramentas read-only sobre o catálogo real,
  produz um rascunho validado e **nunca** ativa ou salva o combo automaticamente.
- Toda alteração é auditada e incrementa `revision`; cache e listas são invalidados.
- AllCode, 9Router e OmniRoute permanecem apenas referências de estudo; nenhum runtime, relay,
  upstream, SDK ou catálogo externo é usado.

**Alternativas rejeitadas:**
- Persistir em JSONC como o AllCode: não isola tenant, não versiona, não escala e não audita.
- Failover só na próxima requisição: piora disponibilidade e contraria o FR-14.
- Cursores em memória: quebram com múltiplas réplicas.
- Combo de combo: recursão, custo imprevisível e UX confusa no P0.
- IA com escrita direta ou ativação automática: risco de configuração incorreta e perda de controle.
- Reutilizar o gateway do AllCode: viola a regra de independência de runtime.

**Consequências:**
- (+) Modelo de combo explícito, testável, multi-tenant e compatível com OpenAI.
- (+) Failover mais forte que a referência, com limite de tentativas e tracing por membro.
- (+) IA útil para descobrir o catálogo sem remover a revisão humana.
- (?) Mais tabelas, endpoints, migrações, cache e testes de concorrência.
- (?) O custo/latência por membro depende de telemetria confiável; no P0 preço/contexto são
  filtros e latência/custo são warnings.

**Documentos correspondentes:** [`reference/COMBOS.md`](reference/COMBOS.md),
[`reference/COMBOS-ALLCODE-REFERENCE.md`](reference/COMBOS-ALLCODE-REFERENCE.md),
[`design-system/pages/combos.md`](../design-system/pages/combos.md).

## ADR-024 - Isolamento de IP por provedor, modo `auto` e poison persistente

**Status:** Proposto
**Data:** 2026-11-13
**Supersede (parcial):** [ADR-021](#adr-021---proxy-webshare-dedicado-e-sticky-por-conta-de-provedor) — a regra "um mesmo IP pode atender N contas do tenant" passa a valer apenas **entre provedores diferentes**; dentro do mesmo provedor o IP passa a ser exclusivo.

**Contexto:** o risco real é um IP de saída ser **restrito/banido por um provedor**. O modelo anterior permitia até 3 contas por IP dentro do tenant, o que gera banimento cruzado entre contas do mesmo provedor. O Webshare entrega outro IP automaticamente quando um proxy falha, e o gateway precisa detectar isso. O plano tem 100 IPs; reutilizar o mesmo IP em provedores distintos é seguro (o banimento é por provedor) e faz o pool render mais.

**Decisão:**
- **Invariante de isolamento:** um endpoint atende no máximo **1 conta por provedor**; reuso entre provedores é permitido. `proxy_assignments` ganha `provider_id` (desnormalizado) e `UNIQUE (tenant_id, provider_id, proxy_endpoint_id)`.
- **Modo `auto` híbrido:** `providers.proxy_auto boolean` (default do provedor). Conta tem `proxy_mode` em `inherit|off|manual|auto`. Modo efetivo: `inherit` → `auto` se `proxy_auto`, senão `off`.
- **Modo `manual`:** a conta usa um endpoint fixo (1 por provedor); **sem fallback direto silencioso** — falha fechada para não expor o IP real do gateway.
- **Legado:** `preferred`/`required` continuam legíveis e são normalizados para `manual` em escrita; migração única reescreve as linhas existentes. Elimina o fallback direto que vaza a saída da plataforma.
- **Pool do `auto`:** `Candidates()` filtra `source='webshare'`, `valid`, não envenenado, não quarentenado e sem outra conta **do mesmo provedor**. IP fixo por conta até falhar (sticky renovado).
- **Poison persistente por (provedor, endpoint):** nova tabela `proxy_poison (tenant_id, provider_id, proxy_endpoint_id, banned_until, reason)`. Classificação: `proxy_unreachable` (connect/timeout/TLS, 15 min), `proxy_auth` (407, 1 h), `upstream_ban` (403/429 com sinal de ban, 24 h) e `upstream_quota` (429 sem sinal, **sem poison**). ADR-021 preservado: 401/403 de credencial **não** rotacionam proxy.
- **Re-sync Webshare na falha:** o coordinator dispara refresh assíncrono com debounce (30 s, lock Redis `t:{tenant}:proxy:refresh`) que re-lista os IPs e importa substitutos; a rotação imediata usa o pool local. O IP morto é substituído para falhas futuras.
- **Rotação transparente:** `canRotate` aceita falha de proxy e `upstream_ban`; re-tenta na mesma request (máx. 2), antes do primeiro chunk.

**Alternativas rejeitadas:**
- **1:1 global (uma conta por IP no tenant):** desperdiça o pool (100 IPs para N×M contas) e é mais restritivo que o risco real (ban é por provedor).
- **Fallback direto automático (`preferred`):** expõe o IP real do gateway ao provedor e pode banir a saída da plataforma inteira.
- **Manter o IP no pool após falha:** gera exatamente o banimento cruzado que se quer eliminar.
- **Poison global por endpoint:** impediria o reuso entre provedores, que é seguro e desejável.

**Consequências:**
- (+) Contas do mesmo provedor em IPs distintos; falha de um IP não derruba as demais.
- (+) Pool rende mais (reuso cross-provider) e o Webshare continua sendo a única fonte de proxies.
- (+) Fail-closed elimina vazamento do IP real em fallback.
- (?) Quebra de compatibilidade: atribuições que compartilham IP dentro do mesmo provedor precisam ser desdobradas por migração.
- (?) O re-sync depende do Webshare e precisa de debounce/backoff para não virar tempestade de chamadas.
- (?) Mais uma tabela (`proxy_poison`), uma coluna (`provider_id`), uma migration e testes de concorrência.

**Documentos correspondentes:** [`reference/WEBSHARE-PROXY-PLAN.md`](reference/WEBSHARE-PROXY-PLAN.md), [`architecture/ADD.md`](architecture/ADD.md), [`02-DER.md`](02-DER.md), [`design-system/pages/proxy-network.md`](../design-system/pages/proxy-network.md), [`design-system/pages/providers.md`](../design-system/pages/providers.md).

## ADR-025 - Sidecar relay vendorizado para o AgentRouter (WAF de fingerprint)

**Status:** Aceito
**Data:** 2026-11-13

**Contexto e decisão:** o AgentRouter bloqueia clientes genéricos por fingerprint de conexão
(`401 unauthorized client detected`); só o SDK Python `anthropic` síncrono passa. O dontstop vendoriza
o relay MIT do AllCode como **sidecar interno** (rede `app`, sem porta pública, sem túnel, sem `pip` em
runtime), com a chave repassada via `x-api-key` pelo gateway (cofre cifrado, `tenant_id`, auth e
políticas do Endpoint Registry), tradução **fail-closed** no gateway, catálogo estático atualizado e
disponibilidade dinâmica (402/503). Emenda a regra "sem relay" restrita ao AgentRouter.

**Detalhe completo:** [`docs/adr/0025-agentrouter-relay-sidecar.md`](adr/0025-agentrouter-relay-sidecar.md).

## Fontes consultadas

- `[MAP-9R]` — fallback, cache, SQLite, multitenancy, mutex global e segredos em texto plano.
- `[MAP-OMNI]` — 380 provedores, circuit breaker/cooldown, policies, route guard, Redis opcional e dívidas de escala.
- `[MAP-ALLCODE]` — catálogo/plugins, protocolos, adaptadores, rotação sem feedback e instalação em runtime.
- `[REPO]` — confirmação de greenfield; decisões são propostas a validar.
