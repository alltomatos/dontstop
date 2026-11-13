# Proxy Webshare por conta de provedor — estudo, comparativo e plano

> **Status:** proposta. Nada implementado. Aguarda aprovação do plano e do ADR-021.
> 9Router e OmniRoute aparecem aqui apenas como **referência conceitual de estudo**; nenhum
> adapter, SDK, relay, upstream ou `base_url` do dontstop aponta para eles (AGENTS.md §1.1 e §14).

## 1. API do Webshare — resumo acionável

Fonte: <https://apidocs.webshare.io/> (consulta em 2026-11). Resumo completo também gravado na
memória do projeto.

### 1.1 Autenticação e base

- Header em todas as chamadas: `Authorization: Token <API_KEY>` (chave criada em Dashboard → API Keys).
- v2: `https://proxy.webshare.io/api/v2` (listas, refresh, proxy replacement).
- v3: `https://proxy.webshare.io/api/v3` (config, status, stats do plano).

### 1.2 Listagem de proxies

```
GET /api/v2/proxy/list/?mode=direct|backbone&page=1&page_size=100
    &country_code__in=BR,US&valid=true&plan_id=<uuid>
```

- `mode` é **obrigatório**; `backbone` para planos residenciais (`plan.pool_filter=residential`).
- Em `backbone`, `search`, `ordering`, `valid`, `created_at`, `proxy_address*`, `asn_*` **não funcionam**.
- Resposta paginada: `{count, next, previous, results[]}`; cada item:
  `id, username, password, proxy_address, port, valid, last_verification, country_code, city_name, created_at`.
- Filtros/ordenação: exato, `__in`, `__gt`, `__lt`, `__contains`, `search`, `ordering` (com `-` para DESC).

### 1.3 Download em massa

```
GET /api/v2/proxy/list/download/{token}/{CCs|'-'}/any/{username|sourceip}/{direct|backbone}/{search|'-'}/?plan_id=<uuid>
```

- Sem autenticação; `token` vem de `proxy_list_download_token` (`GET /api/v3/proxy/config?plan_id=...`)
  ou de `/downloads/get_download_token`; rotação por `/downloads/reset_download_token`.
- Corpo em texto: `ip:port:username:password` (uma linha por proxy).

### 1.4 Configuração, status e estatísticas (v3)

| Endpoint | Uso |
|---|---|
| `GET /api/v3/proxy/config?plan_id=` | timeouts, `ip_authorization_*`, `auto_replace_*`, `proxy_list_download_token` |
| `GET /api/v3/proxy/list/status?plan_id=` | `state`, `countries`, `unallocated_countries`, `username`, `password`, `is_proxy_used` |
| `GET /api/v3/proxy/list/stats?plan_id=` | alocados e `available_*` por país, /24, /16, /8 e ASN |
| `POST /api/v2/proxy/list/refresh/?plan_id=` | `204`; exige `on_demand_refreshes_available` |

### 1.5 Conexão e sticky

- **Backbone:** `p.webshare.io` nas portas `80`, `1080`, `3128`, `9999–19999`.
  Username `{user}-{cc}-{city}-{session}`:
  - `session` numérica = **sticky** (mesmo IP de saída durante a sessão);
  - `rotate` = IP novo por request (não combina com `session`);
  - `city` só em planos residenciais.
- **Direct:** `proxy_address:port` + `username`/`password`, ou **IP Authorization** (sem credenciais,
  origem liberada via `/ipauthorization`).

### 1.6 Limites e erros

- Rate limits: geral **240/min**; proxy list **60/min**; links de download **30/min**. Em `429`,
  aguardar **60 s**.
- Erros: `400` (mapa campo → `{message, code}`), `401` token inválido, `403` plano, `404`, `405`,
  `406`, `429`, `5xx`.
- Paginação: `page`/`page_size` (default 25) ou `starting_after` em endpoints de streaming.

## 2. Comparativo 9Router × OmniRoute (proxy por conta)

| Dimensão | 9Router | OmniRoute |
|---|---|---|
| Modelo | `proxy_pools` com `name`, `proxyUrl`, `noProxy`, `type` (`http`, `vercel`, `cloudflare`, `deno`, `netlify`), `isActive`, `strictProxy` | Registro de proxies no DB, com escopos (global/por escopo), estratégias de rotação e atribuição por conexão |
| Vínculo com a conta | `providerSpecificData.proxyPoolId` (+ legado `connectionProxyEnabled/Url/noProxy`) em cada conexão | `resolveProxyForConnection(connectionId)` e `resolveProxyForScopeFromRegistry(scope, scopeId)` |
| Precedência | pool da conta → pool default → proxy legado → direto | conexão → scope pool → cascata/fallback → direto |
| Rotação | `pickProxyPoolId` (round-robin/random) entre IDs de pool, **estado em memória** por processo | estratégias por pool (round-robin, sticky com janela, random) no registry/DB |
| Sticky | não explícita (depende do pool/fornecedor) | `sticky` com janela por escopo/connection |
| Aplicação por request | `proxyAwareFetch` com `undici.ProxyAgent`, cache de dispatcher, fallback de TLS, bypass de hosts MITM | `proxyDispatcher` (undici Agent/ProxyAgent, SOCKS, Happy Eyeballs, timeouts), tipos relay, cache de dispatchers |
| Falha do proxy | `strictProxy=true` falha; padrão cai para **direto** com warning | `proxyFetch` com `strict`, no-proxy, **redaction**, health check (`proxyHealth`) e fallback |
| Saúde/score | **não há** score, health check ou quarentena por proxy | health check e seleção por candidatos; `proxyFallback` testa em paralelo e cacheia 5 min por URL |
| Descoberta/sync | manual (CRUD de pools) | registry + sync de provedor de proxies (`oneproxy`) + candidatos de env/registry |
| Observabilidade | logs de warning; sem métricas dedicadas | spans/logs com redaction, observação de IP de egress (`proxyEgress`) para detectar grupos de rotação compartilhados |
| Limitações | sem score/quarentena, rotação só em memória, granularidade de pool, credencial embutida na URL | complexidade alta, intercepta `fetch` global, caches por processo, relay types não padronizados |

### 2.1 O que vale reaproveitar como padrão

1. **Vínculo explícito por conta/conexão**, com precedência determinística e modo `strict`.
2. **Fallback direto como padrão**, com opção de falhar fechado (`required`) por conta/tenant.
3. **`no_proxy`/bypass local** e validação rígida de esquema/URL do proxy.
4. **Redaction** de credenciais e URL de proxy em logs/erros.
5. **Health check + quarentena** e seleção por score (OmniRoute), mas com estado em **Redis**, não em
   memória de processo.
6. **Observação de egress** para provar que contas diferentes não compartilham IP.

### 2.2 O que **não** copiar

- Interceptar `globalThis.fetch` (OmniRoute) — o dontstop deve injetar transporte por adapter.
- Rotação/score apenas em memória (9Router) — multi-réplica exige Redis/DB.
- Relay types que reescrevem a URL do provedor (`vercel`/`cloudflare`/`deno`/`netlify`).
- Credenciais de proxy embutidas em URL sem cifragem dedicada.

## 3. Plano de implementação (etapas)

### Fase 0 — ADR e escopo
- Aprovar ADR-021 e este plano. Definir política padrão: `proxy_mode=preferred` com fallback direto,
  sticky de 30 min, quarentena de 10 min.

### Fase 1 — Cliente Webshare (sem DB)
- Pacote `gateway/internal/proxy/webshare`: cliente com timeout, `Authorization: Token`,
  paginação, `mode=direct`, backoff em `429` (respeitar 60 s), erros sanitizados.
- Config/secret: `WEBSHARE_API_TOKEN` (Docker secret ou env), `WEBSHARE_PLAN_ID`, base URL,
  `WEBSHARE_COUNTRY_CODES` opcional. Token nunca logado.
- Testes: `httptest` para paginação, 429/5xx, payload inválido, redaction.

### Fase 2 — Schema e repositórios
- Migração `00xx_webshare_proxy.sql`:
  - `proxy_endpoints(id, tenant_id, provider_id, external_id, address, port, username_cipher,
    password_cipher, country_code, city_name, valid, score, latency_ms, consecutive_failures,
    quarantined_until, last_verified_at, created_at)`;
  - `proxy_assignments(id, tenant_id, provider_account_id UNIQUE, proxy_endpoint_id, sticky_until,
    created_at)`;
  - `provider_accounts.proxy_mode` (`off|preferred|required`) e `proxy_pin_endpoint_id` opcional;
  - índice quente `(tenant_id, provider_id, valid, quarantined_until)`.
- Credenciais cifradas com o `secret.Box` existente (AES-256-GCM).
- Testes de integração (Postgres) com isolamento por tenant.

### Fase 3 — Sync e health
- Worker (goroutine no gateway, lease/intervalo) que sincroniza a lista do Webshare, faz upsert
  idempotente e marca inválidos; execução manual via Admin API.
- Health check amostral via `https://ipv4.webshare.io/` (3 s) por proxy, atualizando
  `last_verified_at/valid`.
- Métricas OTel: `proxy_pool_size`, `proxy_sync_total`, `proxy_sync_duration_ms`.

### Fase 4 — Seleção e atribuição
- `AssignProxy(tenant, account)`: reutiliza assignment saudável e dentro do sticky; senão escolhe
  endpoint livre/atribuído com maior score, grava assignment com `sticky_until`.
- Score em Redis `t:{tenant}:proxy:ep:{id}` (sucesso, falha, p95 EWMA) e quarentena
  `t:{tenant}:proxy:q:{id}` (TTL 10 min).
- Regras: nunca compartilhar o mesmo endpoint sticky entre contas; se faltar proxy, `preferred` cai
  para direto, `required` falha com erro sanitizado.
- Testes: concorrência (duas contas não recebem o mesmo endpoint), expiração de sticky, quarentena.

### Fase 5 — Integração com adapters
- `provider.Account` ganha `ProxyURL`; `http.Client` por endpoint com `Transport.Proxy`, timeouts e
  `MaxIdleConns` limitado; cache por proxy (bounded, com GC).
- Adaptadores existentes (OpenAI/Anthropic) passam a usar o client da conta; streaming preservado.
- Teste de contrato com proxy local (`httptest` CONNECT) e fallback direto.

### Fase 6 — Feedback de roteamento
- Classificação: 407/TLS/timeout/reset = falha **do proxy**; 401/403 = credencial do provedor
  (não rotaciona proxy, exceto 407); 429/5xx = rotaciona proxy e aplica cooldown.
- No `Router`: ao falhar por proxy, libera o assignment, marca falha e tenta o próximo candidato;
  `required` encerra com erro claro.
- OTel: `proxy.attempt` (span), `proxy_success_total`, `proxy_failure_total{reason}`,
  `proxy_latency_ms`, `proxy_quarantine_active`, `proxy_egress_ip_mismatch_total`.

### Fase 7 — Admin e operação
- API: conectar/rotacionar token, listar plano/status, sincronizar agora, ver saúde e atribuições;
  PATCH de `proxy_mode` por conta.
- UI: página **Rede de proxy** centrada em IP (mock em `design-system/mockups/proxy-network.svg`, drawer em `proxy-network-drawer.svg` e modal em `proxy-webshare-config.svg`; spec em `design-system/pages/proxy-network.md`): KPIs por IP, tabela IP→contas, IPs livres, bloco de contas sem proxy, quarentena e sync. Sem expor token/credenciais.

### Fase 8 — Endurecimento
- Caos: proxy lento/morto, 407, 429 do provedor, queda total do Webshare (fallback direto),
  restart no meio do sync.
- Carga: streaming com proxy, p95 e TTFT; limites de conexão por proxy.
- Documentação final e runbook.

## 4. Estratégia de seleção (resumo)

1. Assignment existente, saudável e dentro do `sticky_until` → reutiliza (IP fixo por conta).
2. Senão, endpoint `source='webshare'` não quarentenado/envenenado, **sem outra conta do mesmo provedor** (1:1 por provedor; reuso cross-provider permitido) e maior score.
3. Grava assignment (sticky) e usa o proxy por todas as tentativas daquela conta até falha.
4. Falha classificada (`proxy_unreachable` 15 min, `proxy_auth` 1 h, `upstream_ban` 24 h): grava `proxy_poison` por `(provedor, endpoint)`, escolhe outro IP e dispara **re-sync Webshare** (debounce 30 s) para importar o substituto.
5. `manual` é fixo e fail-closed; `auto` rotaciona; `upstream_quota` (429 sem sinal de ban) não envenena.
6. Sync/health periódico alimenta `valid`, `score` e quarentena.

## 5. Riscos de segurança e privacidade

| Risco | Mitigação |
|---|---|
| Vazamento do token Webshare | Docker secret/env; nunca em log, span ou resposta; redaction testada |
| Vazamento de user/pass do proxy | Cifrados com AES-256-GCM (`secret.Box`); decifra só no momento da conexão; logs mascarados |
| SSRF / proxy malicioso | Só hosts do Webshare; esquema `http/https/socks5`; bloquear IP privado/loopback; sem URL de proxy fornecida por tenant |
| Vazamento de IP entre tenants | `tenant_id` em endpoints/assignments; nenhum IP compartilhado entre tenants; verificação de egress por conta. Dentro do mesmo tenant o compartilhamento é permitido, limitado e visível na UI |
| Isolamento por tenant | `tenant_id` em todas as tabelas; Redis `t:{tenant}:`; RLS no P1; testes com tenant cruzado |
| Abuso de rate limit | Sync com intervalo mínimo (ex.: 30–60 min), paginação, backoff 429/60 s, cache de config |
| Falha silenciosa para direto | Modo explícito (`off/preferred/required`); `required` falha fechado; métrica de fallback |
| PII em logs | Só ID/país/latência; nunca URL com credenciais nem payload |
| TOS/limites do plano | Respeitar concorrência/banda do plano; sync via API oficial; sem scraping |

## 6. Testes obrigatórios

- Unidade: cliente Webshare (paginação/erros/redaction), score/quarentena, seleção e fallback.
- Integração: migração + repositórios (tenant A/B), sync idempotente, assignment concorrente.
- Contrato: adapters OpenAI/Anthropic via proxy CONNECT local, streaming e erros.
- Caos: Webshare offline, proxy morto, 407, 429, restart no meio do run.
- Carga: streaming com proxy e limite de conexões.

## 7. Aceite

- Conta A e conta B usam IPs de saída distintos e estáveis durante o sticky.
- Falha do proxy cai para direto em `preferred` e falha fechado em `required`.
- 429/5xx/transporte rotacionam; 401/403 não rotacionam (credencial do provedor).
- Credenciais nunca aparecem em log/traço/resposta.
- `go test ./...`, `go vet`, `golangci-lint`, testes de integração e caos verdes.

## 8. Status de implementação (2026-11)

- **Implementado:** API key cifrada em `webshare_settings` (AES-256-GCM + `key_version`) com teste real antes de salvar e auditoria; cliente v2/v3 com paginação, download token, refresh e backoff de 60 s em 429; sync sob demanda e worker (`WEBSHARE_SYNC_INTERVAL`) com advisory lock e diff idempotente; `proxy_endpoints` com `external_id`, score e saúde; seleção/rotação em `internal/proxy` com quarentena Redis `t:{tenant}:proxy:*`, fallback direto em `preferred` e falha fechada em `required`; UI de configuração/status/sync em `/proxies`; testes de cliente/contrato, coordenação, integração de sync e rotação.
- **Pendente:** health check ativo contra `ipv4.webshare.io`, métricas OTel dedicadas de proxy, caos/carga de streaming com proxy, fallback de download em massa e backbone sticky.
- **Próxima onda (ADR-024):** isolamento 1 conta/IP **por provedor** (reuso cross-provider permitido), modo `auto` híbrido (`providers.proxy_auto` + `inherit|off|manual|auto`), poison persistente por `(provedor, endpoint)` com classificação de falha e re-sync Webshare na falha. Hoje o limite é de 3 contas por IP no tenant inteiro e não há poison nem `auto`.
- **Progresso E2.1 (2026-11-13):** migração `0020` aplicada no working tree — `proxy_assignments.provider_id` + `UNIQUE(tenant_id, provider_id, proxy_endpoint_id)`, dedup por provedor, `proxy_poison`, colunas de ban e `Candidates` provider-aware. Rotação por classe de falha e re-sync ainda pendentes (E2.3/E2.4).
- **E2 concluído (2026-11-13):** modo efetivo (`inherit|off|manual|auto` + `providers.proxy_auto`), alocação 1 conta/IP por provedor (reuso cross-provider), classificação de falha + poison por `(provedor, endpoint)`, re-sync Webshare com debounce, Admin API (`PATCH /providers/{id}`, `POST /proxies/webshare/refresh`), console (proxy por conta, rede de proxy com ban, re-sync), testes de segurança/concorrência e métricas `proxy.rotate`/`proxy.poison`/`proxy.candidate_miss`/`proxy.refresh`.
