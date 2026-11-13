# ADR-025 — Sidecar relay vendorizado para o AgentRouter (WAF de fingerprint)

**Status:** Aceito
**Data:** 2026-11-13
**Emenda:** a regra "sem relay de referência em runtime" do `AGENTS.md` §8/§14 e a decisão
"o relay Python não é adotado" de [`AGENTROUTER-ALLCODE-RESEARCH.md`](../reference/AGENTROUTER-ALLCODE-RESEARCH.md).

## Contexto

O AgentRouter aplica um **WAF de fingerprint de conexão**. Clientes genéricos (Node `fetch`/undici,
`curl`, `httpx` assíncrono e o `net/http` do Go) recebem
`401 unauthorized client detected, contact support...` (`type: unauthorized_client_error`) **mesmo
com chave válida**. O único transporte observado que passa é o **SDK Python `anthropic` síncrono**.

O AllCode resolve isso com um relay local (`relay.py`, MIT) que recebe Anthropic Messages e reemite
pelo SDK síncrono. O PoC (chave real, porta isolada) confirmou: o relay autentica e retorna chat
(`gpt-5.6-sol` → 200 `text="ok"`, in=80/out=4); o adapter Go nativo recebe 401 de WAF. O console
mostra que a disponibilidade é dinâmica (`gpt-6.1sol`, `gpt-6-astra`, `claude-opus-5.5`) e que
`402 Budget pool quota exhausted` / `503 no available channel` são estados da conta, não do transporte.

## Decisão

- **Vendorizar** o relay do AllCode (`relay.py` + `manager.ts` como referência) em
  `vendor/agentrouter-relay/`, preservando a licença **MIT** (copyright opencode) e o `NOTICE`.
- **Sem instalação de dependências em runtime:** Python 3.12 + `anthropic` (versão pinada) entram na
  imagem Docker no build. Nada de `pip install`/download de Python em produção.
- **Sidecar interno no Docker Swarm** (`deploy/stack.yml`): rede `app`, **sem porta publicada**, sem
  túnel, `healthcheck /health`, `replicas: 1`. O app AllCode **não** entra no caminho; só o código
  vendorizado.
- **Chave via `x-api-key` repassada pelo gateway:** o gateway mantém a credencial cifrada no Postgres,
  aplica `tenant_id`, autenticação e políticas do Endpoint Registry, e envia o header ao relay. O relay
  **não** guarda chave em env/segredo e **recusa** chamadas sem `x-api-key` (401). Payload e chave
  nunca são logados.
- **Tradução fail-closed no gateway:** o relay vendorizado **não** traduz (removida a tradução
  best-effort + instrução pt-BR). O adapter `agentrouter` do gateway traduz para inglês com falha
  fechada e localiza a resposta.
- **Catálogo estático** atualizado com os modelos do console (`gpt-6.1sol`, `gpt-6-astra`,
  `claude-opus-5.5`); **disponibilidade é dinâmica** (402/503), nunca hardcoded.
- **Erros:** `402` → `ErrProviderQuota`; `503` → `ErrProviderUnavailable`; `401/403` →
  `ErrProviderAuth`; nunca tratados como falha de proxy.
- **Escopo restrito ao AgentRouter.** Nenhum outro provider usa relay; 9Router/OmniRoute/AllCode
  continuam proibidos como canal/upstream em runtime.

## Alternativas rejeitadas

- **Cliente Go nativo com fingerprint TLS/HTTP2 (uTLS):** evitaria Python, mas replica um
  fingerprint não documentado e frágil; sem garantia de continuidade.
- **Não integrar (provider indisponível):** perde um provedor já contratado.
- **Usar o app AllCode como relay:** proibido (dependência de gateway de referência em runtime).
- **Tradução best-effort no relay:** contraria a decisão de falha fechada do dontstop.

## Consequências

- (+) AgentRouter funcional no dontstop sem depender do app AllCode; chave no cofre do gateway.
- (+) Isolamento de rede (sidecar interno), sem exposição pública.
- (?) Runtime Python + imagem adicional; WAF pode mudar e exigir atualização do SDK pinado.
- (?) Disponibilidade limitada por canal/budget da conta; exige observabilidade de 402/503.
- (?) Manutenção do fork (limpeza de payload/SSE) e atribuição MIT.

## Documentos correspondentes

[`05-ADRS.md`](../05-ADRS.md) · [`AGENTROUTER-ALLCODE-RESEARCH.md`](../reference/AGENTROUTER-ALLCODE-RESEARCH.md) ·
[`architecture/ADD.md`](../architecture/ADD.md) · [`deploy/README.md`](../../deploy/README.md) ·
[`qa/agentrouter-relay-poc.md`](../qa/agentrouter-relay-poc.md).
