# QA — Isolamento de proxy por provedor (ADR-024)

> Referências: [`05-ADRS.md`](../05-ADRS.md) (ADR-024), [`02-DER.md`](../02-DER.md) §3.3,
> [`architecture/ADD.md`](../architecture/ADD.md) §6.4, [`reference/WEBSHARE-PROXY-PLAN.md`](../reference/WEBSHARE-PROXY-PLAN.md).

## Invariantes

1. **1 conta por IP por provedor** — `UNIQUE (tenant_id, provider_id, proxy_endpoint_id)`.
2. **Reuso entre provedores diferentes é permitido** — o ban é por provedor.
3. **Poison por `(tenant, provedor, endpoint)`** — um IP banido por X continua elegível para Y.
4. **Modo efetivo** — `inherit` → `auto` se `providers.proxy_auto`, senão `off`.
5. **Fail-closed** — `manual`/`auto`/`required` nunca caem para conexão direta.
6. **Rotação transparente** — na mesma request, antes do primeiro chunk (máx. 2).
7. **Credenciais** — nunca retornadas em resposta, log ou trace.
8. **Isolamento por tenant** — nenhuma atribuição/IP cruza tenants.

## Matriz de testes

| Camada | Teste | Cobre |
|---|---|---|
| Unit | `TestParseProxyMode` | `inherit/off/manual/auto` + legados |
| Unit | `TestResolveEffectiveMode` | herança do default do provedor |
| Unit | `TestSelectProxyManualAndAutoFailClosed` | fail-closed sem proxy saudável |
| Unit | `TestClassifyProxyFailure` | classes e TTLs de poison |
| Unit | `TestLooksLikeBan` | sinal de ban em 403 |
| Integration | `TestProxyIsolationSchema` | colunas, CHECK, `proxy_poison`, idempotência |
| Integration | `TestProxyAssignmentOnePerProviderWithCrossProviderReuse` | 1:1 por provedor + reuso cross-provider |
| Integration | `TestProxyAssignmentDedupMigration` | desdobramento de atribuições compartilhadas |
| Integration | `TestPostgresCandidatesResolveInheritedProxyMode` | modo efetivo no roteamento |
| Integration | `TestRecordBanPoisonsOnlyTheProvider` | poison por provedor + expiração |
| Integration | `TestProxyListExposesBanPerAccount` | ban exposto no `GET /admin/v1/proxies` |
| Integration | `TestRefreshPoolDebouncesAndIgnoresUnconfigured` | debounce + no-op sem config |
| Integration | `TestRotateTriggersPoolRefresh` | re-sync assíncrono na rotação |
| Security | `TestConcurrentAssignSameProviderKeepsSingleAssignment` | corrida: 8 assigns → 1 vence |
| Security | `TestProxyAssignmentLimitAndTenantIsolation` | cross-tenant rejeitado |
| Security | `TestProxyModeAndAssignmentPersistWithoutCredentials` | credenciais não vazam |
| Security | `TestAssignRejectsUnknownEndpointWithoutLeaking` | IDs inválidos → 404 sem vazamento |
| Chat | `TestChatRotatesProxyBeforeProviderAuth`, `TestChatRotatesOnUpstreamBan`, `TestChatDoesNotRotateRateLimit`, `TestChatDoesNotRotateProviderAuthErrors` | rotação só em falha de proxy/ban |
| Console | `npm test` (locales parity), `npm run typecheck`, `npm run lint` | i18n, tipos e lint |

## Comandos

```bash
# Gateway (Postgres de teste em localhost:55432)
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:55432/dontstop?sslmode=disable" \
  go test ./... -p 1 -count=1
go vet ./...
gofmt -l internal/

# Admin
cd admin && npm test && npm run typecheck && npm run lint
```

## Resultado (2026-11-13)

- `go test ./... -p 1 -count=1` → **ok** (todos os pacotes).
- `go vet ./...` → **ok**.
- `npm test` → **68/68**; `npm run typecheck` → **ok**; `npm run lint` → **ok**.

## Riscos residuais

| Risco | Mitigação |
|---|---|
| Pool esgotado (sem IP livre para o provedor) | `auto`/`manual` falham fechado; re-sync Webshare + alerta `proxy.candidate_miss` |
| Falso positivo de ban (403 heurístico) | poison de 24h só com sinal no corpo; re-sync repõe IPs |
| Storms de re-sync | debounce 30 s (processo + Redis) |
| Poison de um provedor bloquear outro | poison é por `(provedor, endpoint)`; testado |
| Credencial de proxy em log | cifrada em repouso; nunca serializada na API; teste de redaction |
