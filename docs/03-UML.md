# dontstop — UML e diagramas de arquitetura

> Fontes: `[REPO]`, `[MAP-9R]`, `[MAP-OMNI]`, `[MAP-ALLCODE]` (ver [README](README.md)).
> Diagramas em Mermaid; são propostas de arquitetura, não código existente.

## 1. Diagrama de componentes

```mermaid
flowchart LR
    subgraph Clients[Clientes]
        SDK[SDKs OpenAI / CLIs]
        WEB[Console Admin]
    end

    subgraph Edge[Edge]
        ING[Ingress / TLS / WAF]
        LB[Load Balancer]
    end

    subgraph DataPlane[Data Plane — Go]
        API[API Gateway<br/>/v1/*]
        AUTH[Auth + Tenant Context]
        RL[Rate Limiter<br/>Redis Lua]
        L1[L1 Cache<br/>in-process]
        CACHEX[Exact Cache<br/>Redis L2]
        CACHES[Semantic Cache<br/>Qdrant]
        DLP[DLP / Policy Engine]
        ROUTER[Router / Combos]
        POOL[Account Pool + Health]
        CB[Circuit Breaker]
        ADP[Provider Adapters<br/>OpenAI/Anthropic/Gemini/Bedrock/Azure]
        PROXY[Proxy Manager<br/>Webshare]
        USAGE[Usage / Cost Meter]
    end

    subgraph ControlPlane[Control Plane — TypeScript/Next.js]
        ADMIN[Admin API + Console]
        CATALOG[Catálogo de Providers/Modelos]
        POLICY[Políticas e DLP Rules]
        ENDPOINTS[Endpoint Registry + API Manager]
    end

    subgraph Stores[Stores]
        PG[(PostgreSQL 17<br/>+ PgBouncer)]
        REDIS[(Redis 8)]
        QDRANT[(Qdrant)]
        NATS[(NATS JetStream)]
    end

    subgraph Obs[Observabilidade]
        OTEL[OTel Collector]
        DD[Backend gerenciado<br/>Datadog / Azure Monitor]
    end

    subgraph Upstreams[Upstreams]
        P1[OpenAI]
        P2[Anthropic]
        P3[Gemini]
        P4[Bedrock]
        P5[Compatíveis]
        WS[Webshare Proxies]
    end

    SDK --> ING --> LB --> API
    WEB --> ING
    API --> AUTH --> RL --> L1
    L1 -->|miss| CACHEX
    CACHEX -->|miss| CACHES
    CACHES -->|miss| DLP --> ROUTER --> POOL
    POOL --> CB --> ADP
    ADP --> PROXY --> WS
    PROXY --> P1 & P2 & P3 & P4 & P5
    API --> USAGE --> NATS
    ADMIN --> PG
    CATALOG --> PG
    POLICY --> PG
    ENDPOINTS --> PG
    API --> ENDPOINTS
    ADMIN --> ENDPOINTS
    API --> PG
    CACHEX --> REDIS
    RL --> REDIS
    CACHES --> QDRANT
    USAGE --> PG
    API --> OTEL --> DD
    ADMIN --> OTEL
```

## 2. Sequência — requisição com cache exato

```mermaid
sequenceDiagram
    autonumber
    participant C as Cliente
    participant G as Gateway
    participant A as Auth/Tenant
    participant R as RateLimit
    participant X as ExactCache
    participant D as DLP
    participant RT as Router
    participant P as Provider
    participant U as Usage

    C->>G: POST /v1/chat/completions
    G->>A: valida API key + tenant
    A-->>G: contexto (tenant, key, scopes)
    G->>R: token bucket (tenant/key)
    R-->>G: allow + remaining
    G->>X: GET t:{tenant}:exact:{hash}
    alt cache hit
        X-->>G: resposta + metadados
        G-->>C: 200 (SSE ou JSON)
        G->>U: registra cache_hit
    else cache miss
        G->>D: inspeciona prompt
        D-->>G: allow/redact/block
        G->>RT: resolve rota/modelo
        RT-->>G: alvo + conta
        G->>P: chamada upstream (stream)
        P-->>G: chunks
        G->>D: inspeciona resposta
        G-->>C: chunks (SSE)
        G->>X: SET resposta (TTL)
        G->>U: registra tokens/custo
    end
```

> **P0-α:** o primeiro slice executa Auth → RateLimit → ExactCache → Router/Failover → Provider → Usage/Outbox. DLP e cache semântico entram no P0-β/P1 sem alterar o contrato público. A sequência de entrega está em [ADD §6.2](architecture/ADD.md).

## 3. Sequência — cache semântico

```mermaid
sequenceDiagram
    autonumber
    participant G as Gateway
    participant E as Embedder
    participant Q as Qdrant
    participant PG as Postgres
    participant P as Provider

    G->>E: embedding(prompt normalizado)
    E-->>G: vetor (384/768 dims)
    G->>Q: search(tenant, model, top=3)
    Q-->>G: candidatos + score
    alt score >= threshold (ex.: 0,95)
        G->>PG: valida metadados/TTL/versão
        PG-->>G: resposta elegível
        G-->>G: serve do cache (custo zero)
    else sem hit elegível
        G->>P: chamada upstream
        P-->>G: resposta
        G->>Q: upsert vetor + payload
        G->>PG: registra semantic_cache_entry
    end
```

## 4. Sequência — failover de conta/modelo/provider

```mermaid
sequenceDiagram
    autonumber
    participant G as Gateway
    participant RT as Router
    participant CB as CircuitBreaker
    participant A1 as Conta A
    participant A2 as Conta B
    participant P2 as Provider B

    G->>RT: alvo primário
    RT->>CB: canExecute(provider A)
    CB-->>RT: closed
    RT->>A1: request
    A1-->>RT: 429 / 5xx
    RT->>CB: recordFailure(A)
    RT->>A1: cooldown (exponencial)
    alt há conta B no mesmo provider
        RT->>A2: request
        A2-->>RT: 200 (stream)
        RT-->>G: resposta
    else esgotou contas
        RT->>CB: open(A) + half-open timer
        RT->>P2: próximo provider/combo
        P2-->>RT: 200 (stream)
        RT-->>G: resposta
    end
    RT->>RT: registra failover_event
```

## 5. Sequência — seleção de proxy Webshare (isolamento por provedor)

```mermaid
sequenceDiagram
    autonumber
    participant G as Gateway
    participant PM as ProxyManager
    participant DB as PostgreSQL
    participant R as Redis
    participant W as Webshare API
    participant X as Proxy
    participant U as Upstream

    G->>PM: resolve(conta, modo efetivo)
    PM->>DB: assignment atual + health + poison(provedor, endpoint)
    alt modo auto e assignment ausente
        PM->>DB: candidatos webshare, sem outra conta do MESMO provedor
        DB-->>PM: IP livre (maior score)
        PM->>DB: assign(conta, endpoint)
    end
    PM-->>G: proxy selecionado (IP fixo)
    G->>X: CONNECT (HTTP/SOCKS5)
    X->>U: requisição
    alt falha classificada (unreachable/407/ban)
        U-->>G: erro
        G->>PM: rotate(conta, endpoint)
        PM->>DB: poison(provedor, endpoint, TTL por classe)
        PM->>R: debounce t:{tenant}:proxy:refresh
        PM->>DB: próximo candidato do mesmo provedor
        PM-->>G: novo IP (retry na mesma request)
    else sucesso
        U-->>X: resposta
        X-->>G: stream
        G->>PM: report(sucesso, latência)
        PM->>DB: score/health
    end
    Note over PM,W: assíncrono: re-sync importa o substituto do IP morto
```

> Regra de alocação (ADR-024): 1 conta por IP **por provedor**; reuso entre provedores permitido.

## 6. Sequência — DLP (redact/block)

```mermaid
sequenceDiagram
    autonumber
    participant G as Gateway
    participant PE as PolicyEngine
    participant D as DLP Detectors
    participant V as Token Vault
    participant A as Audit
    participant U as Upstream

    G->>PE: evaluate(tenant, rota, prompt)
    PE->>D: scan(prompt)
    D-->>PE: findings (tipo, severidade, span)
    alt ação = block
        PE-->>G: deny (400/403)
        G->>A: audit(decision=block)
        G-->>G: resposta de erro sanitizada
    else ação = redact
        PE->>V: tokeniza PII (opcional reversível)
        V-->>PE: prompt redigido
        G->>U: prompt redigido
        U-->>G: resposta
        G->>D: scan(resposta)
        G->>A: audit(decision=redact, hashes)
        G-->>G: resposta (rehidratada se autorizado)
    else ação = allow/audit
        G->>U: prompt original
        G->>A: audit(decision=allow/audit)
    end
```

## 6.1 Sequência — Lazy loading do console

```mermaid
sequenceDiagram
    autonumber
    participant U as Usuário
    participant R as Rota Next.js
    participant S as Skeleton
    participant C as Chunk sob demanda
    participant A as Admin API
    participant B as Backend (Go)

    U->>R: abre /dashboard
    R-->>U: shell + skeleton (SSR)
    R->>C: dynamic import (gráficos/tabela)
    C-->>U: componente montado
    R->>A: fetch dados (SWR/React Query)
    A->>B: consulta agregada (cache)
    B-->>A: dados (stale-while-revalidate)
    A-->>R: resposta cacheada
    R-->>U: render final
    Note over R,B: Monaco, embeddings, catálogo completo e NER só carregam ao abrir a feature
```

> **Backend lazy:** adaptadores de provider, catálogo de modelos, modelo de embeddings e pools de proxy inicializam no primeiro uso; apenas auth, rate limit e o provider padrão ficam no caminho quente. Init lazy deve falhar aberto com timeout e telemetria.

## 6.2 Sequência — endpoint + API key

```mermaid
sequenceDiagram
    autonumber
    participant A as Admin
    participant M as API Manager
    participant DB as Postgres
    participant R as Redis
    participant C as Cliente
    participant G as Gateway

    A->>M: cria endpoint/política e escopo de chave
    M->>DB: persiste endpoint + policies + scopes
    M->>R: publica invalidação (pub/sub)
    C->>G: POST /v1/chat/completions (API key)
    G->>R: lê endpoint + política (cache)
    alt cache miss
        G->>DB: carrega endpoint + policies
        G->>R: popula cache com TTL curto
    end
    G->>DB: valida hash da API key + escopos
    G->>G: aplica rate limit/modelos/cache/DLP/proxy
    G-->>C: 200 (SSE) ou erro 401/403/429
    G->>DB: registra uso por endpoint e chave
```

> **Regra:** endpoint desligado, versão deprecada ou escopo ausente falham antes de qualquer chamada upstream; a decisão é auditada. O API Manager nunca retorna o segredo completo após a criação.

## 7. Diagrama de classes (domínio)

```mermaid
classDiagram
    class TenantContext {
      +UUID tenantID
      +UUID apiKeyID
      +Set~Scope~ scopes
      +Budget budget
    }
    class GatewayRequest {
      +String requestID
      +String model
      +Message[] messages
      +Bool stream
      +Map params
      +TenantContext tenant
    }
    class Provider {
      +UUID id
      +String slug
      +Kind kind
      +String baseURL
      +Protocol protocol
      +Bool enabled
    }
    class ProviderAccount {
      +UUID id
      +String name
      +AuthType authType
      +Int priority
      +Int weight
      +AccountStatus status
      +Float healthScore
      +Bool isAvailable(now)
    }
    class Credential {
      +Bytes encrypted
      +String keyVersion
      +decrypt() Secret
    }
    class Route {
      +UUID id
      +String name
      +Strategy strategy
      +Target[] targets
      +resolve(ctx) Target
    }
    class Target {
      +UUID providerID
      +String model
      +Int priority
      +Int weight
    }
    class CircuitBreaker {
      +State state
      +canExecute() Bool
      +recordSuccess()
      +recordFailure()
    }
    class ExactCache {
      +get(key) CachedResponse
      +set(key, resp)
      +invalidate(scope)
    }
    class SemanticCache {
      +find(vector, tenant, model) Match
      +store(vector, resp)
    }
    class DlpEngine {
      +scan(text) Finding[]
      +decide(findings, policy) Decision
    }
    class ProxyManager {
      +resolve(accountID, mode) Lease
      +rotate(accountID, failedEndpointID) Lease
      +report(proxy, outcome)
      +poison(providerID, endpointID, reason, ttl)
      +refreshPool(tenantID)
    }
    class UsageMeter {
      +record(request, usage)
      +cost(tokens, model) Money
    }
    class PolicyEngine {
      +evaluate(action, resource, ctx) Effect
    }

    GatewayRequest --> TenantContext
    GatewayRequest --> Route
    Route --> Target
    Target --> Provider
    Provider --> ProviderAccount
    ProviderAccount --> Credential
    ProviderAccount --> CircuitBreaker
    ProviderAccount --> ProxyManager
    GatewayRequest --> ExactCache
    GatewayRequest --> SemanticCache
    GatewayRequest --> DlpEngine
    GatewayRequest --> UsageMeter
    DlpEngine --> PolicyEngine
```

## 8. Diagrama de implantação

```mermaid
flowchart TB
    subgraph Internet
        C[Clientes]
    end

    subgraph Swarm[Docker Swarm Cluster]
        MGR[Swarm Managers x3<br/>Raft quorum]
        subgraph EdgeNS[edge]
            TRAEFIK[Traefik<br/>Ingress + TLS]
            WAF[WAF / Rate Limit L7]
        end
        subgraph AppNS[app]
            GW1[Gateway task 1<br/>Go]
            GW2[Gateway task 2<br/>Go]
            GW3[Gateway task N<br/>Go]
            ADM[Admin task<br/>Next.js]
            SCALER[Autoscaler controller<br/>external metrics]
        end
        subgraph DataNS[data]
            PGP[(Postgres Primary)]
            PGR[(Postgres Replica)]
            PGB[PgBouncer]
            RDS[(Redis Cluster)]
            QD[(Qdrant Cluster)]
            NATS[(NATS JetStream)]
        end
        subgraph ObsNS[observability]
            OTEL[OTel Collector]
            DD[Backend gerenciado<br/>Datadog / Azure Monitor]
        end
    end

    subgraph Egress[Saída]
        WS[Webshare Pool]
        UP[Provedores LLM]
    end

    C --> TRAEFIK --> WAF --> GW1 & GW2 & GW3
    C --> TRAEFIK --> ADM
    GW1 & GW2 & GW3 --> PGB --> PGP
    PGP --> PGR
    GW1 & GW2 & GW3 --> RDS
    GW1 & GW2 & GW3 --> QD
    GW1 & GW2 & GW3 --> NATS
    GW1 & GW2 & GW3 --> WS --> UP
    GW1 & GW2 & GW3 --> UP
    GW1 & GW2 & GW3 --> OTEL --> DD
    ADM --> PGB
    SCALER -.-> GW1
    SCALER -.-> GW3
    MGR -.-> GW1
```

> **Notas de implantação (Docker Swarm):** 3 managers para quorum Raft e N workers; serviços `gateway`, `admin`, `traefik`, `pgbouncer`, `redis`, `qdrant`, `nats` e `otel-collector`; **sem Prometheus/Grafana/Loki/Tempo** — o `DD` é um backend gerenciado externo (Datadog ou Azure Monitor) que recebe OTLP. Redes overlay `edge`, `app` e `data` (esta com `--opt encrypted`); segredos via Docker secrets; dados persistidos em volumes locais com placement em nós rotulados (`node.labels.tier==data`) ou em serviços gerenciados. O Swarm **não possui HPA nativo**: o `SCALER` lê métricas do backend gerenciado e chama a Docker API; no P0 o scaling pode ser manual via runbook.

## 9. Máquinas de estado

### 9.1 Circuit breaker (provider)

```mermaid
stateDiagram-v2
    [*] --> CLOSED
    CLOSED --> OPEN: 5 falhas ou 50% de erro em 30 s
    OPEN --> HALF_OPEN: após 15 s
    HALF_OPEN --> CLOSED: sucesso na sondagem
    HALF_OPEN --> OPEN: falha na sondagem
    OPEN --> OPEN: novas requisições falham rápido
```

### 9.2 Conta/credencial

```mermaid
stateDiagram-v2
    [*] --> PENDING
    PENDING --> ACTIVE: validação OK
    PENDING --> DISABLED: validação falhou
    ACTIVE --> COOLDOWN: 429/5xx/timeout
    COOLDOWN --> ACTIVE: cooldown expirado + health OK
    ACTIVE --> BANNED: 401/403 repetido ou bloqueio do provider
    BANNED --> ACTIVE: reautenticação manual
    ACTIVE --> DISABLED: admin desativou
    DISABLED --> ACTIVE: admin reativou
```

## 10. Observações de modelagem

- O diagrama de componentes separa **data plane** (Go, caminho quente) de **control plane** (TypeScript/Next.js, CRUD e console).
- O cache exato é consultado **antes** do DLP para evitar custo de inspeção em respostas já aprovadas; a política define se um hit pode ser servido sem reavaliação (por padrão, o cache é escopado por tenant e por versão de política).
- O failover registra evento antes de responder; a resposta ao cliente só é enviada quando um alvo entrega o primeiro chunk (ou corpo completo no não-streaming).
- O proxy é adquirido por tentativa de conta, não por requisição inteira, para permitir rotação sem recriar o contexto.
- O DLP em streaming opera em janelas com sobreposição; decisões `block` só podem ser tomadas antes do primeiro chunk enviado ao cliente (ou via abort explícito).

## 11. Premissas e lacunas

- **Premissa:** o ingress suporta HTTP/2 e SSE sem buffering; validar com o provedor de nuvem.
- **Premissa:** Qdrant e Redis aceitam cluster mode no ambiente alvo.
- **Lacuna:** se o data plane será Go puro ou se o time exigirá TypeScript (ADR-001).
- **Lacuna:** se haverá service mesh (Istio/Linkerd); não foi assumido no P0.
- **Pendente:** WebSocket para Responses API (P2) e backpressure específico.

## 12. Fontes consultadas

- `[MAP-9R]` — pipeline de requisição, combos, fallback de conta, cache/RTK e o risco de mutex global.
- `[MAP-OMNI]` — circuit breaker/cooldown/model lockout, route guard, policy engine e Redis opcional.
- `[MAP-ALLCODE]` — separação catálogo/execução, adaptadores de protocolo, OAuth e lacuna de feedback de rotação.
- `[REPO]` — confirmação de que os diagramas são propostas greenfield.
