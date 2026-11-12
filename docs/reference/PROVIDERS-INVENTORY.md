# Anexo A — Inventário consolidado de provedores

> **Documento gerado por extração automatizada das fontes** (ver `PROVIDERS-MAPPING.md`, seção 0).
> Não editar manualmente: alterações devem ser feitas nos scripts descritos na seção "Como reproduzir".
> Data da extração: 2026-11-10.
>
> Fontes: OmniRoute `v3.8.53` (`191dce8d53`), 9Router `v0.5.112` (`27b05be9`), AllCode `1.27.6` (`687377a2c`, branch `dev`).

## Legenda

- **Origem** — `O` = OmniRoute, `9` = 9Router, `A` = AllCode.
- **Auth** — valor declarado no registry de origem (`apikey`, `oauth`, `optional`, `none`, `freeTier`, `key`, `AWS creds`).
- **Protocolo** — formato de wire declarado. `OpenAI-compatible` cobre chat/completions; provedores com executor dedicado aparecem com o protocolo real.
- **Base URL** — URL declarada no registry (sem path de modelo). `-` = montada em runtime pelo executor ou não declarada.
- **Modelos** — maior contagem estática entre as fontes; `(ctx Nk)` = maior context window explicitamente declarado (o default de 200k do 9Router foi descartado). `-` = catálogo 100% dinâmico.
- **Grupo** — classificação usada no mapeamento: `1P` (adapter/executor dedicado), `openai-compat` (compatível OpenAI sem adapter), `gateway` (agregador/roteador), `web` (API web/cookie/CLI reversa — fora do P0), `midia` (imagem/TTS/STT/embedding/search), `local` (self-hosted).

## Tabela

| Provedor (slug canônico) | Origem | Auth | Protocolo | Base URL | Modelos | Grupo |
|---|---|---|---|---|---|---|
| adapta-web | O | apikey | OpenAI-compatible | https://agent.adapta.one/api/chat/stream/v1 | 7 | web |
| agentrouter | O | apikey | Anthropic Messages | https://agentrouter.org/v1/messages | 3 | 1P |
| agnes | 9+O | apikey | OpenAI-compatible | https://apihub.agnes-ai.com/v1/chat/completions | 4 (ctx 512k) | openai-compat |
| agnes-cn | O | apikey | OpenAI-compatible | https://api.agnes-ai.cn/v1/chat/completions | 3 (ctx 512k) | openai-compat |
| agy | O | oauth | Code Assist (prop.) | - | 19 (ctx 1024k) | 1P |
| ai21 | O | apikey | OpenAI-compatible | https://api.ai21.com/studio/v1/chat/completions | 2 | openai-compat |
| aihorde | O | apikey | OpenAI-compatible | https://oai.aihorde.net/v1/chat/completions | 3 (ctx 32k) | openai-compat |
| aimlapi | O | apikey | OpenAI-compatible | https://api.aimlapi.com/v1/chat/completions | 6 | openai-compat |
| ainative | O | apikey | OpenAI-compatible | https://api.ainative.studio/api/v1/chat/completions | 9 (ctx 128k) | openai-compat |
| ainetcafe | O | apikey | OpenAI-compatible | https://microquickjs.com/v1/chat/completions | - | openai-compat |
| aion | O | apikey | OpenAI-compatible | https://api.aionlabs.ai/v1/chat/completions | 5 (ctx 128k) | openai-compat |
| alibaba | 9+A+O | apikey/key | OpenAI-compatible / OpenAI-compatible (SDK) | https://dashscope-intl.aliyuncs.com/compatible-mode/v1/chat/completions | 14 (ctx 977k) | openai-compat |
| alibaba-cn | O | apikey | OpenAI-compatible | https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions | 14 | openai-compat |
| alibaba-coding | 9 | apikey | OpenAI-compatible | https://coding.dashscope.aliyuncs.com/v1/chat/completions | 8 (ctx 977k) | openai-compat |
| alibaba-coding-intl | 9 | apikey | OpenAI-compatible | https://coding-intl.dashscope.aliyuncs.com/v1/chat/completions | 7 (ctx 977k) | openai-compat |
| ant-ling | O | apikey | OpenAI-compatible | https://api.ant-ling.com/v1/chat/completions | 3 | openai-compat |
| anthropic | 9+A+O | apikey/key | Anthropic Messages / Anthropic Messages | https://api.anthropic.com/v1/messages / https://api.anthropic.com | 14 (ctx 1024k) | 1P |
| antigravity | 9+A+O | oauth | Code Assist (prop.) / Code Assist (prop.) | https://daily-cloudcode-pa.googleapis.com | 28 (ctx 1024k) | 1P |
| anyapi | O | apikey | OpenAI-compatible | https://api.anyapi.ai/v1/chat/completions | - | openai-compat |
| api-airforce | 9+O | apikey | OpenAI-compatible | https://api.airforce/v1/chat/completions | 7 (ctx 1024k) | openai-compat |
| apmix | O | apikey | OpenAI-compatible | https://api.apmix.ai/v1/chat/completions | 41 (ctx 1024k) | gateway |
| arcee-ai | O | apikey | OpenAI-compatible | https://api.arcee.ai/api/v1/chat/completions | - | openai-compat |
| assemblyai | 9 | apikey | OpenAI-compatible | https://api.assemblyai.com/v1/audio/transcriptions | 4 | openai-compat |
| atria | 9 | apikey | OpenAI-compatible | https://api.atria-asi.ai/v1/chat/completions | 1 | openai-compat |
| auggie | O | none | OpenAI-compatible | auggie://cli/stdio | 28 (ctx 977k) | web |
| auriko | O | apikey | OpenAI-compatible | https://api.auriko.ai/v1/chat/completions | - | openai-compat |
| aws-polly | 9 |  |  | - | - | midia |
| azure | 9+A | apikey/key + resourceName | OpenAI-compatible / Azure OpenAI | https://{resource}.openai.azure.com/openai/v1 / https://{resource}.cognitiveservices.azure.com/openai | - | openai-compat |
| bai | 9+O | apikey | OpenAI-compatible | https://api.b.ai/v1/chat/completions | - | openai-compat |
| baichuan | O | apikey | OpenAI-compatible | https://api.baichuan-ai.com/v1/chat/completions | 5 (ctx 128k) | openai-compat |
| baidu | 9+O | apikey | OpenAI-compatible | https://qianfan.baidubce.com/v2/chat/completions | 16 (ctx 977k) | openai-compat |
| bailian-coding-plan | O | apikey | Anthropic Messages | https://token-plan.ap-southeast-1.maas.aliyuncs.com/apps/anthropic/v1 | 6 (ctx 977k) | 1P |
| baseten | O | apikey | OpenAI-compatible | https://inference.baseten.co/v1/chat/completions | 16 (ctx 1024k) | openai-compat |
| bazaarlink | 9+O | apikey | OpenAI-compatible | https://bazaarlink.ai/api/v1/chat/completions | 33 (ctx 1025k) | gateway |
| beatapi | O | apikey | OpenAI-compatible | https://api.beatapi.io/v1/chat/completions | - | openai-compat |
| bedrock | 9+A+O | apikey/AWS creds | OpenAI-compatible / Anthropic Messages / Bedrock Converse | https://bedrock-runtime.{region}.amazonaws.com | 11 (ctx 977k) | 1P |
| bedrock-xai | 9 | apikey | OpenAI-compatible | https://bedrock-runtime.{region}.amazonaws.com | 2 (ctx 488k) | openai-compat |
| bigmodel | O | apikey | OpenAI-compatible | https://open.bigmodel.cn/api/paas/v4/chat/completions | 15 | openai-compat |
| black-forest-labs | 9 |  |  | - | - | midia |
| blackbox | 9+O | apikey | OpenAI-compatible | https://api.blackbox.ai/v1/chat/completions | 11 (ctx 1025k) | openai-compat |
| blackbox-web | O | apikey | OpenAI-compatible | https://app.blackbox.ai/api/chat | 6 | web |
| bluesminds | 9+O | apikey | OpenAI-compatible | https://api.bluesminds.com/v1/chat/completions | 22 (ctx 1024k) | openai-compat |
| brave-search | 9 |  |  | - | - | midia |
| byteplus | 9+O | apikey/freeTier | OpenAI-compatible | https://ark.ap-southeast.bytepluses.com/api/v3/chat/completions / https://ark.ap-southeast.bytepluses.com/api/coding/v3/chat/completions | 7 (ctx 256k) | openai-compat |
| bytez | O | apikey | OpenAI-compatible | https://api.bytez.com/models/v2/openai/v1/chat/completions | 3 | openai-compat |
| cartesia | 9 |  |  | - | - | midia |
| cerebras | 9+A+O | apikey/key | OpenAI-compatible / OpenAI-compatible | https://api.cerebras.ai/v1/chat/completions / https://api.cerebras.ai/v1 | 6 (ctx 977k) | openai-compat |
| charm-hyper | O | apikey | OpenAI-compatible | https://hyper.charm.land/v1/chat/completions | 1 | gateway |
| chat-oripe | O | apikey | OpenAI-compatible | https://api.oriper.com/v1/chat/completions | - | web |
| chatanywhere | O | apikey | OpenAI-compatible | https://api.chatanywhere.org/v1/chat/completions | - | openai-compat |
| chatgpt-web | 9+O | apikey/cookie | OpenAI-compatible | https://chatgpt.com | 11 (ctx 1025k) | web |
| chatgpt-web-codex | O | apikey | OpenAI Responses | https://chatgpt.com | 7 | web |
| chatplayground | O | apikey | OpenAI-compatible | https://app.chatplayground.ai/api/chat | - | web |
| cheaperinference | O | apikey | OpenAI-compatible | https://api.cheaperinference.com/v1/chat/completions | 39 | 1P |
| chenzk | O | apikey | OpenAI-compatible | https://chenzk.top/v1/chat/completions | - | openai-compat |
| chutes | 9+O | apikey | OpenAI-compatible | https://llm.chutes.ai/v1/chat/completions | 1 | openai-compat |
| claude | 9+O | oauth | Anthropic Messages | https://api.anthropic.com/v1/messages | 14 (ctx 977k) | 1P |
| claude-web | O | apikey | OpenAI-compatible | https://claude.ai/api/organizations | 10 (ctx 977k) | web |
| cline | 9+O | oauth | OpenAI-compatible | https://api.cline.bot/api/v1/chat/completions | 17 (ctx 1025k) | openai-compat |
| clinepass | 9+O | oauth | OpenAI-compatible | https://api.cline.bot/api/v1/chat/completions | 11 (ctx 1024k) | openai-compat |
| cloudcode-one | O | apikey | OpenAI-compatible | https://api.cloudcode.one/v1/chat/completions | 2 | openai-compat |
| cloudflare-ai | 9+O | apikey | OpenAI-compatible | https://api.cloudflare.com/client/v4/accounts / https://api.cloudflare.com/client/v4/accounts/{accountId}/ai/v1/chat/completions | 25 (ctx 977k) | 1P |
| cloudflare-ai-gateway | A | key + accountId + gatewayId | OpenAI-compatible | https://gateway.ai.cloudflare.com/v1/{accountId}/{gatewayId}/compat | - | openai-compat |
| cloudflare-playground | O | none | OpenAI-compatible | https://playground.ai.cloudflare.com | 20 | web |
| cloudflare-workers-ai | A | key + accountId | Workers AI | https://api.cloudflare.com/client/v4/accounts/{accountId}/ai/v1 | - | openai-compat |
| clova-studio | O | apikey | Naver Clova (prop.) | https://clovastudio.stream.ntruss.com/v3/chat-completions | 3 (ctx 125k) | 1P |
| codebuddy-cn | 9+O | oauth | OpenAI-compatible | https://copilot.tencent.com/v2/chat/completions | 30 (ctx 977k) | 1P |
| codebuddy-intl | 9+O | oauth | OpenAI-compatible | https://www.codebuddy.ai/v2/chat/completions | 22 (ctx 977k) | 1P |
| codestral | O | apikey | OpenAI-compatible | https://codestral.mistral.ai/v1/chat/completions | 2 | openai-compat |
| codex | 9+O | oauth | OpenAI Responses | https://chatgpt.com/backend-api/codex/responses | 52 (ctx 852k) | 1P |
| codex-app-server | O | none | OpenAI Responses | codex-app-server://cli/websocket | 52 (ctx 852k) | web |
| cohere | 9+A+O | apikey/key | OpenAI-compatible / Cohere | https://api.cohere.com/compatibility/v1/chat/completions / https://api.cohere.ai/v1/chat/completions | 6 (ctx 125k) | openai-compat |
| comfyui | 9 |  |  | - | - | midia |
| commandcode | 9+O | apikey | OpenAI-compatible / CommandCode (prop.) | https://api.commandcode.ai / https://api.commandcode.ai/alpha/generate | 23 (ctx 1024k) | 1P |
| conol-web | O | apikey | OpenAI-compatible | https://conol.ai/api/sessions | 28 | web |
| copilot-m365-web | O | apikey | OpenAI-compatible | wss://substrate.office.com/m365Copilot/Chathub | 4 | web |
| copilot-web | O | apikey | OpenAI-compatible | wss://copilot.microsoft.com/c/api/chat?api-version=2 | 3 | web |
| coqui | 9 |  |  | - | - | midia |
| coze | O | apikey | OpenAI-compatible | https://api.coze.com/v1/chat/completions | 1 | openai-compat |
| crof | O | apikey | OpenAI-compatible | https://crof.ai/v1/chat/completions | 15 | openai-compat |
| cursor | 9+O | oauth | Cursor Connect (prop.) | https://api2.cursor.sh | 224 (ctx 1024k) | web |
| cursor-api | O | apikey | Cursor Connect (prop.) | https://api2.cursor.sh | 224 (ctx 977k) | web |
| dahl | 9+O | apikey | OpenAI-compatible | https://inference.dahl.global/v1/chat/completions | 3 (ctx 977k) | 1P |
| daoxe | O | apikey | OpenAI-compatible | https://api.daoxe.com/v1/chat/completions | 9 | openai-compat |
| databricks | O | apikey | OpenAI-compatible | https://adb-0000000000000000.0.azuredatabricks.net/serving-endpoints | 4 | openai-compat |
| deepai | O | apikey | Proprietario | https://api.deepai.org | 1 | 1P |
| deepgram | 9 | apikey | OpenAI-compatible | https://api.deepgram.com/v1/listen | 4 | openai-compat |
| deepinfra | A+O | apikey/key | OpenAI-compatible / OpenAI-compatible | https://api.deepinfra.com/v1/openai/chat/completions / https://api.deepinfra.com/v1/openai | 22 | openai-compat |
| deepseek | 9+O | apikey | OpenAI-compatible | https://api.deepseek.com/chat/completions | 8 (ctx 977k) | openai-compat |
| deepseek-web | 9+O | apikey/cookie | OpenAI-compatible | https://chat.deepseek.com/api/v0/chat/completion | 14 (ctx 977k) | web |
| demonroute | O | apikey | OpenAI-compatible | https://api.demonroute.com/v1/chat/completions | 7 | openai-compat |
| devin-cli | O | oauth | OpenAI-compatible | devin://acp/stdio | 110 (ctx 1024k) | web |
| devin-cli-agentic | O | none | Anthropic Messages | devin://acp/stdio | 110 (ctx 1024k) | web |
| devin-desktop | O | oauth | OpenAI-compatible | https://server.codeium.com | 110 (ctx 1024k) | web |
| dgrid | O | apikey | OpenAI-compatible | https://api.dgrid.ai/v1/chat/completions | 1 | openai-compat |
| dify | O | apikey | OpenAI-compatible | https://api.dify.ai | 1 | openai-compat |
| digitalocean | O | apikey | OpenAI-compatible | https://inference.do-ai.run/v1/chat/completions | - | openai-compat |
| dit | O | apikey | OpenAI-compatible | https://api.dit.ai/v1/chat/completions | 2 (ctx 391k) | openai-compat |
| doubao | O | apikey | OpenAI-compatible | https://ark.cn-beijing.volces.com/api/v3/chat/completions | 9 (ctx 256k) | openai-compat |
| doubao-web | O | apikey | OpenAI-compatible | https://www.dola.com/chat/completion | 2 | web |
| duckduckgo-web | O | none | OpenAI-compatible | https://duck.ai/duckchat/v1/chat | 6 | web |
| dxnt | O | apikey | OpenAI-compatible | https://www.dxnt.com/v1/chat/completions | - | openai-compat |
| dynamic-provider | A | key | SDK dinamico | operator-defined | - | openai-compat |
| edge-tts | 9 |  |  | - | - | midia |
| electronhub | O | apikey | OpenAI-compatible | https://api.electronhub.ai/v1/chat/completions | - | openai-compat |
| elevenlabs | 9 |  |  | - | - | midia |
| eurouter | O | apikey | OpenAI-compatible | https://api.eurouter.ai/v1/chat/completions | - | gateway |
| exa | 9 |  |  | - | - | midia |
| factory | O | apikey | OpenAI-compatible | https://api.factory.ai/v1/chat/completions | 1 | openai-compat |
| fal-ai | 9 |  |  | - | - | midia |
| fastrouter | O | apikey | OpenAI-compatible | https://api.fastrouter.ai/api/v1/chat/completions | - | gateway |
| featherless | 9+O | apikey | OpenAI-compatible | https://api.featherless.ai/v1/chat/completions | 7 (ctx 977k) | openai-compat |
| firecrawl | 9 |  |  | - | - | midia |
| fireworks | 9+O | apikey | OpenAI-compatible | https://api.fireworks.ai/inference/v1/chat/completions | 10 (ctx 256k) | openai-compat |
| fish-audio | 9 |  |  | - | - | midia |
| free-ai | O | apikey | OpenAI-compatible | https://api.free.ai/v1/chat/ | - | openai-compat |
| freeaiapikey | O | apikey | OpenAI-compatible | https://api.freeaiapikey.com/v1/chat/completions | 10 (ctx 977k) | openai-compat |
| freebuff | O | apikey | OpenAI-compatible | https://www.codebuff.com/api/v1 | 9 (ctx 128k) | web |
| freeinference | O | apikey | OpenAI-compatible | https://freeinference.org/v1/chat/completions | - | openai-compat |
| freemodel-dev | O | apikey | OpenAI-compatible | https://api.freemodel.dev/v1/chat/completions | 4 (ctx 391k) | openai-compat |
| freetheai | O | apikey | OpenAI-compatible | https://api.freetheai.org/v1/chat/completions | 2 | openai-compat |
| friendliai | O | apikey | OpenAI-compatible | https://api.friendli.ai/serverless/v1/chat/completions | 2 | openai-compat |
| g4f-gemini | O | optional | OpenAI-compatible | https://g4f.space/api/gemini/v1/chat/completions | 2 | openai-compat |
| g4f-groq | O | optional | OpenAI-compatible | https://g4f.space/api/groq/v1/chat/completions | 2 | openai-compat |
| g4f-nvidia | O | optional | OpenAI-compatible | https://g4f.space/api/nvidia/v1/chat/completions | 3 | openai-compat |
| g4f-ollama | O | optional | OpenAI-compatible | https://g4f.space/api/ollama/v1/chat/completions | 1 | openai-compat |
| g4f-pollinations | O | optional | OpenAI-compatible | https://g4f.space/api/pollinations/v1/chat/completions | 2 | openai-compat |
| galadriel | O | apikey | OpenAI-compatible | https://api.galadriel.ai/v1/chat/completions | 1 | openai-compat |
| gateway | A | key | OpenAI-compatible | - (SDK gateway) | - | gateway |
| gemini | 9+A+O | apikey/key | Gemini generateContent / Gemini | https://generativelanguage.googleapis.com/v1beta/models / https://generativelanguage.googleapis.com | 28 (ctx 1024k) | 1P |
| gemini-cli | 9 | free | Code Assist (prop.) | https://cloudcode-pa.googleapis.com/v1internal | 7 (ctx 1024k) | 1P |
| gemini-web | 9+O | apikey/cookie | OpenAI-compatible | https://gemini.google.com/app | 4 (ctx 1024k) | web |
| ghe-copilot | O | oauth | OpenAI-compatible | https://api.githubcopilot.com/chat/completions | 32 (ctx 1025k) | 1P |
| gigachat | O | apikey | OpenAI-compatible | https://gigachat.devices.sberbank.ru/api/v1 | 3 | openai-compat |
| gisoo | O | apikey | OpenAI-compatible | https://gisoo.pro/api/v1/chat/completions | 9 | openai-compat |
| github | 9+A+O | oauth | OpenAI-compatible / OpenAI-compatible | https://api.githubcopilot.com/chat/completions / https://api.githubcopilot.com | 30 (ctx 1025k) | 1P |
| gitlab | 9+A | oauth/key/oauth | OpenAI-compatible / GitLab AI Provider | https://gitlab.com/api/v4/chat/completions / https://{instance}/api/v4 | - | openai-compat |
| gitlab-duo | O | oauth | OpenAI-compatible | https://gitlab.com/api/v4/code_suggestions/completions | 2 | 1P |
| gitlawb | O | apikey | OpenAI-compatible | https://opengateway.gitlawb.com/v1/xiaomi-mimo | 5 (ctx 1024k) | web |
| gitlawb-gmi | O | apikey | OpenAI-compatible | https://opengateway.gitlawb.com/v1/gmi-cloud | 40 (ctx 1025k) | web |
| glm | 9+O | apikey/oauth | OpenAI-compatible / Anthropic Messages | https://api.z.ai/api/coding/paas/v4/chat/completions / https://api.z.ai/api/anthropic/v1/messages | 21 (ctx 977k) | 1P |
| glm-cn | 9+O | apikey | OpenAI-compatible | https://open.bigmodel.cn/api/coding/paas/v4/chat/completions | 21 (ctx 977k) | 1P |
| glmt | O | apikey | OpenAI-compatible | https://api.z.ai/api/coding/paas/v4/chat/completions | 21 (ctx 977k) | 1P |
| google-pse | 9 |  |  | - | - | midia |
| google-tts | 9 |  |  | - | - | midia |
| greenpt | O | apikey | OpenAI-compatible | https://api.greenpt.ai/v1/chat/completions | - | openai-compat |
| grok-cli | 9+O | oauth | OpenAI-compatible / OpenAI Responses | https://cli-chat-proxy.grok.com/v1/chat/completions / https://cli-chat-proxy.grok.com/v1/responses | 6 (ctx 488k) | 1P |
| grok-web | 9+O | apikey/cookie | OpenAI-compatible / Grok Web (prop.) | https://grok.com/rest/app-chat/conversations/new | 14 (ctx 250k) | web |
| groq | 9+A+O | apikey/key | OpenAI-compatible / OpenAI-compatible | https://api.groq.com/openai/v1/chat/completions / https://api.groq.com/openai/v1 | 7 (ctx 977k) | openai-compat |
| haiper | O | apikey | OpenAI-compatible | https://api.haiper.ai/v1 | 2 | openai-compat |
| hcnsec | O | apikey | OpenAI-compatible | https://api.hcnsec.cn/v1/chat/completions | - | openai-compat |
| helixmind | O | apikey | OpenAI-compatible | https://helixmind.online/v1/chat/completions | - | openai-compat |
| helyxai | O | apikey | OpenAI-compatible | https://helyxai.space/v1/chat/completions | - | openai-compat |
| heroku | O | apikey | OpenAI-compatible | https://us.inference.heroku.com/v1/chat/completions | 12 | openai-compat |
| huggingchat | O | apikey | OpenAI-compatible | https://huggingface.co/chat/conversation | 24 | web |
| huggingface | 9+O | apikey | OpenAI-compatible | https://router.huggingface.co/v1/chat/completions | 6 | midia |
| hyperagent | O | apikey | OpenAI-compatible | https://hyperagent.com/api/threads | 6 (ctx 977k) | web |
| hyperbolic | 9+O | apikey | OpenAI-compatible | https://api.hyperbolic.xyz/v1/chat/completions | 8 (ctx 977k) | openai-compat |
| ideogram | O | apikey | OpenAI-compatible | https://api.ideogram.ai | 2 | openai-compat |
| iflow | 9 | oauth | OpenAI-compatible | https://apis.iflow.cn/v1/chat/completions | 15 (ctx 977k) | openai-compat |
| iflytek | O | apikey | OpenAI-compatible | https://spark-api-open.xf-yun.com/v1/chat/completions | 6 (ctx 128k) | openai-compat |
| inception | O | apikey | OpenAI-compatible | https://api.inceptionlabs.ai/v1/chat/completions | 2 (ctx 254k) | openai-compat |
| inference-net | O | apikey | OpenAI-compatible | https://api.inference.net/v1/chat/completions | 3 | openai-compat |
| inner-ai | O | apikey | OpenAI-compatible | https://chatapi.innerai.com/chat | 19 | 1P |
| internlm | O | apikey | OpenAI-compatible | https://chat.intern-ai.org.cn/api/v1/chat/completions | 5 | openai-compat |
| inworld | 9 |  |  | - | - | midia |
| jina-ai | 9 |  |  | - | - | midia |
| jina-reader | 9 |  |  | - | - | midia |
| kenari | O | apikey | OpenAI-compatible | https://kenari.id/v1/chat/completions | - | openai-compat |
| kie | O | apikey | OpenAI-compatible | https://api.kie.ai/v1/chat/completions | 10 | openai-compat |
| kilo-gateway | 9+O | optional/apikey | OpenAI-compatible | https://api.kilo.ai/api/gateway/chat/completions | 6 (ctx 125k) | gateway |
| kilocode | 9+A+O | oauth | OpenAI-compatible / OpenAI-compatible | https://api.kilo.ai/api/openrouter/chat/completions / https://api.kilo.ai/api/gateway | 15 (ctx 1024k) | gateway |
| kimchi | 9 | freeTier | OpenAI-compatible | https://llm.kimchi.dev/openai/v1/chat/completions | 8 (ctx 977k) | openai-compat |
| kimi | 9+O | apikey/oauth | OpenAI-compatible / Anthropic Messages | https://api.moonshot.ai/v1/chat/completions / https://api.kimi.com/coding/v1/messages | 10 (ctx 1024k) | 1P |
| kimi-coding | O | oauth | Anthropic Messages | https://api.kimi.com/coding/v1/messages?beta=true | 5 (ctx 1024k) | 1P |
| kimi-coding-apikey | O | apikey | Anthropic Messages | https://api.kimi.com/coding/v1/messages?beta=true | 5 (ctx 1024k) | 1P |
| kimi-web | O | apikey | OpenAI-compatible | https://www.kimi.ai | 2 | web |
| kiro | 9+A+O | oauth/free | AWS EventStream (prop.) / AWS EventStream (prop.) | https://codewhisperer.us-east-1.amazonaws.com/generateAssistantResponse / https://runtime.us-east-1.kiro.dev/generateAssistantResponse | 48 (ctx 977k) | 1P |
| lambda-ai | O | apikey | OpenAI-compatible | https://api.lambda.ai/v1/chat/completions | 3 | openai-compat |
| leonardo | O | apikey | OpenAI-compatible | https://cloud.leonardo.ai/api/rest/v1 | 2 | openai-compat |
| linkup | 9 |  |  | - | - | midia |
| liquid | O | apikey | OpenAI-compatible | https://inference.liquid.ai/v1/chat/completions | 1 | openai-compat |
| literouter | O | apikey | OpenAI-compatible | https://api.literouter.com/v1/chat/completions | - | gateway |
| llamagate | O | apikey | OpenAI-compatible | https://llamagate.ai/v1/chat/completions | 3 | openai-compat |
| llm-kiwi | O | apikey | OpenAI-compatible | https://api.llm.kiwi/v1/chat/completions | 2 | openai-compat |
| llm7 | 9+O | apikey | OpenAI-compatible | https://api.llm7.io/v1/chat/completions | 5 (ctx 1025k) | gateway |
| llmgateway | A+O | apikey/key | OpenAI-compatible / OpenAI-compatible | https://api.llmgateway.io/v1/chat/completions / https://api.llmgateway.io/v1 | - | gateway |
| lmarena | O | apikey | OpenAI-compatible | https://arena.ai/nextjs-api/stream/create-evaluation | 48 | web |
| local-device | 9 |  |  | - | - | midia |
| logfare | O | apikey | OpenAI-compatible | https://logfare.ai/v1/chat/completions | - | openai-compat |
| longcat | O | apikey | OpenAI-compatible | https://api.longcat.chat/openai/v1/chat/completions | 1 (ctx 1024k) | openai-compat |
| lyceum | O | apikey | OpenAI-compatible | https://api.lyceum.technology/openai/v1/chat/completions | - | openai-compat |
| maritalk | O | apikey | OpenAI-compatible | https://chat.maritaca.ai/api | 3 | openai-compat |
| maxai | O | apikey | OpenAI-compatible | https://api.maxai.me | 13 (ctx 1953k) | web |
| meganova-ai | O | apikey | OpenAI-compatible | https://api.meganova.ai/v1/chat/completions | - | openai-compat |
| meta-llama | O | apikey | OpenAI-compatible | https://api.llama.com/compat/v1/chat/completions | 4 | openai-compat |
| minimax | 9+O | apikey | OpenAI-compatible / Anthropic Messages | https://api.minimax.io/v1/chat/completions / https://api.minimax.io/anthropic/v1/messages | 13 (ctx 1024k) | 1P |
| minimax-cn | 9+O | apikey | OpenAI-compatible / Anthropic Messages | https://api.minimaxi.com/v1/chat/completions / https://api.minimaxi.com/anthropic/v1/messages | 12 (ctx 1024k) | 1P |
| minimax-code | 9 | oauth | Anthropic Messages | https://agent.minimax.cn/mavis/api/v1/llm/v1/messages | 4 (ctx 500k) | 1P |
| minimax-code-global | 9 | oauth | Anthropic Messages | https://agent.minimax.io/mavis/api/v1/llm/v1/messages | 4 (ctx 500k) | 1P |
| mistral | 9+A+O | apikey/key | OpenAI-compatible / OpenAI-compatible | https://api.mistral.ai/v1/chat/completions / https://api.mistral.ai/v1 | 9 (ctx 1024k) | openai-compat |
| mixlayer | O | apikey | OpenAI-compatible | https://models.mixlayer.ai/v1/chat/completions | 1 | openai-compat |
| mlx-gemma | O | apikey | OpenAI-compatible | http://localhost:11435/v1 | 1 (ctx 8k) | local |
| mlx-qwen | O | apikey | OpenAI-compatible | http://localhost:11436/v1 | 1 (ctx 8k) | local |
| mnn-ai | O | apikey | OpenAI-compatible | https://api.mnnai.ru/v1/chat/completions | - | openai-compat |
| modal | O | apikey | OpenAI-compatible | https://api.modal.ai/v1/chat/completions | 1 | openai-compat |
| modelscope | O | apikey | OpenAI-compatible | https://api-inference.modelscope.cn/v1/chat/completions | - | openai-compat |
| monsterapi | O | apikey | OpenAI-compatible | https://api.monsterapi.ai/v1/chat/completions | 2 | gateway |
| morph | 9+O | apikey | OpenAI-compatible | https://api.morphllm.com/v1/chat/completions | 7 (ctx 1024k) | openai-compat |
| muse | 9+O | apikey/oauth | OpenAI-compatible | https://api.meta.ai/v1/responses / https://api.meta.ai/v1/chat/completions | 13 (ctx 1024k) | openai-compat |
| muse-spark-web | O | apikey | OpenAI-compatible | https://www.meta.ai/api/graphql | 3 | web |
| muse-web | 9 | cookie | OpenAI-compatible | https://muse.ai | 3 | web |
| nace | 9 |  |  | - | - | midia |
| naga-ac | O | optional | OpenAI-compatible | https://api.naga.ac/v1/chat/completions | - | openai-compat |
| naga-ai | O | apikey | OpenAI-compatible | https://api.naga.ac/v1/chat/completions | - | openai-compat |
| nan | O | apikey | OpenAI-compatible | https://api.nan.builders/v1/chat/completions | 8 | openai-compat |
| nanobanana | 9 | apikey | OpenAI-compatible | https://api.nanobananaapi.ai/v1/chat/completions | 2 | openai-compat |
| nanogpt | O | apikey | OpenAI-compatible | https://nano-gpt.com/api/v1/chat/completions | 3 | gateway |
| nara | O | apikey | OpenAI-compatible | https://router.bynara.id/v1/chat/completions | 8 (ctx 977k) | openai-compat |
| navy | O | apikey | OpenAI-compatible | https://api.navy/v1/chat/completions | 6 (ctx 9766k) | openai-compat |
| nebius | 9+O | apikey | OpenAI-compatible | https://api.tokenfactory.nebius.com/v1/chat/completions / https://api.studio.nebius.ai/v1/chat/completions | 2 (ctx 256k) | openai-compat |
| nlpcloud | O | apikey | OpenAI-compatible | https://api.nlpcloud.io/v1/gpu | 5 (ctx 16k) | openai-compat |
| notion-web | O | apikey | OpenAI-compatible | https://app.notion.com/api/v3/runInferenceTranscript | 37 | web |
| notrack-web | O | apikey | OpenAI-compatible | https://notrack.ai/api/dispatch | 1 | web |
| nous | 9 | oauth | OpenAI-compatible | https://inference-api.nousresearch.com/v1/chat/completions | 8 (ctx 1024k) | openai-compat |
| nous-research | O | apikey | OpenAI-compatible | https://inference-api.nousresearch.com/v1/chat/completions | 2 | openai-compat |
| novita | O | apikey | OpenAI-compatible | https://api.novita.ai/openai/v1/chat/completions | 19 (ctx 1024k) | openai-compat |
| nscale | O | apikey | OpenAI-compatible | https://inference.api.nscale.com/v1/chat/completions | 6 | openai-compat |
| nube | O | apikey | OpenAI-compatible | https://ai.nube.sh/api/v1/chat/completions | - | openai-compat |
| nvidia | 9+A+O | apikey/key | OpenAI-compatible / OpenAI-compatible | https://integrate.api.nvidia.com/v1/chat/completions / https://integrate.api.nvidia.com/v1 | 11 (ctx 977k) | openai-compat |
| ofoxai | O | apikey | OpenAI-compatible | https://api.ofox.ai/v1/chat/completions | - | openai-compat |
| ollama | 9 | apikey | Ollama | https://ollama.com/api/chat | 11 (ctx 977k) | 1P |
| ollama-cloud | O | apikey | OpenAI-compatible | https://ollama.com/v1/chat/completions | 12 (ctx 1024k) | openai-compat |
| ollama-local | 9 | apikey | Ollama | http://localhost:11434/api/chat | 114 (ctx 256k) | local |
| ollama-search | 9 |  |  | - | - | midia |
| omniroute | A | key + baseURL | OpenAI-compatible | operator-defined | - | gateway |
| oneminai | O | apikey | OpenAI-compatible | https://api.1min.ai/api/chat-with-ai | 1 | openai-compat |
| onomeo | O | apikey | OpenAI-compatible | https://onomeo.com/v1/chat/completions | - | openai-compat |
| openadapter | O | apikey | OpenAI-compatible | https://api.openadapter.in/v1/chat/completions | 1 (ctx 125k) | openai-compat |
| openai | 9+A+O | apikey/key/oauth | OpenAI-compatible / OpenAI Responses/Chat | https://api.openai.com/v1/chat/completions / https://api.openai.com/v1 | 36 (ctx 1025k) | openai-compat |
| openai-compatible | A | key + baseURL | OpenAI-compatible | operator-defined | - | openai-compat |
| opencode | 9+A | free/oauth/key | OpenAI-compatible / OpenAI-compatible | https://opencode.ai / https://opencode.ai/console | 4 (ctx 1024k) | gateway |
| opencode-go | 9+O | apikey | OpenAI-compatible | https://opencode.ai/zen/go/v1 / https://opencode.ai/zen/go/v1/chat/completions | 69 (ctx 1025k) | 1P |
| opencode-zen | 9+O | apikey | OpenAI-compatible | https://opencode.ai/zen/v1 / https://opencode.ai/zen/v1/chat/completions | 74 (ctx 1025k) | 1P |
| openference | O | oauth | OpenAI-compatible | https://api.openference.com/v1/chat/completions | 3 (ctx 830k) | gateway |
| openference-api | O | apikey | OpenAI-compatible | https://api.openference.com/v1/chat/completions | 3 (ctx 830k) | gateway |
| openrouter | 9+A+O | apikey/key | OpenAI-compatible / OpenAI-compatible | https://openrouter.ai/api/v1/chat/completions / https://openrouter.ai/api/v1 | 18 (ctx 256k) | gateway |
| openvecta | O | apikey | OpenAI-compatible | https://api.openvecta.com/v1/chat/completions | 9 (ctx 1024k) | openai-compat |
| opper | O | apikey | OpenAI-compatible | https://api.opper.ai/v3/compat/chat/completions | - | openai-compat |
| orcarouter | O | apikey | OpenAI-compatible | https://api.orcarouter.ai/v1/chat/completions | 8 (ctx 1025k) | gateway |
| ovhcloud | O | optional | OpenAI-compatible | https://oai.endpoints.kepler.ai.cloud.ovh.net/v1/chat/completions | 3 | openai-compat |
| perplexity | 9+A+O | apikey/key | OpenAI-compatible / OpenAI-compatible | https://api.perplexity.ai/chat/completions / https://api.perplexity.ai | 4 (ctx 125k) | openai-compat |
| perplexity-agent | 9+O | apikey | OpenAI Responses | https://api.perplexity.ai/v1/responses | 11 (ctx 1025k) | 1P |
| perplexity-web | 9+O | apikey/cookie | OpenAI-compatible / Perplexity Web (prop.) | https://www.perplexity.ai/rest/sse/perplexity_ask | 11 (ctx 1024k) | web |
| pioneer | O | apikey | OpenAI-compatible | https://api.pioneer.ai/v1/chat/completions | 10 | openai-compat |
| plamo | O | apikey | OpenAI-compatible | https://api.platform.preferredai.jp/v1/chat/completions | 1 (ctx 256k) | openai-compat |
| playht | 9 |  |  | - | - | midia |
| poe | O | apikey | OpenAI-compatible | https://api.poe.com/v1/chat/completions | 3 | openai-compat |
| poixe-ai | O | apikey | OpenAI-compatible | https://api.poixe.com/v1/chat/completions | - | gateway |
| pollinations | O | optional | OpenAI-compatible | https://gen.pollinations.ai/v1/chat/completions | 31 | 1P |
| poolside | 9+O | apikey | OpenAI-compatible | https://inference.poolside.ai/v1/chat/completions | 2 (ctx 977k) | openai-compat |
| predibase | O | apikey | OpenAI-compatible | https://serving.app.predibase.com/v1/chat/completions | 1 | openai-compat |
| promptql | O | apikey | OpenAI-compatible | https://data.prompt.ql.app/promptql/playground-v2-hge/v1/graphql | 14 | web |
| publicai | O | apikey | OpenAI-compatible | https://api.publicai.co/v1/chat/completions | 7 | openai-compat |
| qiniu | O | apikey | OpenAI-compatible | https://api.qnaigc.com/v1/chat/completions | - | openai-compat |
| qoder | 9+O | apikey/oauth | OpenAI-compatible | https://api.qoder.com/v1/chat/completions / https://api3.qoder.sh/algo/api/v2/service/pro/sse/agent_chat_generation | 18 (ctx 977k) | 1P |
| qoder-cn | 9 | oauth | OpenAI-compatible | https://gateway.qoder.com.cn/algo/api/v2/service/pro/sse/agent_chat_generation | 16 | openai-compat |
| qwen-cloud | O | apikey | OpenAI-compatible | https://dashscope-intl.aliyuncs.com/compatible-mode/v1/chat/completions | 14 | openai-compat |
| qwen-cloud-token-plan | 9+O | apikey | OpenAI-compatible | https://token-plan.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1/chat/completions | 9 (ctx 977k) | openai-compat |
| recraft | 9 |  |  | - | - | midia |
| regolo | O | apikey | OpenAI-compatible | https://api.regolo.ai/v1/chat/completions | - | openai-compat |
| reka | O | apikey | OpenAI-compatible | https://api.reka.ai/v1/chat/completions | 3 | openai-compat |
| requesty | O | apikey | OpenAI-compatible | https://router.requesty.ai/v1/chat/completions | - | gateway |
| routeway | O | apikey | OpenAI-compatible | https://api.routeway.ai/v1/chat/completions | 8 (ctx 250k) | openai-compat |
| runcrate | 9+O | apikey | OpenAI-compatible | https://api.runcrate.ai/v1/chat/completions | 6 (ctx 977k) | openai-compat |
| runwayml | 9 |  |  | - | - | midia |
| sambanova | 9+O | apikey | OpenAI-compatible | https://api.sambanova.ai/v1/chat/completions | 7 (ctx 200k) | openai-compat |
| sap-ai-core | A | oauth client credentials | OpenAI-compatible | deployment-defined | - | openai-compat |
| sarvam | O | apikey | OpenAI-compatible | https://api.sarvam.ai/v1/chat/completions | 2 (ctx 128k) | openai-compat |
| scaleway | O | apikey | OpenAI-compatible | https://api.scaleway.ai/v1/chat/completions | 6 | openai-compat |
| sdwebui | 9 |  |  | - | - | midia |
| sealion | O | apikey | OpenAI-compatible | https://api.sea-lion.ai/v1/chat/completions | 5 (ctx 128k) | openai-compat |
| searchapi | 9 |  |  | - | - | midia |
| searxng | 9 |  |  | - | - | midia |
| seekai | O | apikey | OpenAI-compatible | https://seekai.cc/v1/chat/completions | - | openai-compat |
| selfhosted-embedding | 9 |  |  | - | - | midia |
| selfhosted-stt | 9 |  |  | - | - | midia |
| selfhosted-tts | 9 |  |  | - | - | midia |
| sensenova | 9+O | apikey | OpenAI-compatible | https://token.sensenova.cn/v1/chat/completions | 7 (ctx 1024k) | openai-compat |
| serper | 9 |  |  | - | - | midia |
| siliconflow | 9+O | apikey | OpenAI-compatible | https://api.siliconflow.com/v1/chat/completions | 69 (ctx 977k) | openai-compat |
| snowflake | O | apikey | OpenAI-compatible | https://{account}.snowflakecomputing.com/api/v2 | 4 | openai-compat |
| snowflake-cortex | A | key/pat + account | Snowflake REST | https://{account}.snowflakecomputing.com | - | openai-compat |
| sparkdesk | O | apikey | OpenAI-compatible | https://spark-api-open.xf-yun.com/v1/chat/completions | 4 (ctx 128k) | openai-compat |
| speka | O | apikey | OpenAI-compatible | https://speka.me/v1/chat/completions | - | openai-compat |
| stability-ai | 9 |  |  | - | - | midia |
| stepfun | O | apikey | OpenAI-compatible | https://api.stepfun.com/v1/chat/completions | 6 (ctx 1000k) | openai-compat |
| sumopod | O | apikey | OpenAI-compatible | https://ai.sumopod.com/v1/chat/completions | - | openai-compat |
| synthetic | O | apikey | OpenAI-compatible | https://api.synthetic.new/openai/v1/chat/completions | 7 (ctx 512k) | openai-compat |
| syntx | O | apikey | OpenAI-compatible | https://api.syntx.ai/api/v1 | 34 | web |
| t3-web | O | apikey | OpenAI-compatible | https://t3.chat/api/chat | 23 | web |
| tabitoken | O | apikey | Anthropic Messages | https://tabitoken.com/v1/messages | 4 | 1P |
| tavily | 9 |  |  | - | - | midia |
| tencent | 9+O | apikey | OpenAI-compatible | https://api.hunyuan.cloud.tencent.com/v1/chat/completions | 6 (ctx 256k) | openai-compat |
| tencent-aistudio-web | O | apikey | OpenAI-compatible | https://aistudio.tencent.ai/api/chat | 3 | web |
| tinycms-web | O | apikey | OpenAI-compatible | https://gov.freegpt.win/api/openai/oneapi/v1/chat/completions | 22 | web |
| tinyfish | 9 |  |  | - | - | midia |
| together | 9+A+O | apikey/key | OpenAI-compatible / OpenAI-compatible | https://api.together.xyz/v1/chat/completions / https://api.together.xyz/v1 | 7 (ctx 977k) | openai-compat |
| token-kiosk | O | apikey | OpenAI-compatible | https://agent-router.gaib.ai/v1/chat/completions | 5 (ctx 195k) | gateway |
| tokenharbor | 9+O | apikey | OpenAI-compatible | https://tokenharbor.ai/v1/chat/completions | 6 (ctx 1025k) | gateway |
| tokenmarket | O | apikey | OpenAI-compatible | https://api.tokensmarket.ai/v1/chat/completions | - | gateway |
| tokenreply | O | apikey | OpenAI-compatible | https://api.tokenreply.com/v1/chat/completions | - | gateway |
| tokenrouter | 9+O | apikey | OpenAI-compatible | https://api.tokenrouter.com/v1/chat/completions | 22 (ctx 1025k) | gateway |
| topaz | 9 |  |  | - | - | midia |
| tortoise | 9 |  |  | - | - | midia |
| trae | O | oauth | OpenAI-compatible | https://core-normal.trae.ai/api/remote/v1 | 9 (ctx 1024k) | 1P |
| twinmind | O | apikey | OpenAI-compatible | https://api2.twinmind.com/api/v3 | 13 | web |
| typhoon | O | apikey | OpenAI-compatible | https://api.opentyphoon.ai/v1/chat/completions | 1 (ctx 128k) | openai-compat |
| uc | 9+O | none/cookie | OpenAI-compatible | https://internal-6.pubyar.com / https://internal.chatuncensored.ai | 53 (ctx 1953k) | 1P |
| uc-direct | O | apikey | OpenAI-compatible | https://api.uncensored.com/api/v1 | 82 | openai-compat |
| udio | O | apikey | OpenAI-compatible | https://www.udio.com/api/generate-proxy | 1 | openai-compat |
| uncloseai | O | optional | OpenAI-compatible | https://hermes.ai.unturf.com/v1/chat/completions | 1 (ctx 64k) | openai-compat |
| unifically | O | apikey | OpenAI-compatible | https://api.unifically.com/v1/chat/completions | - | openai-compat |
| unorouter | O | apikey | OpenAI-compatible | https://api.unorouter.com/v1/chat/completions | - | gateway |
| upstage | O | apikey | OpenAI-compatible | https://api.upstage.ai/v1/chat/completions | 2 | openai-compat |
| v0-vercel | O | apikey | OpenAI-compatible | https://api.v0.dev/v1/chat/completions | 3 | openai-compat |
| v1m | 9 |  |  | - | - | midia |
| venice | 9+A+O | apikey/key | OpenAI-compatible / OpenAI-compatible | https://api.venice.ai/api/v1/chat/completions / https://api.venice.ai/api/v1 | 15 (ctx 977k) | openai-compat |
| veoaifree-web | 9+O | none | OpenAI-compatible | https://veoaifree.com/wp-admin/admin-ajax.php | 2 | midia |
| vercel-ai-gateway | 9+A+O | apikey/key | OpenAI-compatible / OpenAI-compatible | https://ai-gateway.vercel.sh/v1/chat/completions / https://ai-gateway.vercel.sh/v1 | 5 | gateway |
| vertex | 9+A+O | apikey/freeTier/service account/ADC | Gemini generateContent / Gemini/Vertex / Gemini/Vertex / Anthropic Messages | https://us-central1-aiplatform.googleapis.com/v1/projects / https://aiplatform.googleapis.com | 34 (ctx 1024k) | 1P |
| vertex-partner | 9+O | apikey | Gemini generateContent / OpenAI-compatible | https://us-central1-aiplatform.googleapis.com/v1/projects / https://aiplatform.googleapis.com | 24 (ctx 512k) | 1P |
| void-ai | O | apikey | OpenAI-compatible | https://api.voidai.app/v1/chat/completions | - | openai-compat |
| volcengine | 9+O | apikey | OpenAI-compatible | https://ark.cn-beijing.volces.com/api/v3/chat/completions / https://ark.cn-beijing.volces.com/api/coding/v3/chat/completions | 9 (ctx 977k) | openai-compat |
| volcengine-agent-plan | O | apikey | OpenAI-compatible | https://ark.cn-beijing.volces.com/api/plan/v3/chat/completions | 12 (ctx 1024k) | openai-compat |
| volcengine-coding-plan | O | apikey | OpenAI-compatible | https://ark.cn-beijing.volces.com/api/coding/v3/chat/completions | 9 (ctx 1024k) | openai-compat |
| voyage-ai | 9 |  |  | - | - | midia |
| wafer | O | apikey | Anthropic Messages | https://pass.wafer.ai/v1/messages | 4 | 1P |
| wandb | O | apikey | OpenAI-compatible | https://api.inference.wandb.ai/v1/chat/completions | 3 | openai-compat |
| workbuddy | O | oauth | OpenAI-compatible | https://www.workbuddy.ai/v2/chat/completions | - | openai-compat |
| writer | O | apikey | OpenAI-compatible | https://api.writer.com/v1/chat/completions | 2 (ctx 1024k) | openai-compat |
| x5lab | O | apikey | OpenAI-compatible | https://api.x5lab.dev/v1/chat/completions | - | openai-compat |
| xai | 9+A+O | apikey/oauth/key | OpenAI-compatible / OpenAI Responses | https://api.x.ai/v1/chat/completions / https://api.x.ai/v1 | 9 (ctx 488k) | 1P |
| xai-oauth | O | oauth | OpenAI-compatible | https://api.x.ai/v1/chat/completions | 7 (ctx 488k) | 1P |
| xiaomi-mimo | 9+O | apikey/free/oauth | OpenAI-compatible | https://api.xiaomimimo.com/v1 / https://api.xiaomimimo.com/api/free-ai/openai/chat | 8 (ctx 1024k) | openai-compat |
| xiaomi-mimo-token-plan | 9+O | apikey | OpenAI-compatible | https://token-plan-sgp.xiaomimimo.com/v1 / https://token-plan-sgp.xiaomimimo.com/v1/chat/completions | 9 (ctx 1024k) | openai-compat |
| xkiro | O | apikey | OpenAI-compatible | https://api.xkiro.com/v1/chat/completions | 39 (ctx 1024k) | openai-compat |
| xquik | 9 |  |  | - | - | midia |
| y-api | O | apikey | OpenAI-compatible | https://api.y-api.bestvirtualgoods.com/v1/chat/completions | - | openai-compat |
| yi | O | apikey | OpenAI-compatible | https://api.lingyiwanwu.com/v1/chat/completions | 1 | openai-compat |
| yolo-auto | O | apikey | OpenAI-compatible | https://yolo-auto.com/v1/chat/completions | 1 | openai-compat |
| youcom | 9 |  |  | - | - | midia |
| yuanbao-web | O | apikey | OpenAI-compatible | https://yuanbao.tencent.com/api/chat | 8 | web |
| zai | O | apikey | Anthropic Messages | https://api.z.ai/api/anthropic/v1/messages | 8 (ctx 977k) | 1P |
| zai-web | O | apikey | OpenAI-compatible | https://chat.z.ai | 3 | web |
| zcode | O | none | OpenAI-compatible | zcode://app-server/stdio | 14 (ctx 977k) | web |
| zed | 9 | oauth | OpenAI-compatible | https://cloud.zed.dev/completions | - | web |
| zed-hosted | O | oauth | OpenAI-compatible | https://cloud.zed.dev/completions | - | web |
| zenmux | A+O | apikey/key | OpenAI-compatible / OpenAI-compatible | https://zenmux.ai/api/v1/chat/completions / https://zenmux.ai/api/v1 | 9 (ctx 1024k) | openai-compat |
| zenmux-free | O | apikey | OpenAI-compatible | https://zenmux.ai/api/anthropic/v1/messages | 11 | 1P |
| zerolimitai | O | apikey | OpenAI-compatible | https://www.zerolimitai.com/api/v1/chat/completions | - | openai-compat |
| zylo-api | O | apikey | OpenAI-compatible | https://api.zyloai.net/v1/chat/completions | - | openai-compat |
