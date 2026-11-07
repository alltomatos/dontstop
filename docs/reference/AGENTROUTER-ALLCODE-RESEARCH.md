# AgentRouter — investigação no AllCode

> Fonte de verdade consultada: `D:\dev\allcode`, em 2026-11-07.
> Objetivo: registrar o comportamento real do AgentRouter antes da integração no dontstop.
> Nada neste documento foi inferido de documentação pública; os pontos abaixo vêm do código do AllCode.

## Arquivos de referência

| Arquivo | Conteúdo |
|---|---|
| `packages/allcode/src/provider/provider.ts` | registro do provider, lista estática de modelos, relay e blocklist |
| `packages/allcode/src/provider/agentrouter/manager.ts` | ciclo de vida do relay, porta, health, recuperação de processo órfão |
| `packages/allcode/src/provider/agentrouter/relay.py` | relay HTTP local, protocolo Anthropic, limpeza de payload, tradução e SSE |
| `packages/allcode/src/provider/agentrouter/relay-source.ts` | cópia embutida do relay para o app empacotado |
| `packages/allcode/test/provider/agentrouter-manager.test.ts` | testes de porta/health/versão do relay |
| `packages/allcode/src/session/prompt-enhance.ts` | regra de idioma do prompt enhancer |

## Transporte e autenticação

- O AgentRouter expõe **Anthropic Messages**, não OpenAI Chat Completions.
- Endpoint usado: `POST /v1/messages`.
- Autenticação: `x-api-key` + `anthropic-version: 2023-06-01` (via SDK `anthropic`).
- Upstream padrão: `https://agentrouter.org`.
- O AgentRouter aplica um **WAF de fingerprint de conexão**. O AllCode registra que clientes genéricos (Node `fetch`/undici, `curl`, `httpx` assíncrono) recebem `unauthorized client detected` mesmo com chave válida.
- Solução do AllCode: relay local em Python usando o SDK **síncrono** `anthropic`. Porta padrão `7187`; health em `GET /health` retorna `{"ok":true,"version":2}`. O relay reemite as requisições e esconde o WAF do processo principal.
- Consequência para o dontstop: o adaptador fala Anthropic Messages e deve apontar `base_url` para um relay compatível. Não é seguro assumir que `net/http` direto passará no WAF.

## Modelos e limites

Lista estática do AllCode (o `/v1/models` real também é bloqueado pelo WAF):

| Modelo | Observação |
|---|---|
| `claude-opus-4-8` | disponível; não é escolhido automaticamente pelo AllCode |
| `claude-opus-5` | disponível; não é escolhido automaticamente pelo AllCode |
| `deepseek-v4-flash` | disponível |
| `glm-5.3` | disponível |
| `gpt-5.6-sol` | disponível; não é escolhido automaticamente pelo AllCode |

- Limites declarados no AllCode são placeholders: contexto `200_000`, saída `8_192`, custo zero.
- AgentRouter é **pré-pago por créditos**; não há preço por token publicado nem endpoint dinâmico de modelos.
- Capabilities declaradas: texto e tool calls; sem imagem/áudio/vídeo/PDF.
- O AllCode remove Opus/GPT da escolha automática por instabilidade (loops, lentidão, rate limit/filtro), mas mantém seleção manual.

## Payload e filtro de conteúdo

- O filtro do AgentRouter rejeita texto não inglês, especialmente português, com `400 content-blocked`.
- O AllCode remove campos que disparam o filtro:
  - topo: `thinking`, `output_config`;
  - blocos de mensagens: `thinking`, `redacted_thinking`;
  - `cache_control` dos blocos antes do relay.
- Em streaming, o AllCode descarta eventos `billing_summary` e serializa eventos Anthropic com `exclude_none=True` para evitar campos `null` que quebram parsers Zod.
- O relay usa `client.messages.create(stream=True)` de baixo nível, não `messages.stream()`, para não sintetizar eventos extras.

## Tradução no AllCode

- O relay traduz **todos os blocos de texto** (mensagens do usuário, histórico do assistente e system prompt que pareça português) para inglês antes de sair.
- Tradução usa Google `translate_a/single` (`client=gtx`, `sl=auto`, `tl=en`) com fallback MyMemory; cache em memória e paralelismo (`ThreadPoolExecutor(8)`) para reduzir latência.
- O AllCode acrescenta uma instrução no system prompt pedindo resposta em pt-BR e faz fallback **best-effort** para o texto original quando a tradução falha.
- **Decisão do dontstop:** não copiar o fallback silencioso. O requisito do projeto é falha fechada: se a tradução do prompt falhar, não chamar o AgentRouter e registrar o motivo. A resposta será traduzida de volta ao idioma original (confirmado pelo solicitante).

## Streaming

- O AgentRouter fala SSE Anthropic (`message_start`, `content_block_delta`, `message_delta`, `message_stop`).
- O relay remove `billing_summary` e normaliza campos nulos.
- O adaptador do dontstop acumula a resposta em inglês e traduz o texto completo antes de emitir o primeiro chunk ao cliente. Isso preserva a qualidade da tradução, ao custo de latência; o comportamento deve ser documentado no runbook.
- Falha na tradução da resposta antes do primeiro chunk retorna erro explícito; não há emissão parcial.

## Decisão de integração no dontstop

- Novo kind de provider: `agentrouter`.
- Protocolo: Anthropic Messages.
- `Account.BaseURL`: deve apontar para o relay (ex.: `http://127.0.0.1:7187`) ou para um endpoint compatível que passe no WAF.
- Tradução: `Translator` injetável; Google primário, MyMemory fallback; `ErrTranslation` é falha fechada.
- Prompt original: nunca é mutado; permanece na requisição do serviço e é usado para auditoria/traço. Logs registram apenas estágio, idioma de origem e motivo, sem o texto.
- Resposta: traduzida de volta ao idioma detectado no último prompt de usuário; se o idioma for inglês, não há tradução reversa.
- Modelos: catálogo estático deve ser mantido no banco/seed; não chamar `/v1/models` do AgentRouter.

## Riscos e pendências

- O relay Python é uma dependência operacional do AllCode; o dontstop deve preferir um relay versionado ou um transporte que passe no WAF, nunca instalar SDK em runtime no container do gateway.
- A tradução adiciona latência e depende de serviço externo; falhas devem ser monitoradas e bloqueadas, não mascaradas.
- O limite real de contexto/saída e o custo por modelo não são publicados; usar os placeholders do catálogo até termos medição própria.
- Testes de contrato/adaptador usam `httptest`; o teste live é opt-in (`AGENTROUTER_LIVE_KEY`, `AGENTROUTER_LIVE_BASE_URL`).
