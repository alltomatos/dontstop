# QA — P0-α E1.8

> Gate de usage/custo, outbox e NATS JetStream.
> Data: 2026-11-07. Issue: #9.

## Escopo testado

- Preço por modelo/provider (`input_price_per_million`, `output_price_per_million`).
- `requests` + `usage_daily` + `usage_outbox` na mesma transação.
- Idempotência por `request_id`; repetição não duplica usage/custo.
- Outbox publisher com retry quando o broker falha.
- Publicação NATS com `Msg-Id = event_id` e deduplicação JetStream.
- Usage record para chat JSON e streaming.
- Payload do evento sem prompt/resposta/PII.

## Fora de escopo

- Consumidor NATS interno; os agregados são escritos transacionalmente no Postgres.
- Batch multi-evento (cada request faz upsert atômico em `usage_daily`).
- ClickHouse/analytics histórico (P2).
- Exportação SIEM/chargeback (P1).

## Casos executados

| ID | Caso | Resultado |
|---|---|---|
| QA-70 | Custo de 1k/500 tokens com preço fixture = US$ 0,002 | PASS |
| QA-71 | Mesmo `request_id` não duplica request/aggregate/outbox | PASS |
| QA-72 | `usage_daily` é atualizado atomicamente | PASS |
| QA-73 | Outbox falha sem marcar publicado e retenta depois | PASS |
| QA-74 | NATS publica com deduplicação por Msg-Id | PASS |
| QA-75 | Chat JSON registra tokens in/out | PASS |
| QA-76 | Streaming registra tokens ao final | PASS |
| QA-77 | Payload não contém prompt/content | PASS |
| QA-78 | `go test -race -p 1 ./...` com Postgres/Redis/NATS | PASS |
| QA-79 | `golangci-lint`, `go vet`, `go build` | PASS |
| QA-80 | `govulncheck` com Go 1.26.10 | PASS, 0 vulnerabilidades |

## Auditoria de segurança

- Evento contém IDs, modelo, tokens, custo, latência e status; nunca mensagens/respostas.
- `event_id` é UUID gerado no banco e vira `Msg-Id`; duplicata não duplica no stream.
- Falha de NATS não perde o evento: permanece no outbox até publicação.
- Custo usa preço do catálogo, não valor enviado pelo cliente.
- Transação impede request sem outbox ou outbox sem request.

## Riscos residuais

- Agregados são por request, não batch; throughput alto pode exigir batch/ClickHouse.
- NATS consumer/analytics externo não está no repositório.
- Preço `float64` no cálculo; migrar para decimal/inteiro de micros no P1 se necessário.
- Sem reconciliação com fatura do provider; apenas tabela de preços interna.
