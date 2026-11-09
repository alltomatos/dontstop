# Plano de Testes — Onboarding e Autenticação do Console

> Ciclo: revisão do diff de onboarding wizard + endurecimento de auth (2026-11-09)
> Executado contra: gateway Go local (código atual) + admin `next start` (build atual), DB `dontstop_qa`.

## Escopo

**Dentro**
- Onboarding de primeiro acesso: idioma → conta do owner → conexão do primeiro provedor (incluindo atalho "Skip").
- Criação do owner (`POST /admin/v1/setup`) e proteção de idempotência.
- Login/logout do console, cookie de sessão e rejeição de cookie forjado.
- Autorização das rotas `Next /api/admin/*` que executam ações privilegiadas.
- Rate limiting de login e propagação de `429`/`Retry-After`.
- Open redirect via `?next=`.
- Headers do edge (Caddy) quando aplicável.

**Fora (com justificativa)**
- Conectividade real com upstreams de IA: exige chaves de produção; validado apenas o contrato de criação de provedor.
- OAuth de provedores: marcado como não suportado no wizard (`supported: false`).
- Multi-tenancy/RLS: fase P1 do PRD.
- Regressão visual, Lighthouse e testes de carga: não afetados por este diff.
- Recuperação de senha: funcionalidade não existe ainda no escopo.

## Riscos priorizados

| # | Risco | Impacto | Cobertura |
|---|-------|---------|-----------|
| R1 | Bypass de autorização no console (cookie forjado) | Crítico | Playwright `admin-access-control.spec.sec.ts` |
| R2 | Corrida na criação do owner (dois owners) | Alto | Execução concorrente via API |
| R3 | Wizard sem saída após erro | Alto | Playwright TC-ONB-04 |
| R4 | Força bruta de senha | Alto | Playwright `login-hardening.spec.sec.ts` |
| R5 | Open redirect pós-login | Médio | Playwright + verificação do HTML |
| R6 | Vazamento de segredo em resposta/log | Médio | Playwright `sensitive-data.spec.sec.ts` |
| R7 | Falha de gateway confundida com "sem dados" | Médio | Revisão de código (`getConfiguredProviders`) |
| R8 | Acessibilidade do catálogo de provedores | Baixo | Revisão estática ARIA |

## Estratégia por camada

| Camada | Ferramenta | O que cobre |
|--------|-----------|-------------|
| Unitário | `go test ./...` | Limiter de login (janela, reset, isolamento por e-mail), auth, providers |
| Unitário FE | `npm test` (admin) | Locales, guia, links, preferências, URL pública |
| Integração/API | Playwright `request` | `/admin/v1/setup`, `/admin/v1/auth/login`, `/api/admin/*` negativos, 401/403/429 |
| E2E | Playwright + Chromium | Onboarding, login, dashboard, providers |
| Exploratório | HTTP direto (`HttpClient`) | Corrida de setup com duas requisições simultâneas |

## Casos de teste

### TC-ONB-01 — onboarding leva do idioma ao dashboard pulando provedor
- **Prioridade**: P0 · **Tipo**: funcional
- **Pré-condições**: instância sem owner
- **Passos**: `/onboarding` → escolher English → Continue → preencher nome/e-mail/senha (≥8) → Create account → Skip
- **Esperado**: passo 3 exibido; Skip leva a `/dashboard`
- **Obtido**: ✅ passou

### TC-ONB-02 — onboarding conecta o primeiro provedor
- **Prioridade**: P0 · **Tipo**: funcional
- **Pré-condições**: instância sem owner
- **Passos**: onboarding até o passo 3 → selecionar OpenAI → nome `openai-qa` → API key fictícia → Connect provider
- **Esperado**: redireciona a `/dashboard`; `/providers` lista OpenAI
- **Obtido**: ✅ passou

### TC-ONB-03 — senha curta não avança do passo 2
- **Prioridade**: P1 · **Tipo**: erro
- **Passos**: preencher senha `short` e submeter
- **Esperado**: permanece no passo 2 (validação HTML `minLength=8`)
- **Obtido**: ✅ passou

### TC-ONB-04 — voltar do passo 3 e reenviar a conta deixa o wizard travado
- **Prioridade**: P1 · **Tipo**: inesperado
- **Passos**: concluir a conta → no passo 3 clicar Back → clicar Create account novamente
- **Esperado**: retomar o passo 3 (setup já concluído) ou mensagem clara com caminho de saída
- **Obtido**: ✅ passou após a correção do BUG-01 (o wizard memoriza a conta criada e avança ao passo 3; `409` de outra sessão mostra aviso com link de login)

### TC-ONB-05 — recarregar o wizard após o setup redireciona ao login
- **Prioridade**: P2 · **Tipo**: inesperado
- **Obtido**: ✅ passou (`/onboarding` → `/login`)

### TC-ONB-06 — login com credenciais válidas abre o console
- **Prioridade**: P0 · **Tipo**: funcional
- **Obtido**: ✅ passou (`/login` → `/dashboard`)

### TC-UI-01 — topbar sem tenant/produção e rodapé neutro
- **Prioridade**: P2 · **Tipo**: funcional
- **Passos**: autenticar e abrir `/dashboard`
- **Esperado**: `.who` sem "tenant"/"production"/"produção"; `.sidebar-foot` = `dontstop · v0.1`
- **Obtido**: ✅ passou

### TC-SEC-01..12 — suíte de segurança (ver `tests/security`)
- Cookie forjado em `api-keys`, `providers/accounts`, `mcp/settings` → 401/403
- Força bruta → 429 com `Retry-After`
- `?next=//evil.example` descartado no HTML e no redirect autenticado
- Setup já concluído → 409
- Login 204 sem token no corpo, cookie `HttpOnly; SameSite=Strict`
- **Obtido**: 11 ✅, 1 ⏭ (headers do edge exigem `E2E_EDGE=true`)

### TC-CONC-01 — setup concorrente
- **Prioridade**: P1 · **Tipo**: inesperado
- **Passos**: DB sem owners; duas requisições simultâneas `POST /admin/v1/setup` com e-mails distintos
- **Esperado**: um `201` e um `409`
- **Obtido**: ✅ após a correção do BUG-02 — 3/3 execuções com `201` + `409` e `count(admin_users)=1`

## Relatos de bug

### BUG-01 — Wizard trava no passo 2 ao reenviar a conta depois do setup
- **Severidade**: S2 · **Prioridade**: P1
- **Ambiente**: Chromium (Playwright), admin build `next start`, gateway Go local, DB `dontstop_qa`
- **Passos**: concluir a conta no passo 2 → no passo 3 clicar Back → Create account
- **Esperado**: avançar ao passo 3 (setup já concluído) ou orientar o usuário com saída clara
- **Observado**: `POST /api/auth/setup` retorna `409`; a UI exibe `onboarding.error` ("Could not complete this step…") e permanece no passo 2, sem botão de avanço. Saída só recarregando a página, o que joga o usuário em `/login`.
- **Evidência**: caso TC-ONB-04; resposta 409 do gateway
- **Frequência**: sempre
- **Relacionado**: TC-ONB-04
- **Status**: ✅ corrigido — `accountCreated` evita o reenvio e avança ao passo 3; `409` residual exibe `onboarding.setupConflict` com link para `/login`. Re-testado em TC-ONB-04.

### BUG-02 — Duas requisições simultâneas de setup criam dois owners
- **Severidade**: S2 · **Prioridade**: P1
- **Ambiente**: gateway Go local, DB `dontstop_qa` sem owners
- **Passos**: disparar `POST /admin/v1/setup` em paralelo com `owner-a@` e `owner-b@`
- **Esperado**: exatamente um owner; a segunda chamada retorna `409`
- **Observado**: `201` para as duas; `SELECT count(*) FROM admin_users` = 2
- **Evidência**: `status1=201 status2=201`, `count=2`
- **Frequência**: reproduzido em 1 de 1 execuções (antes da correção)
- **Relacionado**: TC-CONC-01
- **Observação de segurança**: em instância recém-provisionada, um ator que alcance o console antes do administrador pode disputar a criação do owner.
- **Status**: ✅ corrigido — `pg_advisory_xact_lock(hashtext('dontstop:setup'))` serializa o `SELECT count(*)` + `INSERT`. Re-testado em 3/3 execuções (1×`201`, 1×`409`, 1 owner) e por `TestAuthServiceSetupIsAtomic`.

### BUG-03 — Provedores sem autenticação não têm filtro no onboarding
- **Severidade**: S4 · **Prioridade**: P3
- **Passos**: no passo 3, usar os chips de autenticação
- **Esperado**: um chip "No auth" ou filtro explícito
- **Observado**: só existiam `All`, `API key`, `OAuth`; `ollama`, `lm-studio` e `opencode-free` (`authType: "none"`) ficavam sem filtro dedicado
- **Evidência**: `admin/app/onboarding/onboarding-wizard.tsx`
- **Status**: ✅ corrigido — chip "No auth" adicionado (reaproveita `onboarding.auth.none`)

### BUG-04 — Catálogo de provedores usa ARIA de listbox em botões
- **Severidade**: S3 · **Prioridade**: P2
- **Observado**: o container tinha `role="listbox"` mas os itens são `<button aria-pressed>` sem `role="option"`/`aria-selected`; leitores de tela recebiam semântica inconsistente
- **Evidência**: `admin/app/onboarding/onboarding-wizard.tsx`
- **Status**: ✅ corrigido — container passou a `role="group"` (os itens continuam botões toggle com `aria-pressed`)

### BUG-05 — `DONTSTOP_CONSOLE_AUTH` tem três interpretações divergentes
- **Severidade**: S3 · **Prioridade**: P3
- **Observado**: `proxy.ts` trata qualquer valor ≠ `off` como "exigir sessão"; `hasValidConsoleSession` usa a mesma regra; `consoleAuthRequired()` retorna `true` apenas para `"required"` e não é usado; compose e `.env.example` usam `on`
- **Impacto**: configuração ambígua e código morto; um operador pode acreditar que `required` é necessário
- **Evidência**: `admin/proxy.ts`, `admin/lib/admin-auth.ts`, `docker-compose.yml`, `admin/.env.example`
- **Status**: ✅ corrigido — `admin/lib/console-auth.ts` centraliza a regra (`off` desliga, qualquer outro valor exige sessão); `consoleAuthRequired` removido

### BUG-06 — Falha de gateway é exibida como "nenhum provedor configurado"
- **Severidade**: S3 · **Prioridade**: P2
- **Observado**: `getConfiguredProviders` converte 401, 500, timeout e erro de rede em `[]`; a UI não diferencia "vazio" de "falhou"
- **Evidência**: `admin/lib/admin-api.ts`
- **Impacto**: diagnóstico enganoso; usuário autenticado pode achar que perdeu a configuração
- **Status**: ✅ corrigido — `getConfiguredProvidersResult()` expõe `error`; dashboard e página de provedores exibem banner `common.gatewayError`

### BUG-07 — Sessão é derrubada quando o gateway demora mais de 3 s
- **Severidade**: S3 · **Prioridade**: P2
- **Observado**: `hasValidConsoleSession` usa `AbortSignal.timeout(3000)` e redireciona ao login em qualquer falha
- **Evidência**: `admin/lib/admin-auth.ts`
- **Impacto**: indisponibilidade transitória do gateway vira logout aparente
- **Status**: ✅ corrigido — `consoleSessionState()` distingue `authenticated`/`unauthenticated`/`unavailable`; só `unauthenticated` (401/403) redireciona ao login, com timeout de 5 s

### BUG-08 — Dashboard exibia métricas fabricadas quando faltavam dados
- **Severidade**: S2 · **Prioridade**: P1
- **Observado**: com `observability`/`cost` nulos, o dashboard mostrava números fixos (`72.800` requests, `28 ms`, `US$ 312`, deltas `+8%`, incidentes e top models fictícios) como se fossem reais; a tabela de saúde derivava P95/erros/health do índice da linha
- **Evidência**: `admin/app/(console)/dashboard/dashboard-view.tsx` (revisão de código)
- **Impacto**: decisões operacionais baseadas em telemetria inventada
- **Status**: ✅ corrigido — valores reais ou `—`, listas vazias usam `common.noData`, deltas/incidentes fictícios removidos e banner de erro do gateway adicionado

## Análise de causa raiz

### RCA — BUG-01
- **Causa raiz**: o wizard trata o passo 2 como estado definitivo de criação, sem reconhecer `409` (setup já concluído) nem lembrar que o owner já foi criado na mesma sessão.
- **Por que não foi detectado antes**: os testes existentes cobriam apenas o caminho feliz; não havia caso de "voltar e reenviar" (duplo submit).
- **Ação preventiva (implementada)**: guarda `accountCreated` evita reenvio e avança ao passo 3; `409` exibe aviso com link de login.
- **Teste de regressão**: TC-ONB-04 (sem `test.fail`) exige o avanço ao passo 3 — verde.

### RCA — BUG-02
- **Causa raiz**: `AuthService.Setup` faz `SELECT count(*)` e insere na mesma transação sem serialização; a tabela não restringe "um owner por tenant".
- **Por que não foi detectado antes**: só havia teste sequencial do setup.
- **Ação preventiva (implementada)**: `pg_advisory_xact_lock(hashtext('dontstop:setup'))` antes do `SELECT count(*)`, mantendo a decisão e a inserção na mesma transação.
- **Teste de regressão**: `TestAuthServiceSetupIsAtomic` (duas goroutines concorrentes → 1 sucesso, 1 `ErrSetupComplete`, 1 owner).

### RCA — BUG-06
- **Causa raiz**: o fallback de mock foi removido, mas o contrato de erro não foi substituído por um estado de erro explícito.
- **Ação preventiva**: tipar o retorno como `{ data } | { error }` e renderizar estado de erro com retry.
- **Teste de regressão**: teste de servidor com gateway indisponível (mock de fetch) esperando estado de erro.

## Critérios de saída

- Todos os casos P0/P1 executados e passando: **atingido** — TC-ONB-01/02/03/04/05/06, TC-UI-01, TC-CONC-01 e TC-SEC verdes.
- Nenhum bug S1/S2 aberto: **atingido** — BUG-01, BUG-02 e BUG-08 corrigidos e re-testados.
- P2/P3 corrigidos neste ciclo: BUG-03, BUG-04, BUG-05, BUG-06, BUG-07.

## Baseline executada

| Suíte | Comando | Resultado |
|-------|---------|-----------|
| Go | `go test ./...` / `go vet ./...` | ✅ |
| Go integração | `TEST_DATABASE_URL=… go test ./internal/admin` | ✅ inclui `TestAuthServiceSetupIsAtomic` |
| Admin unit | `npm test` (admin) | ✅ 31/31 |
| E2E funcional | `npm run test:e2e` (tests/e2e) | ✅ TC-ONB-01/02/03/04/05/06 e TC-UI-01 |
| Segurança | `npm run test:sec` (tests/security) | ✅ 11, ⏭ 1 |

### Como reproduzir

```powershell
# gateway com o código atual + DB de QA
docker exec dontstop-pg psql -U postgres -d postgres -c "CREATE DATABASE dontstop_qa;"
$env:DATABASE_URL="postgres://postgres:postgres@localhost:55432/dontstop_qa?sslmode=disable"
go run ./cmd/migrate            # dentro de gateway/
# subir gateway em 8081 e admin (next start) apontando DONTSTOP_GATEWAY_URL=http://localhost:8081

# reset entre casos que exigem primeiro acesso
docker exec dontstop-pg psql -U postgres -d dontstop_qa -c "TRUNCATE admin_sessions, admin_users, providers, provider_accounts, provider_models CASCADE;"

# execução
$env:E2E_BASE_URL="http://localhost:3101"
$env:E2E_FRESH_INSTANCE="true"; npm run test:e2e -- -g "TC-ONB-01"
```
