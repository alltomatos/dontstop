# QA — AgentRouter (P0-α)

> Data: 2026-11-07. Referência de implementação: `D:\dev\allcode`.

## Escopo

- Adaptador `agentrouter` no gateway Go.
- Tradução de prompt para inglês antes do envio.
- Tradução da resposta de volta ao idioma detectado no prompt.
- Falha fechada quando qualquer tradução falha.
- Preservação do prompt original na requisição do serviço.
- Testes unitários/contrato com `httptest` e teste live opt-in.

## Casos executados

| ID | Caso | Resultado |
|---|---|---|
| QA-AR-01 | Prompt e system traduzidos antes do upstream | PASS (`TestAgentRouterTranslatesPromptBeforeSendingAndResponseBack`) |
| QA-AR-02 | Prompt original não é mutado | PASS |
| QA-AR-03 | Falha de tradução bloqueia chamada upstream | PASS (`TestAgentRouterBlocksRequestWhenPromptTranslationFails`) |
| QA-AR-04 | Resposta não streaming traduzida de volta | PASS |
| QA-AR-05 | Streaming acumula, traduz e emite chunk final | PASS (`TestAgentRouterStreamTranslatesBufferedResponse`) |
| QA-AR-06 | Falha de tradução no streaming não emite conteúdo parcial | PASS |
| QA-AR-07 | Prompt em inglês não faz tradução reversa | PASS |
| QA-AR-08 | Google translator + fallback MyMemory | PASS |
| QA-AR-09 | Ambos tradutores indisponíveis retornam `ErrTranslation` | PASS |
| QA-AR-10 | Live opt-in via relay AllCode | PASS com `gpt-5.6-sol`; resposta `Olá! Como posso ajudar?` |

## Evidência live

- Relay local AllCode (`relay.py`, porta 7187, `RELAY_VERSION=2`) usando a chave de teste fornecida fora do repositório.
- Modelo `gpt-5.6-sol` foi o único com canal disponível para a chave testada; `claude-opus-5`, `claude-opus-4-8`, `deepseek-v4-flash` e `glm-5.3` retornaram “sem canal disponível”.
- Um prompt contendo “Just respond OK.” foi bloqueado pelo filtro (`content-blocked`); “Hello” passou. Isso confirma que o filtro do AgentRouter é sensível ao conteúdo mesmo em inglês.
- A chave não é versionada; o teste live usa `AGENTROUTER_LIVE_KEY`, `AGENTROUTER_LIVE_BASE_URL` e `AGENTROUTER_LIVE_MODEL`.

## Riscos residuais

- O relay Python do AllCode não é empacotado no gateway; é uma dependência operacional para produção até existir transporte que passe no WAF.
- A tradução reversa de streaming é bufferizada: o primeiro chunk só chega após a resposta completa, o que aumenta o TTFT.
- O catálogo de modelos e os limites reais precisam ser confirmados por modelo/conta.
- Falhas de tradução retornam 502 `translation_error`; monitorar taxa e latência.
