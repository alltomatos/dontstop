# Triage labels

Estados de triagem usados no GitHub. Um item novo entra em `needs-triage` e caminha para um estado
terminal antes de ser executado.

| Label | Significado | Próximo passo |
|---|---|---|
| `needs-triage` | Ainda não classificado ou desbloqueio indefinido | Triar escopo, prioridade e dependências |
| `needs-info` | Falta informação para executar | Pedir dados ao solicitante |
| `ready-for-agent` | Desbloqueado, escopo e critérios de aceite claros | Executar via `/developer` (TDD) |
| `ready-for-human` | Exige decisão humana (produto, risco, aprovação) | Revisão humana |
| `wontfix` | Fora de escopo por decisão | Registrar motivo e fechar |

## Fluxo

```
new → needs-triage
        ├─ needs-info → (resposta) → needs-triage
        ├─ ready-for-agent → (implementado + testes) → fechado
        ├─ ready-for-human → (decisão) → ready-for-agent | wontfix
        └─ wontfix → fechado
```

Regras:

- Um item bloqueado por outra issue **não** é `ready-for-agent`; fica `needs-triage` com `Blocked by`.
- `ready-for-agent` exige critérios de aceite e artefatos de referência (ADR/ADD/mockup/contrato).
- Fechar exige evidência (testes/build/lint) no comentário final.
