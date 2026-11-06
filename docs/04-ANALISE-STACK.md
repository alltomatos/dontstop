# dontstop — Análise de stack e capacidade

> Fontes: `[REPO]`, `[MAP-9R]`, `[MAP-OMNI]`, `[MAP-ALLCODE]` (ver [README](README.md)).
> **Recomendação única:** data plane em **Go**, control plane em **TypeScript/Next.js**, **PostgreSQL 17**, **Redis 8**, **Qdrant**, **NATS JetStream**, **OpenTelemetry + backend gerenciado (Datadog/Azure Monitor)**, **Docker Swarm** e **Webshare**.

## 1. Requisitos que guiam a decisão

| Requisito | Meta | Implicação |
|---|---|---|
| Concorrência | 20.000 SSE; 5.000/réplica | Runtime com memória por conexão baixa e agendamento eficiente |
| Throughput | 1.000 RPS; 3.000 burst | Pipeline assíncrono, pool de conexões e backpressure |
| Latência | overhead p95 ≤ 30 ms | Cache e rate limit em memória/Redis; evitar hops desnecessários |
| Cache | exato ≥ 25%; semântico +15% | Dois stores especializados + embeddings |
| Failover | RTO ≤ 2 s | Estado de saúde distribuído e decisão local rápida |
| Proxy | Webshare com rotação e health | Gerenciador de pool com score e quarentena |
| DLP | p95 ≤ 20–60 ms | Regex in-process + NER opcional em sidecar |
| Multi-tenancy | hooks agora, isolamento depois | `tenant_id`, namespaces, RLS |
| Observabilidade | 100% trace-id, golden signals | OTel desde o P0 |

## 2. Cálculos de capacidade (premissas explícitas)

> **Premissa:** 1.000 RPS sustentado, 3.000 RPS por 60 s, duração média de stream de 20 s.
> Pela Lei de Little: `concorrência = throughput × duração` → `1.000 × 20 = 20.000` streams.

### 2.1 Memória e réplicas

| Item | Cálculo | Resultado |
|---|---|---|
| Memória por conexão SSE (Go) | ~16 KB buffers + ~8 KB stack | ~24 KB/conexão |
| 5.000 conexões/réplica | 5.000 × 24 KB | ~120 MB |
| Heap/GC/caches/buffers | margem 3–4× | ~400–600 MB |
| Limite por réplica | 2 GiB | folga para picos, DLP e variação de GC |
| Réplicas mínimas | 20.000 / 5.000 | 4 réplicas |
| Réplicas de pico | burst + HA | 8–12 réplicas |
| Comparação Node/TS | 50–100 KB/conexão | 250–500 MB só de conexões; 1,5–2 GiB/réplica |

### 2.2 Rede

| Item | Cálculo | Resultado |
|---|---|---|
| Payload médio | 8 KB request + 32 KB response | 40 KB/req |
| Sustentado | 1.000 × 40 KB | 40 MB/s ≈ 320 Mbps |
| Burst | 3.000 × 40 KB | 120 MB/s ≈ 960 Mbps |
| Provisionamento | 2× o burst | 2 Gbps de egress |
| Proxy | se 30% do tráfego for via Webshare | ~96 Mbps ≈ 31 TB/mês |

> **Alerta de custo:** 31 TB/mês de banda via proxy é material. A política deve rotear por proxy **somente** provedores que exigem geo/IP dedicado; o restante deve sair direto.

### 2.3 CPU

| Carga | Cálculo | Resultado |
|---|---|---|
| Proxy/streaming | majoritariamente I/O | 2–4 vCPU/réplica |
| Tradução JSON | 0,2–1 ms/req × 1.000 RPS | 0,2–1 vCPU |
| DLP regex | 1–5 ms/req × 1.000 RPS | 1–5 vCPU (distribuído) |
| Embeddings (semântico) | 600 req/s × 5–10 ms | 3–6 vCPU ou GPU/serviço dedicado |
| Recomendação por réplica | 4 vCPU / 2 GiB | 4–12 réplicas |

### 2.4 Redis

| Item | Cálculo | Resultado |
|---|---|---|
| Exact cache | 1 GET + 1 SET por req | ~2.000 ops/s |
| Rate limit | 1–2 Lua/req | ~2.000 ops/s |
| Total | < 5.000 ops/s | 1 nó cluster suporta com folga |
| Memória | 1M entradas × 8 KB × 1,5 | ~12 GB; provisionar 16–32 GB |
| HA | réplica + sentinel/cluster | RTO < 30 s |

### 2.5 PostgreSQL

| Item | Cálculo | Resultado |
|---|---|---|
| Conexões | 50/réplica × 8 réplicas | 400 → **PgBouncer obrigatório** |
| Escrita de uso | não síncrona; lote de 1–5 s | ~1.000 linhas/s em batch |
| Metadados | providers/keys/políticas | poucos GB |
| `requests` detalhado | 1.000 RPS × 1,5 KB | ~130 GB/dia (inviável em Postgres puro) |
| Decisão | P0: agregados + amostragem; P2: ClickHouse para detalhe | ver ADR-002 e seção 8 |

### 2.6 Qdrant (cache semântico)

| Item | Cálculo | Resultado |
|---|---|---|
| Vetores | 2M × 384 dims × 4 B | ~3,1 GB |
| Índice HNSW | overhead 1,5–2× | ~5–6 GB |
| Payload/metadados | ~1 GB | ~7 GB |
| RAM recomendada | 16 GB/nó, 3 nós | HA e headroom |
| Latência alvo | top-3 com filtro por tenant | p95 ≤ 20 ms |

### 2.7 NATS / filas

| Item | Cálculo | Resultado |
|---|---|---|
| Eventos | 1.000/s × ~1 KB | ~1 MB/s |
| Retenção | 7 dias para replay | dezenas de GB |
| NATS JetStream | 3 nós | suficiente para P0/P1 |

### 2.8 Ordem de grandeza de infraestrutura

> **Suposição:** nuvem pública, 1 região, 3 AZs. Valores em USD são estimativas de ordem de grandeza, não cotação.

| Componente | Dimensionamento | Faixa mensal estimada |
|---|---|---|
| Réplicas do gateway | 8 × (4 vCPU/2 GB) | US$ 800–1.600 |
| Postgres gerenciado | 4 vCPU/16 GB + réplica | US$ 600–1.200 |
| Redis gerenciado | 16 GB + réplica | US$ 300–700 |
| Qdrant | 3 × 16 GB | US$ 600–1.200 |
| NATS | 3 pequenos | US$ 150–400 |
| Observabilidade | backend gerenciado (Datadog/Azure Monitor) + OTel Collector | US$ 400–1.200 |
| Webshare | plano a confirmar | US$ 300–1.500 |
| **Total** | | **~US$ 3.250–7.800/mês** |

## 3. Comparação de linguagens/runtime para o data plane

Pesos: concorrência/memória 25%, streaming 20%, ecossistema 15%, velocidade de entrega 15%, operação 10%, fit com o time 10%, segurança de memória 5%. Notas de 0–10 (julgamento de engenharia, não benchmark).

| Runtime | Concorr. | Streaming | Ecossistema | Entrega | Operação | Time | Segurança | **Score** |
|---|---|---|---|---|---|---|---|---|
| **Go** | 9 | 9 | 8 | 8 | 9 | 7 | 8 | **8,6** |
| Rust (axum) | 10 | 10 | 7 | 5 | 7 | 5 | 10 | 8,0 |
| Node/TS (Fastify) | 7 | 8 | 9 | 9 | 8 | 9 | 6 | 8,0 |
| Elixir/Phoenix | 9 | 9 | 6 | 6 | 8 | 4 | 8 | 7,4 |
| Java/Kotlin | 8 | 8 | 9 | 6 | 8 | 6 | 8 | 7,6 |
| Bun | 7 | 7 | 8 | 9 | 6 | 8 | 5 | 7,2 |

**Decisão:** Go. Ele entrega 90% do ganho de concorrência de Rust com muito menos custo de desenvolvimento e um ecossistema maduro para HTTP/SSE, Postgres (`pgx`), Redis, Qdrant, NATS e OTel. O fallback documentado é Node/TS (Fastify + `undici` + workers) caso o time seja 100% TypeScript — mesmo score, com mais memória e GC.

**Alternativas descartadas para o P0:** Rust (custo de entrega), Elixir (fit de time/ecossistema), serverless (cold start e limites de SSE), Bun (maturidade de streaming/proxy).

## 4. Cache exato

| Opção | Prós | Contras | Veredito |
|---|---|---|---|
| **Redis 8/Valkey 8** | TTL, Lua, cluster, persistência, ecossistema | Memória cara; operação de cluster | **Escolhido** |
| Dragonfly | Throughput alto, compatível com Redis, eficiente em memória | Maturidade menor | Alternativa P2 |
| Memcached | Simples e rápido | Sem Lua/persistência/estruturas | Não |
| In-process (L1) | Latência ~µs | Não compartilhado entre réplicas; invalidação difícil | Complemento opcional com pub/sub |

**Desenho:** L1 opcional por réplica (TTL 5–15 s) + **Redis como L2**; invalidação por pub/sub. Chave: `sha256(tenant|scope|model|normalized_messages|params|prompt_version)`.

## 5. Cache semântico

| Opção | Prós | Contras | Veredito |
|---|---|---|---|
| **Qdrant** | HNSW, filtros por payload, snapshots, cliente Go | Mais um componente para operar | **Escolhido** |
| pgvector | Menos um store; SQL | ANN/latência pior em 2M+; tuning trabalhoso | Fallback P1 |
| Redis Vector | Menos componentes | Memória alta; ANN menos maduro | Alternativa |
| Weaviate/Milvus | Recursos avançados | Operação pesada | Não no P0 |

**Embeddings:** modelo local ONNX (ex.: BGE-small, 384 dims) para custo zero e privacidade; fallback para `/v1/embeddings` de um provedor se a qualidade exigir. **Threshold inicial 0,95**, TTL 24 h, somente respostas determinísticas (temperatura ≤ 0,2) e sem tools no P0.

## 6. Failover, rate limit e coordenação

| Necessidade | Opção escolhida | Alternativas | Motivo |
|---|---|---|---|
| Rate limit distribuído | Redis + Lua (token bucket/sliding window) | Envoy RLS, serviço dedicado | Precisão e baixa latência |
| Circuit breaker | Estado em Redis + cache local | Só local (não compartilha entre réplicas) | Consistência entre réplicas |
| Cooldown/lockout | Redis com TTL | Postgres | TTL nativo e leitura O(1) |
| Idempotência | Redis + Postgres | Só Redis (perde durabilidade) | Replay seguro |
| Filas | **NATS JetStream** | Kafka, Redis Streams, RabbitMQ | Leve, Go-native, at-least-once |

## 7. Persistência e analytics

| Necessidade | Escolha | Alternativas | Motivo |
|---|---|---|---|
| System of record | **PostgreSQL 17** | MySQL, CockroachDB | JSONB, RLS, partições, extensões |
| Analytics detalhado | **ClickHouse (P2)** | Postgres particionado | 130 GB/dia não cabe bem em OLTP |
| Cache de configuração | Redis | Postgres | Baixa latência |
| Busca de auditoria | Postgres + partições | Elasticsearch | Escala inicial suficiente |

**Limiar para ClickHouse:** > 50 milhões de linhas/mês em `requests` ou > 5 TB de detalhe. Antes disso, agregados em `usage_daily` + amostragem de 10% no Postgres.

## 8. Observabilidade (sem stack própria)

> **Restrição do operador:** não usar Prometheus, Grafana, Loki, Tempo ou stack OSS auto-operada. A instrumentação continua **OpenTelemetry**; o backend é gerenciado.

| Opção | Prós | Contras | Veredito |
|---|---|---|---|
| **OTel + Datadog** | APM/logs/métricas/RUM em um só lugar, alertas e SLOs, correlação nativa | Custo por host/ingest; egress para SaaS; não atende air-gapped | **Escolhido (padrão corporativo)** |
| OTel + Azure Monitor/Application Insights | Integração Microsoft, bom para ambientes Azure, sem operar backend | Menos unificado fora do Azure | Alternativa se o corporativo for Microsoft |
| OTel + New Relic | APM forte, bom para SRE | Custo e lock-in | Alternativa |
| Só logs | Simples | Sem SLO/tracing real | Não |

**Sinais mínimos:** latência (p50/p95/p99) por provider/modelo, RPS, erros por código, saturação (réplicas, Redis, pools), cache hit/miss, failover, saúde de proxy, decisões DLP.
**Controles:** amostragem de 10% (100% em erro), cardinalidade limitada (sem `tenant_id` como label), redaction de PII e retenção configurável. Em ambiente air-gapped, a decisão D11 define o backend substituto.

## 9. Proxy Webshare

| Opção | Prós | Contras | Veredito |
|---|---|---|---|
| **Webshare** | Custo acessível, API, rotating/dedicated | Banda e reputação variam | **Escolhido (requisito)** |
| Bright Data | Qualidade e escala | Custo alto | Alternativa P2 |
| Oxylabs | Enterprise | Custo/contrato | Alternativa P2 |
| Self-hosted | Controle | Impossível cobrir diversidade de IPs | Não |

**Desenho:** sync a cada 15 min, health check a cada 30 s, score = `0,5×sucesso + 0,3×(1/latência) + 0,2×(1−ban_rate)`, sticky por conta (5–10 min), limite de concorrência por proxy, quarentena e fallback direto. Credenciais cifradas; nunca logar `user:password`.

## 9.1 Lazy loading e cache em camadas

### Frontend (console)

| Técnica | Aplicação | Meta |
|---|---|---|
| Route-level code splitting | Cada tela em chunk próprio (`loading.tsx`/Suspense) | Bundle inicial ≤ 250 KB gzip |
| Dynamic import | Charts, Monaco, logs, tabelas grandes, drawers | Nenhum no primeiro paint |
| Virtualização | Tabelas/listas > 50 linhas | Memória estável |
| Imagens/fontes | `next/image`, `font-display: swap`, preload só do crítico | LCP ≤ 2,5 s |
| Dados | SWR/React Query com `stale-while-revalidate` | Navegação percebida < 100 ms |
| Prefetch | Chunks prováveis no idle | TTI menor sem competir com LCP |

### Backend (gateway)

- **Eager (caminho quente):** config, auth, rate limit, provider padrão, schema do banco e Redis.
- **Lazy (primeiro uso):** adaptadores de providers não padrão, catálogo completo de modelos, modelo de embeddings, sync de proxies Webshare, NER/Presidio e exporters opcionais.
- **Regras:** init lazy com timeout, falha aberta e telemetria; nunca bloquear o primeiro token; cachear o resultado do init; testar cold start.

### Cache em camadas

| Camada | Store | TTL | Hit esperado | Papel |
|---|---|---|---|---|
| L1 | memória do processo | 5–15 s | 5–10% | absorber rajadas e reduzir ops no Redis |
| L2 exato | Redis | 1 h | ≥ 25% | cache entre réplicas |
| Semântico | Qdrant | ≤ 24 h | +15% | similaridade ≥ 0,95 |
| HTTP | CDN/Traefik | imutável/1 ano | — | assets estáticos do console |
| Negativo | Redis | 30–60 s | — | evitar repetir erro/404 de provider |

**Invalidação:** pub/sub no Redis para L1; versão de prompt/modelo na chave; TTL curto no semântico; nunca cruzar tenants.

## 10. Deploy e multi-tenancy

| Decisão | Escolha | Alternativas | Motivo |
|---|---|---|---|
| Orquestração | **Docker Swarm** (autoscaling por conexões/RPS via controlador externo) | Bare metal, Nomad, serverless | Escala e rolling update com menor complexidade que Kubernetes |
| Isolamento de tenant | **Shared schema + `tenant_id` + RLS** | Schema/DB por tenant | Menor custo operacional; RLS no P1/P2 |
| Cache por tenant | Prefixo + payload filter | Coleção por tenant | Isolamento lógico no P0; físico no P2 |
| Segredos | AES-256-GCM + KMS/Vault | Somente env | Rotação e auditoria |
| Ingress | Traefik/NGINX com HTTP/2 e SSE | API Gateway de nuvem | Controle de buffering/timeout |

### 10.1 Operação em Docker Swarm

- **Topologia:** 3 managers (quorum Raft) + N workers; labels `tier=edge|app|data` para placement.
- **Stack file:** `stack.yml` versionado; `docker stack deploy -c stack.yml dontstop`.
- **Serviços:** `gateway`, `admin`, `traefik`, `pgbouncer`, `redis`, `qdrant`, `nats`, `otel-collector`. **Sem Prometheus/Grafana/Loki/Tempo**; o OTel Collector exporta OTLP para o backend gerenciado (Datadog/Azure Monitor).
- **Redes:** overlay `edge` (pública), `app` (gateway/admin) e `data` (bancos) com `--opt encrypted`.
- **Segredos:** Docker secrets para senhas, chaves de API e TLS; nunca em `environment` ou imagem.
- **Rolling update:** `--update-order start-first --update-parallelism 2 --update-delay 30s --update-failure-action rollback`; `--stop-grace-period 90s` para drenar SSE.
- **Healthchecks:** `/healthz` e `/readyz` por serviço; `--health-start-period` compatível com o boot.
- **Autoscaling:** o Swarm **não tem HPA nativo**. Usar `docker service scale gateway=N` via runbook (P0) ou um controlador externo em Go que lê as métricas do backend gerenciado e chama a Docker API (P1). O alvo é 70% de utilização de conexões/CPU.
- **Persistência:** Postgres/Redis/Qdrant/NATS em nós `tier=data` com volumes locais e backup; em nuvem, preferir serviços gerenciados e manter apenas o gateway no Swarm.
- **Multi-região (P2):** clusters Swarm independentes atrás de um balanceador global; replicação de Postgres e cache por região.
- **Limitações:** sem operator/CRD, sem service mesh nativa, autoscaling e gestão de estado mais manuais; exigem runbooks fortes e observabilidade madura.

## 11. Recomendação final

**Adotar Go no data plane, TypeScript/Next.js no control plane, PostgreSQL + Redis + Qdrant + NATS, OpenTelemetry + backend gerenciado (Datadog/Azure Monitor), Docker Swarm e Webshare.**

Justificativa em uma frase: é a combinação que atende 20 mil streams com memória previsível, mantém o custo de cache e DLP controlado, permite failover em menos de 2 s com estado compartilhado e preserva um caminho claro de multi-tenancy via `tenant_id` + RLS, sem exigir a reescrita que os gateways auditados sofreram ao manter dois stacks `[MAP-ALLCODE]`.

## 12. Riscos da stack e mitigação

| Risco | Prob. | Impacto | Mitigação |
|---|---|---|---|
| Time sem experiência em Go | Média | Alto | ADR-001 com fallback TS; pair programming; começar pelo caminho crítico |
| Operação de Qdrant/Redis cluster | Média | Médio | Serviços gerenciados ou runbooks; snapshots; alertas |
| Cache semântico errar resposta | Média | Alto | Threshold alto, TTL curto, escopo por tenant, feature flag |
| Banda Webshare estourar | Alta | Alto | Rotear por proxy só quando necessário; medir; plano alternativo |
| DLP falso positivo | Média | Médio | Modo audit antes de block; allowlist; métricas de FP |
| Postgres virar gargalo de escrita | Média | Alto | Batch/NATS, PgBouncer, partições, ClickHouse no P2 |
| RLS degradar performance | Baixa | Médio | Índices com tenant; teste de carga; schema-per-tenant no P2 se necessário |
| Swarm sem autoscaling nativo | Alta | Médio | Controlador externo (métricas do backend + Docker API) no P1; scale manual com alertas no P0; capacidade reservada |
| Lock-in/custo de observabilidade SaaS | Média | Médio | OTel como contrato; orçamento de cardinalidade; decisão D11 para air-gapped |
| Lazy loading degradar o primeiro uso | Média | Médio | Eager só do crítico; timeout/fallback; teste de cold start |
| Estado em volumes locais | Média | Alto | Nós `tier=data` dedicados, backups testados e serviços gerenciados quando possível |
| Ausência de CRDs/service mesh nativos | Média | Baixo | Runbooks, observabilidade e evitar dependência de recursos exclusivos de Kubernetes |

## 13. ADRs relacionados

ADR-001 (Go), ADR-002 (Postgres), ADR-003 (Redis), ADR-004 (Qdrant), ADR-005 (NATS), ADR-006 (OTel + backend gerenciado), ADR-007 (multi-tenancy), ADR-008 (Webshare), ADR-009 (DLP), ADR-010 (Docker Swarm), ADR-011 (segredos), ADR-012 (compatibilidade OpenAI), ADR-013 (lazy loading/cache em camadas), ADR-014 (identidade corporativa), ADR-015 (endpoint registry/API key scopes). Detalhes em [05-ADRS.md](05-ADRS.md).

## 14. Fontes consultadas

- `[MAP-9R]` — fallback, cache/RTK, SQLite, mutex global e segredos em texto plano.
- `[MAP-OMNI]` — 380 provedores, circuit breaker/cooldown, Redis opcional, 222 migrações, policies e dívidas de escala.
- `[MAP-ALLCODE]` — catálogo/plugins, protocolos, adaptadores, rotação sem feedback e instalação em runtime.
- `[REPO]` — confirmação de greenfield; nenhum benchmark local foi executado.
