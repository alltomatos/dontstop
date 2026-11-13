# Issue tracker

O GitHub é a fonte persistente de escopo, critérios de aceite, dependências e status.

- **Repositório:** `alltomatos/dontstop` (`gh repo view`).
- **Issues:** use `gh issue create/list/view/close`; dependências com `--blocked-by <n>`.
- **Labels de triage:** ver [`triage-labels.md`](triage-labels.md).
- **Branches:** `feat/`, `fix/`, `refactor/`, `docs/`, `test/`, `chore/`; nunca commitar direto em `main`.
- **Commits:** Conventional Commits (`feat(proxy): ...`).
- **PR:** pequeno, testes verdes, evidência de observabilidade; merge só com CI verde.

## Regras

1. Nunca criar uma DAG apenas em memória ou em arquivo local quando a tarefa puder ser rastreada no GitHub.
2. Cada issue deve ter critérios de aceite verificáveis e referências aos artefatos (ADR/ADD/mockup).
3. Fechar a issue só após a suíte e o build verdes, com comentário de evidência (comandos + resultados).
4. `ESTADO_DEVELOPER.md` é apenas a visão operacional da DAG, nunca a fonte de escopo.
