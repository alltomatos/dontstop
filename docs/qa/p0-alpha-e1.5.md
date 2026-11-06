# QA — P0-α E1.5

> Gate de `/v1/chat/completions` JSON + SSE.
> Data: 2026-11-06. Issue: #6.

## Escopo testado

- Rota autenticada com escopo `chat:write`.
- Validação de JSON, `model`, `messages`, roles, temperature e `max_tokens`.
- Resposta JSON OpenAI-compatible e SSE com `data: [DONE]`.
- Idempotência não streaming em Redis, com chave escopada por tenant.
- Uso do roteador E1.6, resolução de credencial E1.12 e adapters E1.4.
- Erros de provider/roteamento mapeados para 4xx/5xx sem payload sensível.
- Limite de body de 1 MiB.

## Fora de escopo

- Cache exato (E1.7), usage/custo (E1.8), OTel/carga (E1.9).
- DLP no streaming e rate limiting de entrada.
- Ferramentas/tool calls completas e Responses API.
- Retry depois do primeiro token (proibido por design).

## Casos executados

| ID | Caso | Resultado |
|---|---|---|
| QA-44 | Sem `chat:write` retorna 403 | PASS |
| QA-45 | JSON inválido/messages vazio retorna 400 | PASS |
| QA-46 | Resposta JSON tem `chat.completion`, choice e usage | PASS |
| QA-47 | `stream:true` retorna SSE e `[DONE]` | PASS |
| QA-48 | Idempotência retorna resposta armazenada | PASS |
| QA-49 | Chave de idempotência não cruza tenants | PASS |
| QA-50 | Failover do router é usado antes do primeiro token | PASS |
| QA-51 | Falha pós-primeiro-token não faz retry | PASS |
| QA-52 | Body acima do limite é rejeitado | PASS |
| QA-53 | `go test -race -p 1 ./...` com Postgres/Redis | PASS |
| QA-54 | `golangci-lint`, `go vet`, `go build` | PASS |
| QA-55 | `govulncheck` com Go 1.26.10 | PASS, 0 vulnerabilidades |

## Auditoria de segurança

- `chat:write` é exigido no middleware; tenant vem do principal autenticado.
- Body limitado a 1 MiB; decoder padrão sem `eval`/comandos.
- Credencial só existe em memória após decrypt; não é serializada na resposta.
- Erros SSE não incluem erro bruto do provider.
- Idempotência usa prefixo de tenant e TTL de 24 h.
- Nenhum retry após início do stream evita duplicação de conteúdo.

## Riscos residuais

- Ainda não há rate limiting de entrada nem quota por chave/modelo.
- Conteúdo/estrutura de mensagens além de role/model não é validado profundamente.
- Cancelamento do cliente fecha o stream, mas não há métrica dedicada nesta slice.
- DLP não está no caminho do chat P0-α.
