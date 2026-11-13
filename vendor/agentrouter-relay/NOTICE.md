# AgentRouter relay (vendored)

- **Source:** <https://github.com/alltomatos/allcode> —
  `packages/allcode/src/provider/agentrouter/relay.py` (and `manager.ts` for reference).
- **Commit:** `33a0c8b38de2bb2b918847f428ee36c783dce346` (2026-11-12).
- **License:** MIT — Copyright (c) 2025 opencode. See [`LICENSE`](LICENSE).
- **Decision:** [ADR-025](../../docs/adr/0025-agentrouter-relay-sidecar.md).

## Por que existe

O AgentRouter aplica um WAF de **fingerprint de conexão**: clientes genéricos (Node `fetch`/undici,
`curl`, `httpx` assíncrono e o `net/http` do Go) recebem `401 unauthorized client detected` mesmo com
chave válida. Apenas o SDK Python `anthropic` **síncrono** passa. Este relay é um sidecar interno que
recebe Anthropic Messages do gateway e reemite pelo SDK.

## Mudanças em relação ao AllCode

1. **Chave via `x-api-key`** (por requisição), repassada pelo gateway; `401` sem chave. Nenhuma chave
   em disco/env no Swarm. `AGENTROUTER_API_KEY` é apenas fallback de uso standalone.
2. **Sem tradução**: removida a tradução best-effort e a instrução pt-BR. A tradução para inglês é
   **fail-closed** no adapter `agentrouter` do gateway.
3. `AGENTROUTER_BIND` (default `127.0.0.1`; no sidecar `0.0.0.0`, sem porta publicada).
4. `RELAY_VERSION = 3` e catálogo estático atualizado (`gpt-6.1sol`, `gpt-6-astra`,
   `claude-opus-5.5` etc.). Disponibilidade é dinâmica (402/503), não hardcoded.
5. `Dockerfile`/`requirements.txt` com dependências **pinadas no build**; nada de `pip` em runtime.
6. `manager.ts` é **referência** (ciclo de vida do app AllCode); o dontstop usa o sidecar Swarm.

## Build / run

```bash
docker build -t registry/dontstop-agentrouter-relay:latest vendor/agentrouter-relay
docker run --rm -p 7187:7187 registry/dontstop-agentrouter-relay:latest
# produção: deploy/stack.yml (serviço agentrouter-relay, rede app, sem porta pública)
```

## Testes

No host (fora da imagem), com as deps instaladas em um venv:

```bash
python -m pip install -r requirements.txt
python -m unittest -v test_relay.py
```
