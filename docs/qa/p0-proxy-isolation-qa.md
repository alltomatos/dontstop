# QA — Isolamento de proxy por provedor (Gate 1)

> Feature: ADR-024 · Issues #53–#61 · Data: 2026-11-13.
> Complementa [`proxy-isolation.md`](proxy-isolation.md) (matriz técnica) e o
> [runbook](proxy-isolation-runbook.md).

## Escopo

- Isolamento 1 conta/IP por provedor (reuso cross-provider).
- Modo efetivo `inherit|off|manual|auto` e `providers.proxy_auto`.
- Classificação de falha + poison por `(provedor, endpoint)` + expiração.
- Rotação transparente e re-sync Webshare com debounce.
- Admin API (`proxy_auto`, refresh), console (proxy por conta, `/proxies`) e guia.

**Fora de escopo:** túneis, proxies fora de Webshare/manual, health probe ativo contra
`ipv4.webshare.io`, carga/chaos de streaming com proxy, RLS (P1).

## Requisitos × evidência

| Requisito | Verificação | Resultado |
|---|---|---|
| Cada conta do provedor com IP próprio | `TestProxyAssignmentOnePerProviderWithCrossProviderReuse`, `TestConcurrentAssignSameProviderKeepsSingleAssignment` | ✅ |
| Proxy manual por conta | `TestProxyModeAndAssignmentPersistWithoutCredentials`; UI `AccountProxyControl` renderizado | ✅ |
| Reuso entre provedores diferentes | `TestRecordBanPoisonsOnlyTheProvider`, `TestProxyAssignmentOnePerProviderWithCrossProviderReuse` | ✅ |
| `inherit` → `auto`/`off` | `TestResolveEffectiveMode`, `TestPostgresCandidatesResolveInheritedProxyMode` | ✅ |
| Fail-closed em `manual`/`auto` | `TestSelectProxyManualAndAutoFailClosed`, `TestChatRequiredProxyFailsBeforeUpstream` | ✅ |
| Rotação em falha/ban; não em 401/403/429 | `TestChatRotatesProxyBeforeProviderAuth`, `TestChatRotatesOnUpstreamBan`, `TestChatDoesNotRotateRateLimit`, `TestChatDoesNotRotateProviderAuthErrors` | ✅ |
| Poison por provedor + expiração | `TestRecordBanPoisonsOnlyTheProvider` | ✅ |
| Re-sync com debounce | `TestRefreshPoolDebouncesAndIgnoresUnconfigured`, `TestRotateTriggersPoolRefresh` | ✅ |
| Admin API `proxy_auto`/refresh | `TestAdminProviderSetProxyAuto`, `TestAdminProviderSetProxyAutoRequiresField`, `TestAdminWebshareRefresh` | ✅ |
| `/proxies` expõe ban | `TestProxyListExposesBanPerAccount` | ✅ |
| Credenciais nunca retornam | `TestProxyModeAndAssignmentPersistWithoutCredentials` | ✅ |

## Casos executados

### Funcionais (caminho feliz)
- `inherit` com `proxy_auto=true` → `auto`; com `false` → `off`.
- Duas contas do mesmo provedor → IPs distintos; mesmo IP em provedores distintos.
- Rotação após falha de proxy/ban mantém a conta funcionando (mesma request).
- `proxy_auto` persiste e o console reflete no detalhe.

### Erro / limites
- Modo inválido → `400`; `proxy_auto` ausente → `400`; provedor inexistente → `404`.
- Endpoint/conta inexistentes ou IDs inválidos → `ErrNotFound` (404).
- Pool esgotado → `ErrProxyRequired` (fail-closed), nunca conexão direta.
- Webshare não configurado → `RefreshPool` no-op (sem erro).

### Inesperados / concorrência
- 8 atribuições concorrentes do mesmo provedor → exatamente **1** vence, 7 recebem `ErrProxyLimit`.
- Poison expirado deixa de excluir o IP.
- Poison de X não exclui o IP para Y.

## Achados

| # | Severidade | Descrição | Ação |
|---|---|---|---|
| 1 | Média | `GET /admin/v1/providers/{id}` mostrava a conta como `healthy` mesmo com o IP envenenado para o provedor | **Corrigido**: join em `proxy_poison`; health vira `quarantined`; `TestProviderAccountHealthReflectsPoison` |
| 2 | Baixa | Guia do usuário (markdown) atualizado, mas o HTML de `admin/public/guide` estava desatualizado | **Corrigido**: `node scripts/build-guide-html.js` (84 páginas) |

Sem regressões funcionais identificadas.

## Regressão

- `go test ./... -p 1 -count=1` → **ok**
- `go vet ./...` → **ok** · `golangci-lint run` → **0 issues** · `govulncheck ./...` → **0 no código**
- `npm test` → **68/68** · `npm run typecheck` → **ok** · `npm run lint` → **ok**

## Riscos residuais

- `upstream_ban` é heurístico (corpo do 403); falso positivo custa 24 h do IP, mitigado pelo re-sync.
- Sem health probe ativo; detecção é passiva (na falha) + sync periódico.
- Poison é por provedor; um IP pode seguir em uso por outro provedor (intencional).
