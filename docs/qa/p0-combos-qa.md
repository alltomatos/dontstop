# QA — Combos (pré-commit)

> Data: 2026-11-13 · Escopo: WIP de combos (migração 0019, admin/API/console) + ambiente local.
> Nenhuma chave real é reproduzida neste documento.

## 1. Suíte e build

| Etapa | Comando | Resultado |
|---|---|---|
| Gateway build | `go build ./...` | ✅ |
| Gateway vet | `go vet ./...` | ✅ |
| Gateway testes (unit + integração, inclui combos) | `TEST_DATABASE_URL=... go test ./... -p 1 -count=1` | ✅ todos os pacotes |
| Gateway lint | `golangci-lint run ./...` | ✅ 0 issues |
| Gateway vuln | `govulncheck ./...` | ✅ 0 no código |
| Admin testes | `npm test` | ✅ 68/68 |
| Admin typecheck | `npm run typecheck` | ✅ |
| Admin lint | `npm run lint` | ✅ |
| Admin build | `npm run build` | ✅ (rotas `/combos`, `/api/admin/combos`, `/api/admin/combos/[...path]`) |

## 2. Validação com chaves reais (via env/HTTP, sem persistência de longo prazo)

Fonte: `D:\apikey-tests.txt` (fora do repositório). Valores nunca impressos.

| Serviço | Verificação | Resultado |
|---|---|---|
| OpenRouter #1 | `GET /api/v1/key` | ✅ válida (uso reportado) |
| OpenRouter #2 | `GET /api/v1/key` | ✅ válida (uso 0) |
| OpenCode Go (alltomatos) | `GET /v1/models` | ✅ 30 modelos |
| OpenCode Go (ronaldo davi) | `GET /v1/models` | ✅ 30 modelos |
| Webshare | `GET /api/v2/proxy/config/` + `/proxy/list/` | ✅ plano 14210600 · 100 proxies |
| AgentRouter | `POST /v1/messages` (`x-api-key`, `Bearer`, raw) | ❌ **401** |

**Integração pelo gateway** (provider criado, testado e **removido** — `provider_deleted`):

| Provider | Teste | Modelos | Latência |
|---|---|---|---|
| OpenRouter #1 | ✅ | 436 | 124 ms |
| OpenRouter #2 | ✅ | 436 | 120 ms |
| OpenCode alltomatos | ✅ | 30 | 223 ms |
| OpenCode ronaldo | ✅ | 30 | 208 ms |
| Webshare (`/proxies/webshare/test`, sem persistir) | ✅ | 100 proxies | — |

### Pendências / falhas
- **AgentRouter 401** em três esquemas de auth (`x-api-key`, `Authorization: Bearer`, raw). Chave inválida/expirada ou endpoint do relay divergente — **não inventada/substituída**; requer nova chave ou confirmação do fornecedor. O catálogo/adapters não foram alterados por causa disso.
- Sem chaves para **OpenAI, Anthropic, Gemini, Azure, Bedrock** no arquivo → não validados nesta rodada.
- Observação de segurança: o `label` retornado por `GET /api/v1/key` do OpenRouter contém a própria chave (truncada). O gateway **não** usa esse endpoint (usa `/models`), mas qualquer log do label vazaria a chave — não logar.

## 3. Ambiente local

| Serviço | URL/porta | Observação |
|---|---|---|
| Admin (Next dev) | `http://localhost:3000` | dev para cookie de sessão funcionar em HTTP |
| Gateway (build local) | `http://localhost:8081` | DB/Redis locais; inclui combos |
| Postgres | `localhost:55432` | `postgres/postgres`, db `dontstop` |
| Redis | `localhost:56379` | sem senha |
| Owner do console | `owner@dontstop.local` / `dontstop-qa-123` | criado via `/admin/v1/setup` |

> O gateway Docker em `:8080` é outro ambiente; use **8081** para este build.
> Artefatos locais (não versionados): `%TEMP%\allcode\dontstop-master-key.txt`,
> `%TEMP%\allcode\dontstop-admin-token.txt`, logs `gateway-8081.log` / `admin-3000.log`.

## 4. Roteiro manual

### UI
1. Abra `http://localhost:3000/login` e entre com `owner@dontstop.local` / `dontstop-qa-123`.
2. Sidebar → **Combos** (`/combos`): deve listar **Demo Combo** (`active`, 1 membro).
3. **Criar combo**: slug `qa-combo-1`, nome `QA Combo`, estratégia `priority_round_robin`, `max_attempts=3`; adicione o membro `demo-openai / gpt-4.1`; salve (nasce `draft`) e **ative**.
4. **Negativos**: slug inválido (`QA Combo!`) → erro; membro duplicado → conflito; ativar combo sem membros → conflito.
5. Sem sessão, `/combos` redireciona para `/login`.

### API (gateway 8081)
```bash
TOKEN=$(curl -s -X POST http://localhost:8081/admin/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"owner@dontstop.local","password":"dontstop-qa-123"}' | jq -r .token)

curl -s http://localhost:8081/admin/v1/combos/catalog -H "Authorization: Bearer $TOKEN"
curl -s http://localhost:8081/admin/v1/combos         -H "Authorization: Bearer $TOKEN"

# criar (troque <PROVIDER_ID> pelo id do catálogo)
curl -s -X POST http://localhost:8081/admin/v1/combos \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"slug":"qa-combo-2","name":"QA Combo 2","strategy":"priority_round_robin",
       "failover_enabled":true,"max_attempts":3,
       "members":[{"provider_id":"<PROVIDER_ID>","model_id":"gpt-4.1",
                   "priority":10,"position":1,"weight":1,"enabled":true,"constraints":{}}]}'

curl -s -X POST http://localhost:8081/admin/v1/combos/<ID>/activate -H "Authorization: Bearer $TOKEN"
curl -s -X POST http://localhost:8081/admin/v1/combos/<ID>/deactivate -H "Authorization: Bearer $TOKEN"
curl -s -X DELETE http://localhost:8081/admin/v1/combos/<ID> -H "Authorization: Bearer $TOKEN"
```

### Pass-through do console
```bash
curl -s -c /tmp/ds.cookies -X POST http://localhost:3000/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"owner@dontstop.local","password":"dontstop-qa-123"}'
curl -s -b /tmp/ds.cookies http://localhost:3000/api/admin/combos
```

### Chat real através do combo
O provider `demo-openai` usa credencial fictícia (CRUD/ativação apenas). Para testar chat real:
cadastre um provider com uma chave real (via UI/API) e crie o combo com ele; depois
`POST /v1/chat/completions` com `{"model":"combo/<slug>",...}` e uma API key com `chat:write`.
Isso consome tokens do provedor.

## 5. Limpeza e segurança

- Providers temporários `qa-*` removidos (0 no banco); credenciais cifradas descartadas em cascata.
- Chaves reais **não** foram gravadas em arquivos do repositório nem em logs; uso transitório via variáveis/HTTP.
- Recomenda-se **rotacionar/excluir** `D:\apikey-tests.txt` após a validação.
