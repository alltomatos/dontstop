# Security — Isolamento de proxy por provedor (Gate 2)

> Feature: ADR-024 · Issues #53–#61 · Data: 2026-11-13.
> Complementa [`p0-proxy-isolation-qa.md`](p0-proxy-isolation-qa.md) e o
> [runbook](proxy-isolation-runbook.md).

## Superfície de ataque

| Superfície | Rotas |
|---|---|
| Admin API (gateway) | `PATCH /admin/v1/providers/{id}`, `PATCH .../accounts/{accountId}`, `PUT|DELETE .../proxy`, `GET /admin/v1/proxies`, `POST /admin/v1/proxies/webshare/{sync,refresh}` |
| Console (pass-through) | `admin/app/api/admin/providers/[id]`, `.../proxies/webshare/refresh` |
| Data plane | seleção/rotação de proxy no `/v1/chat/completions` |

## Auditoria estática

| Controle | Verificação | Resultado |
|---|---|---|
| Autorização | Todas as rotas novas usam `requireManage`; sem `manage` → 403 | ✅ `TestAdminProviderSetProxyAutoRequiresManageScope`, `TestAdminProxyAssignRequiresManageScope`, `TestAdminWebshareRefreshRequiresManageScope` |
| Isolamento por tenant | Queries com `tenant_id`; atribuição cross-tenant → 404 | ✅ `TestProxyAssignmentLimitAndTenantIsolation`, `TestAssignRejectsUnknownEndpointWithoutLeaking` |
| Validação de entrada | `proxy_auto` bool obrigatório; `proxy_mode` no enum; IDs UUID | ✅ `TestAdminProviderSetProxyAutoRequiresField`, `TestParseProxyMode`, `TestAssignRejectsSQLiLookingID` |
| SQL injection | Queries parametrizadas; IDs inválidos rejeitados antes do cast | ✅ `TestAssignRejectsSQLiLookingID` |
| XSS | Console renderiza label/endereço como texto (React escapa); sem `dangerouslySetInnerHTML` | ✅ inspeção + Playwright |
| Exposição de segredos | `ProxyEndpoint`/`ProxyEndpointAccount` omitem username/password; credenciais cifradas (AES-256-GCM) | ✅ `TestProxyModeAndAssignmentPersistWithoutCredentials` |
| SSRF | `BuildProxyURL` rejeita scheme/credenciais no endereço; refresh chama apenas a API do Webshare | ✅ `TestBuildProxyURLValidatesEndpoint` |
| Abuso / rate limit | Refresh com debounce (30 s + Redis); rotas admin exigem `manage` | ✅ `TestRefreshPoolDebouncesAndIgnoresUnconfigured` |
| Falha fechada | `manual`/`auto`/`required` sem proxy saudável → erro, nunca conexão direta | ✅ `TestSelectProxyManualAndAutoFailClosed` |
| Corrida | 8 assigns concorrentes do mesmo provedor → 1 vence | ✅ `TestConcurrentAssignSameProviderKeepsSingleAssignment` |

## Testes E2E de segurança (Playwright)

Arquivo: `tests/e2e/proxy-isolation.spec.sec.ts` (`@security`, 4 cenários):

1. `/proxies` sem sessão → redirect para `/login`.
2. `GET /api/admin/proxies` sem sessão → não-200 e **sem** `username/password`.
3. `POST /api/admin/proxies/webshare/refresh` sem sessão → não-200.
4. `PATCH /api/admin/providers/{id}` sem sessão → não-200.

Execução (requer console + gateway):

```bash
cd tests/e2e
E2E_SECURITY=true npx playwright test --grep @security
```

> Sem `E2E_SECURITY=true`, os cenários são `skip` (não quebram CI sem stack).

## Achados

| # | Severidade | Achado | Ação |
|---|---|---|---|
| — | — | Nenhuma vulnerabilidade crítica/alta | — |

Observações de baixo risco:

- O console (Next) retorna `503 admin_api_not_configured` sem chave admin; é fail-safe e não expõe dados.
- `upstream_ban` é heurístico (corpo do 403); um falso positivo custa até 24 h do IP, mitigado pelo re-sync.
- `golangci-lint` exclui `internal/admin/combos.go` (WIP pré-existente, untracked) apenas do linter `unused`.

## Regressão de segurança

- `go test ./... -p 1 -count=1` → **ok**
- `go vet ./...` → **ok** · `golangci-lint run` → **0 issues** · `govulncheck ./...` → **0 no código**
- `npm test` → **68/68** · `npm run typecheck` → **ok** · `npm run lint` → **ok**
- Playwright `@security` → **4 testes listados** (executam com `E2E_SECURITY=true`).
