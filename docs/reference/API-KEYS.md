# API Manager (console) — contrato e mapa de consumo

**Escopo:** UI real de API keys do console (`admin`) e as rotas do gateway usadas por ela.
**Fonte de verdade:** gateway/Postgres; a UI apenas apresenta e envia comandos.

## 1. Mapa UI → Next → gateway

| Ação da UI | Rota Next (sessão `dontstop_session`) | Rota do gateway | Efeito |
|---|---|---|---|
| Listar chaves | `GET /api/admin/api-keys` | `GET /admin/v1/api-keys` | Lista sem segredo, com status calculado |
| Catálogo de escopos | `GET /api/admin/api-keys/scopes` | `GET /admin/v1/api-keys/scopes` | Escopos aceitos na criação/edição |
| Detalhe | `GET /api/admin/api-keys/{id}` | `GET /admin/v1/api-keys/{id}` | Metadados sem segredo |
| Criar | `POST /api/admin/api-keys` | `POST /admin/v1/api-keys` | Cria chave e devolve o segredo **uma vez** |
| Editar | `PUT /api/admin/api-keys/{id}` | `PUT /admin/v1/api-keys/{id}` | Substitui nome, escopos, expiração, IP allowlist e limites |
| Rotacionar | `POST /api/admin/api-keys/{id}/rotate` | `POST /admin/v1/api-keys/{id}/rotate` | Novo segredo, metadados preservados |
| Revogar | `POST /api/admin/api-keys/{id}/revoke` | `POST /admin/v1/api-keys/{id}/revoke` | Marca `revoked_at`; falha fechada no gateway |

Os proxies Next ficam em `admin/app/api/admin/api-keys/` e usam `getAdminKey()` (cookie de sessão ou
`DONTSTOP_ADMIN_API_KEY`). Nenhuma regra de quota, hash ou validação de domínio é duplicada no frontend;
o gateway valida e persiste.

## 2. Campos

| Campo | Origem/regra |
|---|---|
| `name` | Obrigatório, até 80 caracteres |
| `prefix` | Prefixo público da chave (nunca o segredo completo) |
| `scopes` | Pelo menos um escopo do catálogo; `chat:write`, `models:read`, `embeddings:write`, `web:read`, `usage:read`, `combos:read`, `combos:write`, `manage` |
| `status` | Calculado: `active`, `expired` (`expires_at` no passado) ou `revoked` |
| `expires_at` | `NULL` = sem expiração; datas passadas são rejeitadas |
| `ip_allowlist` | Lista de IPs/CIDRs, até 64 entradas; vazia permite qualquer IP |
| `rpm_limit`, `tpm_limit` | Inteiros não negativos |
| `usd_monthly_limit`, `usd_daily_limit`, `usd_weekly_limit` | Valores não negativos |
| `created_at` | Criação |
| `last_rotated_at` | Preenchido apenas por rotação |
| `last_used_at` | Derivado da última requisição da chave |

O segredo completo (`key`) só aparece nas respostas de **criação** e **rotação**. Listagem, detalhe e
edição nunca o devolvem.

## 3. Segurança e auditoria

- Hash Argon2id; o segredo não é armazenado em claro.
- Revogação e expiração são verificadas pelo `auth.Verifier` antes de qualquer chamada upstream.
- Eventos em `audit_outbox`: `api_key.create`, `api_key.update`, `api_key.rotate`, `api_key.revoke`.
  O payload contém tenant, ator, id/nome, escopos, allowlist e limites; **nunca** o segredo.
- A UI limpa o estado do segredo ao fechar o modal; não há rota de reveal.

## 4. UI

- Página: `admin/app/(console)/api-manager/`.
- Lista com loading, vazio e erro reais; KPIs derivados das chaves recebidas.
- Criação/edição em modal; validação no cliente e no servidor.
- Criação/rotação exibem o segredo uma única vez com aviso explícito e `CopyButton`.
- Rotação e revogação exigem confirmação em modal.
- i18n em `admin/locales/*.json`; fallback para `en`.
- Nenhum dado mock permanece na página.

## 5. Testes

- `gateway/internal/admin/api_keys_integration_test.go`: criação sem expiração, lista, atualização,
  rotação, revogação, status expirado, validação de escopo/IP e auditoria sem segredo.
- `gateway/internal/api/admin_api_keys_test.go`: rotas de lista/catálogo/get/put/rotate/revoke e
  garantia de que a listagem não retorna o segredo.
- `admin/tests/*`: catálogo/locales; a UI é exercitada manualmente/Playwright contra o gateway real.
