# Anexo B — Modelos e preços das fontes

> **Documento gerado por extração automatizada das fontes** (ver `PROVIDERS-MAPPING.md`, seção 0).
> Não editar manualmente.
>
> **Advertência:** os IDs de modelo e preços abaixo são exatamente o que está declarado nas
> fontes na data da extração (2026-11-06). Eles **não foram validados** contra a API ou a página
> de preços de cada fornecedor. Antes de cadastrar no catálogo do dontstop, cada entrada precisa
> de verificação de fornecedor (ver checklist na seção 8.7 do documento principal).

## B.1 Modelos por provedor (flags declaradas)

Fonte: registry do OmniRoute (flags `supportsReasoning`, `supportsVision`, `toolCalling`,
`contextLength`, `maxOutputTokens`) e registry do 9Router quando o OmniRoute não tem catálogo
estático. Preço é preenchido quando o ID existe na tabela canônica do 9Router (`pricing.js`);
`nao encontrado` significa que a fonte não declara preço para aquele ID.

- `-` em qualquer flag = não declarado na fonte (não significa "não suporta").
- `ctx` em tokens; `max out` em tokens; preço em USD por 1M de tokens.

### anthropic (14 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| claude-fable-5-1 | 1000000 | 128000 | sim | sim | - | nao encontrado |
| claude-fable-5 | 1048576 | - | sim | - | - | 10/50 |
| claude-opus-5-5 | 1000000 | 128000 | sim | sim | - | 4/20 |
| claude-opus-5 | 1000000 | 128000 | sim | - | - | nao encontrado |
| claude-opus-4.7 | - | - | sim | - | - | nao encontrado |
| claude-opus-4.8 | 1048576 | - | sim | - | - | nao encontrado |
| claude-opus-4.6 | - | - | sim | - | - | 5/25 |
| claude-opus-4.5 | - | - | sim | - | - | 5/25 |
| claude-sonnet-5 | 1048576 | - | - | - | - | 2/10 |
| claude-sonnet-5-5 | 1048576 | - | - | - | - | 2/10 |
| claude-sonnet-4.6 | - | - | - | - | - | 3/15 |
| claude-sonnet-4.5 | - | - | - | - | - | 3/15 |
| claude-haiku-4.5 | - | - | - | - | - | 0.5/2.5 |
| claude-haiku-5-5 | 1000000 | 128000 | sim | - | - | nao encontrado |

### claude (14 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| claude-fable-5-1 | 1000000 | 128000 | sim | sim | - | nao encontrado |
| claude-fable-5 | 1000000 | 128000 | sim | - | - | 10/50 |
| claude-opus-5 | 1000000 | 128000 | sim | - | - | nao encontrado |
| claude-opus-5-5 | 1000000 | 128000 | sim | sim | - | 4/20 |
| claude-opus-4-8 | 1000000 | 128000 | sim | - | - | nao encontrado |
| claude-opus-4-7 | 1000000 | 128000 | sim | - | - | nao encontrado |
| claude-opus-4-6 | 1000000 | 128000 | sim | - | - | 5/25 |
| claude-opus-4-5-20251101 | 200000 | 64000 | sim | - | - | 5/25 |
| claude-sonnet-5 | 1000000 | 128000 | - | - | - | 2/10 |
| claude-sonnet-5-5 | 1000000 | 128000 | - | - | - | 2/10 |
| claude-sonnet-4-6 | 1000000 | 64000 | - | - | - | 3/15 |
| claude-sonnet-4-5-20250929 | 200000 | 64000 | - | - | - | 3/15 |
| claude-haiku-4-5-20251001 | 200000 | 64000 | - | - | - | 1/5 |
| claude-haiku-5-5 | 1000000 | 128000 | - | - | - | nao encontrado |

### openai (24 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| gpt-6.1-sol | 1050000 | 128000 | sim | sim | sim | 2/10 |
| gpt-6-astra | 1050000 | 128000 | sim | sim | sim | 10/50 |
| gpt-6-sol | 1050000 | 128000 | sim | sim | sim | 2/10 |
| gpt-6-luna | 1050000 | 128000 | sim | sim | sim | 0.1/0.5 |
| gpt-5.6 | 1050000 | 128000 | sim | sim | sim | 2.5/15 |
| gpt-5.6-sol | 1050000 | 128000 | sim | sim | sim | 5/30 |
| gpt-5.6-terra | 1050000 | 128000 | sim | sim | sim | 2.5/15 |
| gpt-5.6-luna | 1050000 | 128000 | sim | sim | sim | 1/6 |
| gpt-5.5 | 1050000 | - | - | - | - | nao encontrado |
| gpt-5.5-pro | 1050000 | - | - | - | - | nao encontrado |
| gpt-5.4 | 1050000 | - | - | - | - | nao encontrado |
| gpt-5.4-pro | 1050000 | - | - | - | - | nao encontrado |
| gpt-5.4-mini | 400000 | - | - | - | - | nao encontrado |
| gpt-5.4-nano | 400000 | - | - | - | - | nao encontrado |
| gpt-4.1 | 1047576 | - | - | - | - | 2.5/10 |
| gpt-4.1-mini | 1047576 | - | - | - | - | nao encontrado |
| gpt-4.1-nano | 1047576 | - | - | - | - | nao encontrado |
| gpt-4o | 128000 | - | - | - | - | 2.5/10 |
| gpt-4o-2024-11-20 | 128000 | - | - | - | - | nao encontrado |
| gpt-4o-mini | 128000 | - | - | - | - | 0.15/0.6 |
| o3 | 200000 | - | - | - | - | nao encontrado |
| o3-mini | 200000 | - | - | - | - | nao encontrado |
| o4-mini | 200000 | - | - | - | - | nao encontrado |

### codex (52 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| gpt-6-astra | 872000 | 128000 | sim | sim | sim | 10/50 |
| gpt-6-astra-ultra | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6-astra-max | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6-astra-xhigh | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6-astra-high | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6-astra-medium | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6-astra-low | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6-sol | 872000 | 128000 | sim | sim | sim | 2/10 |
| gpt-6-sol-ultra | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6-sol-max | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6-sol-xhigh | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6-sol-high | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6-sol-medium | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6-sol-low | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6.1-sol | 872000 | 128000 | sim | sim | sim | 2/10 |
| gpt-6.1-sol-ultra | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6.1-sol-max | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6.1-sol-xhigh | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6.1-sol-high | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6.1-sol-medium | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6.1-sol-low | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6-luna | 872000 | 128000 | sim | sim | sim | 0.1/0.5 |
| gpt-6-luna-max | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6-luna-xhigh | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6-luna-high | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6-luna-medium | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-6-luna-low | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.6-sol | 872000 | 128000 | sim | sim | sim | 5/30 |
| gpt-5.6-sol-ultra | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.6-sol-max | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.6-sol-xhigh | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.6-sol-high | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.6-sol-medium | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.6-sol-low | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.6-terra | 872000 | 128000 | sim | sim | sim | 2.5/15 |
| gpt-5.6-terra-ultra | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.6-terra-max | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.6-terra-xhigh | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.6-terra-high | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.6-terra-medium | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.6-terra-low | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.6-luna | 872000 | 128000 | sim | sim | sim | 1/6 |
| gpt-5.6-luna-max | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.6-luna-xhigh | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.6-luna-high | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.6-luna-medium | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.6-luna-low | 872000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.5 | 400000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.5-xhigh | 400000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.5-high | 400000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.5-medium | 400000 | 128000 | sim | sim | sim | nao encontrado |
| gpt-5.5-low | 400000 | 128000 | sim | sim | sim | nao encontrado |

### gemini (10 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| gemini-3.8-flash | - | - | - | sim | sim | 1.5/7.5 |
| gemini-4-argon | - | - | - | sim | sim | nao encontrado |
| gemini-3.7-flash | - | - | - | sim | sim | 1.5/7.5 |
| gemini-3.1-pro-preview | - | - | - | sim | sim | nao encontrado |
| gemini-3.1-flash-lite | - | - | - | sim | sim | nao encontrado |
| gemini-3-flash-preview | - | - | - | sim | sim | 0.5/3 |
| gemini-3.1-flash-tts-preview | - | - | - | - | - | nao encontrado |
| gemini-2.5-pro | - | - | - | sim | sim | 2/12 |
| gemini-2.5-flash | - | - | - | sim | sim | 0.3/2.5 |
| gemini-2.5-flash-lite | - | - | - | sim | sim | 0.15/1.25 |

### vertex (34 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| gemini-4-argon | - | - | - | - | - | nao encontrado |
| gemini-3.7-flash | - | - | - | - | - | 1.5/7.5 |
| gemini-3.6-flash | - | - | - | - | - | 1.5/7.5 |
| gemini-3.5-flash | - | - | - | - | - | nao encontrado |
| gemini-3.5-flash-lite | - | - | - | - | - | 0.3/2.5 |
| gemini-3.1-pro-preview | - | - | - | - | - | nao encontrado |
| gemini-3.1-flash-lite | - | - | - | - | - | nao encontrado |
| gemini-3-flash-preview | - | - | - | - | - | 0.5/3 |
| gemini-2.5-pro | - | - | - | - | - | 2/12 |
| gemini-2.5-flash | - | - | - | - | - | 0.3/2.5 |
| gemini-2.5-flash-lite | - | - | - | - | - | 0.15/1.25 |
| deepseek-ai/deepseek-v3.2-maas | - | - | - | - | - | nao encontrado |
| deepseek-ai/deepseek-v3.1-maas | - | - | - | - | - | nao encontrado |
| qwen/qwen3-next-80b-a3b-instruct-maas | - | - | - | - | - | nao encontrado |
| zai-org/glm-5-maas | - | - | - | - | - | nao encontrado |
| xai/grok-4.6 | 524288 | - | sim | sim | sim | nao encontrado |
| xai/grok-4.3 | - | - | - | - | - | nao encontrado |
| xai/grok-4.20-reasoning | - | - | sim | - | - | nao encontrado |
| xai/grok-4.20-non-reasoning | - | - | - | - | - | nao encontrado |
| claude-fable-5-1 | - | - | - | - | - | nao encontrado |
| claude-fable-5 | - | - | - | - | - | 10/50 |
| claude-opus-5-5 | - | - | - | - | - | 4/20 |
| claude-opus-5 | - | - | - | - | - | nao encontrado |
| claude-sonnet-5 | - | - | - | - | - | 2/10 |
| claude-sonnet-5-5 | - | - | - | - | - | 2/10 |
| claude-opus-4-8 | - | - | - | - | - | nao encontrado |
| claude-opus-4-7 | - | - | - | - | - | nao encontrado |
| claude-opus-4-6 | - | - | - | - | - | 5/25 |
| claude-sonnet-4-6 | - | - | - | - | - | 3/15 |
| claude-sonnet-4-5-v2 | - | - | - | - | - | nao encontrado |
| claude-sonnet-4-5 | - | - | - | - | - | nao encontrado |
| claude-opus-4-5 | - | - | - | - | - | nao encontrado |
| claude-haiku-4-5 | - | - | - | - | - | nao encontrado |
| claude-haiku-5-5 | - | - | - | - | - | nao encontrado |

### bedrock (9 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| anthropic.claude-fable-5-1 | 1000000 | 128000 | sim | sim | sim | nao encontrado |
| anthropic.claude-opus-5-5 | 1000000 | 128000 | sim | sim | sim | nao encontrado |
| anthropic.claude-sonnet-4-6 | 1000000 | - | sim | sim | sim | nao encontrado |
| anthropic.claude-sonnet-4-5 | 200000 | - | - | sim | sim | nao encontrado |
| anthropic.claude-opus-4-6 | 1000000 | - | sim | sim | sim | nao encontrado |
| anthropic.claude-opus-4-7 | 1000000 | - | sim | sim | sim | nao encontrado |
| anthropic.claude-haiku-4-5 | - | - | - | sim | sim | nao encontrado |
| anthropic.claude-haiku-5-5 | 1000000 | - | sim | sim | sim | nao encontrado |
| openai.gpt-oss-120b-1:0 | - | - | - | - | - | nao encontrado |

### azure

Nenhum modelo estatico na fonte (catalogo dinamico em runtime).

### cloudflare-ai (9 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| @cf/mistral/mistral-7b-instruct-v0.2-lora | 15000 | - | - | - | - | nao encontrado |
| @cf/qwen/qwen2.5-coder-32b-instruct | 32768 | - | - | - | - | nao encontrado |
| @cf/deepseek-ai/deepseek-r1-distill-qwen-32b | 80000 | - | - | - | - | nao encontrado |
| @cf/meta/llama-3.3-70b-instruct-fp8-fast | 24000 | - | - | - | - | nao encontrado |
| @cf/meta/llama-3.2-3b-instruct | 80000 | - | - | - | - | nao encontrado |
| @cf/qwen/qwq-32b | 24000 | - | - | - | - | nao encontrado |
| @cf/zai-org/glm-4.7-flash | 131072 | - | - | - | - | nao encontrado |
| @cf/moonshotai/kimi-k2.6 | 262144 | - | - | - | - | 1/4 |
| @cf/google/gemma-4-26b-a4b-it | 262144 | - | - | - | - | nao encontrado |

### github (16 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| gpt-6-astra | 1000000 | 128000 | sim | sim | sim | 10/50 |
| gpt-5.6-sol | 1050000 | 128000 | sim | sim | sim | 5/30 |
| gpt-5.6-terra | 1050000 | 128000 | sim | sim | sim | 2.5/15 |
| gpt-5.6-luna | 1050000 | 128000 | sim | sim | sim | 1/6 |
| claude-fable-5.1 | 1000000 | 64000 | sim | sim | sim | nao encontrado |
| claude-opus-5 | 1000000 | 64000 | sim | sim | sim | nao encontrado |
| claude-sonnet-5 | 1000000 | 64000 | sim | sim | sim | 2/10 |
| claude-sonnet-5-5 | 1000000 | 128000 | - | - | - | 2/10 |
| claude-opus-4.8-fast | 1000000 | 64000 | sim | sim | sim | nao encontrado |
| claude-opus-4.8 | 1000000 | 64000 | sim | sim | sim | nao encontrado |
| claude-haiku-4.5 | 144000 | 32000 | sim | sim | sim | 0.5/2.5 |
| gemini-3.8-flash | 1048576 | 65536 | sim | sim | sim | 1.5/7.5 |
| mai-code-1.1-flash | 256000 | 128000 | sim | sim | sim | nao encontrado |
| kimi-k3 | 1048576 | 131072 | sim | sim | sim | 3/15 |
| grok-4.6 | 500000 | 128000 | - | - | - | nao encontrado |
| auto | - | - | - | - | sim | 2/8 |

### gitlab-duo (2 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| claude-sonnet-4-6 | - | - | sim | - | - | 3/15 |
| claude-haiku-4-5 | - | - | - | - | - | nao encontrado |

### xai (6 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| grok-4.6 | 500000 | - | sim | sim | sim | nao encontrado |
| grok-4.3 | - | - | - | - | - | nao encontrado |
| grok-build-0.1 | 256000 | - | - | - | - | nao encontrado |
| grok-4.20-multi-agent-0309 | - | - | - | - | - | nao encontrado |
| grok-4.20-0309-reasoning | - | - | - | - | - | nao encontrado |
| grok-4.20-0309-non-reasoning | - | - | - | - | - | nao encontrado |

### deepseek (3 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| deepseek-v4-pro | 1000000 | 384000 | sim | - | sim | 0.435/0.87 |
| deepseek-v4-flash | 1000000 | 384000 | sim | sim | sim | 0.14/0.28 |
| deepseek-flash | 1000000 | 384000 | sim | sim | sim | 0.14/0.28 |

### kimi (6 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| kimi-k3.1-pro | 1048576 | 1048576 | sim | sim | sim | 2/10 |
| kimi-k3.1-pro-highspeed | 1048576 | 1048576 | sim | sim | sim | nao encontrado |
| kimi-k3 | 1048576 | 1048576 | sim | sim | sim | 3/15 |
| kimi-k2.7-code | 262144 | 262144 | sim | sim | sim | 0.95/4 |
| kimi-k2.7-code-highspeed | 262144 | 262144 | sim | sim | sim | 1.9/8 |
| kimi-k2.6 | 262144 | 262144 | sim | sim | sim | 1/4 |

### kimi-coding (5 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| k3.1-pro | 1048576 | - | sim | - | - | nao encontrado |
| k3.1-pro-highspeed | 1048576 | - | sim | - | - | nao encontrado |
| k3 | 1048576 | - | sim | - | - | 3/15 |
| kimi-for-coding | 1048576 | - | sim | - | - | 0.95/4 |
| kimi-for-coding-highspeed | 1048576 | - | sim | - | - | 1.9/8 |

### glm (21 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| glm-5.3-flash | 1000000 | 131072 | sim | sim | sim | nao encontrado |
| glm-5.3 | 1000000 | 131072 | sim | nao | sim | nao encontrado |
| glm-5.3-high | 1000000 | 131072 | sim | nao | sim | nao encontrado |
| glm-5.3-low | 1000000 | 131072 | sim | nao | sim | nao encontrado |
| glm-5.3-max | 1000000 | 131072 | sim | nao | sim | nao encontrado |
| glm-5.3-flash-high | 1000000 | 131072 | sim | sim | sim | nao encontrado |
| glm-5.3-flash-low | 1000000 | 131072 | sim | sim | sim | nao encontrado |
| glm-5.3-flash-max | 1000000 | 131072 | sim | sim | sim | nao encontrado |
| glm-5.2 | 1000000 | 131072 | sim | - | sim | nao encontrado |
| glm-5.2-high | 1000000 | 131072 | sim | - | sim | nao encontrado |
| glm-5.2-max | 1000000 | 131072 | sim | - | sim | nao encontrado |
| glm-5.1 | 204800 | 131072 | sim | - | sim | nao encontrado |
| glm-5 | 200000 | 131072 | sim | - | sim | 1/4 |
| glm-5-turbo | 200000 | 131072 | sim | - | sim | nao encontrado |
| glm-4.7-flash | 200000 | 131072 | sim | - | sim | nao encontrado |
| glm-4.7 | 200000 | 131072 | sim | - | sim | 0.75/3 |
| glm-4.6v | 128000 | 32768 | sim | sim | sim | 0.75/3 |
| glm-4.6 | 200000 | 32768 | sim | - | sim | 0.5/2 |
| glm-4.5v | 16000 | 32768 | sim | sim | sim | nao encontrado |
| glm-4.5 | 128000 | 32768 | sim | - | sim | nao encontrado |
| glm-4.5-air | 128000 | 32768 | sim | - | sim | nao encontrado |

### zai (8 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| glm-5.3 | 1000000 | 131072 | sim | - | sim | nao encontrado |
| glm-5.3-flash | 1000000 | 131072 | sim | sim | sim | nao encontrado |
| glm-5.2 | - | - | - | - | - | nao encontrado |
| glm-5.1 | - | - | - | - | - | nao encontrado |
| glm-5 | - | - | - | - | - | 1/4 |
| glm-5-turbo | - | - | - | - | - | nao encontrado |
| glm-4.7-flash | - | - | - | - | - | nao encontrado |
| glm-4.7 | - | - | - | - | - | 0.75/3 |

### minimax (6 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| MiniMax-M3 | 1048576 | - | - | sim | - | 0.3/1.2 |
| MiniMax-M3.1-Flash | 1000000 | - | sim | sim | - | nao encontrado |
| MiniMax-M2.7 | - | - | - | - | - | 0.5/2 |
| MiniMax-M2.7-highspeed | - | - | - | - | - | nao encontrado |
| MiniMax-M2.5 | - | - | - | - | - | 0.5/2 |
| MiniMax-M2.5-highspeed | - | - | - | - | - | nao encontrado |

### alibaba (14 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| qwen3.8-max | - | - | - | - | - | nao encontrado |
| qwen3.7-max | - | - | - | - | - | nao encontrado |
| qwen3.7-plus | - | - | - | - | - | nao encontrado |
| qwen3.6-plus | - | - | - | - | - | nao encontrado |
| qwen3.6-27b | - | - | - | - | - | nao encontrado |
| qwen3.6-35b-a3b | - | - | - | - | - | nao encontrado |
| qwen3.5-plus | - | - | - | - | - | nao encontrado |
| qwen3.5-122b-a10b | - | - | - | - | - | nao encontrado |
| qwen3.5-397b-a17b | - | - | - | - | - | nao encontrado |
| glm-5.2 | - | - | - | - | - | nao encontrado |
| glm-5.2-fast-preview | - | - | - | - | - | nao encontrado |
| deepseek-v4-pro | - | - | - | - | - | 0.435/0.87 |
| deepseek-v4-flash | - | - | - | - | - | 0.14/0.28 |
| kimi-k2.7-code | - | - | - | - | - | 0.95/4 |

### baidu (16 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| ernie-5.1 | 131072 | - | - | - | - | nao encontrado |
| ernie-5.0 | 131072 | - | - | - | - | nao encontrado |
| ernie-x1.1 | 32768 | - | - | - | - | nao encontrado |
| ernie-4.5-turbo-128k | 131072 | - | - | - | - | nao encontrado |
| ernie-4.5-turbo-32k | 32768 | - | - | - | - | nao encontrado |
| ernie-4.5-turbo-vl | 131072 | - | - | - | - | nao encontrado |
| ernie-4.5-21b-a3b | 131072 | - | - | - | - | nao encontrado |
| ernie-4.5-0.3b | 131072 | - | - | - | - | nao encontrado |
| ernie-4.0-8k | - | - | - | - | - | nao encontrado |
| ernie-4.0-turbo-128k | 131072 | - | - | - | - | nao encontrado |
| ernie-4.0-turbo-8k | 8192 | - | - | - | - | nao encontrado |
| ernie-3.5-8k | 8192 | - | - | - | - | nao encontrado |
| ernie-speed-128k | 131072 | - | - | - | - | nao encontrado |
| ernie-speed-8k | 8192 | - | - | - | - | nao encontrado |
| ernie-lite-8k | 8192 | - | - | - | - | nao encontrado |
| ernie-tiny-8k | 8192 | - | - | - | - | nao encontrado |

### volcengine (7 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| deepseek-v3-2-251201 | - | - | - | - | - | nao encontrado |
| doubao-seed-2-0-pro-260215 | - | - | - | - | - | nao encontrado |
| doubao-seed-2-0-code-preview-260215 | - | - | - | - | - | nao encontrado |
| kimi-k2-5-260127 | - | - | - | - | - | nao encontrado |
| glm-4-7-251222 | - | - | - | - | - | nao encontrado |
| DeepSeek-V4-Flash | - | - | - | - | - | nao encontrado |
| DeepSeek-V4-Pro | - | - | - | - | - | nao encontrado |

### sensenova (3 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| sensenova-6.7-flash-lite | 262144 | 65536 | - | sim | sim | nao encontrado |
| deepseek-v4-flash | 1048576 | 65536 | sim | - | - | 0.14/0.28 |
| glm-5.2 | 1048576 | 65536 | sim | - | - | nao encontrado |

### stepfun (6 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| step-5-preview | 1024000 | - | sim | sim | sim | nao encontrado |
| step-3.7-flash | 262144 | - | - | - | - | nao encontrado |
| step-3.5-flash | 262144 | - | - | - | - | nao encontrado |
| step-3.5-flash-2603 | 262144 | - | - | - | - | nao encontrado |
| step-1o-turbo-vision | 32768 | - | - | - | - | nao encontrado |
| step-1v | - | - | - | - | - | nao encontrado |

### tencent (6 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| hunyuan-turbos-latest | 200000 | - | - | - | - | nao encontrado |
| hunyuan-t1-latest | 256000 | - | - | - | - | nao encontrado |
| hunyuan-pro | - | - | - | - | - | nao encontrado |
| hunyuan-vision | - | - | - | - | - | nao encontrado |
| hunyuan-functioncall | - | - | - | - | - | nao encontrado |
| hunyuan-lite | - | - | - | - | - | nao encontrado |

### xiaomi-mimo (2 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| mimo-v2.5-pro | 1048576 | 131072 | - | - | - | nao encontrado |
| mimo-v2.5 | 1048576 | 131072 | - | - | - | nao encontrado |

### openrouter (1 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| auto | - | - | - | - | - | 2/8 |

### perplexity (4 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| sonar-deep-research | - | - | - | - | - | nao encontrado |
| sonar-reasoning-pro | - | - | - | - | - | nao encontrado |
| sonar-pro | - | - | - | - | - | nao encontrado |
| sonar | - | - | - | - | - | nao encontrado |

### cohere (6 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| command-a-reasoning-08-2025 | - | - | - | - | - | nao encontrado |
| command-a-vision-07-2025 | - | - | - | - | - | nao encontrado |
| command-a-03-2025 | - | - | - | - | - | nao encontrado |
| command-r7b-12-2024 | - | - | - | - | - | nao encontrado |
| command-r-plus-08-2024 | - | - | - | - | - | nao encontrado |
| command-r-08-2024 | - | - | - | - | - | nao encontrado |

### mistral (9 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| mistral-large-latest | 262144 | - | - | sim | sim | nao encontrado |
| mistral-large-4 | 1048576 | - | sim | sim | sim | nao encontrado |
| mistral-medium-3-5 | 262144 | - | sim | sim | sim | nao encontrado |
| mistral-small-latest | 262144 | - | sim | sim | sim | nao encontrado |
| codestral-latest | 131072 | - | - | - | sim | nao encontrado |
| ministral-14b-latest | 262144 | - | - | - | sim | nao encontrado |
| ministral-8b-latest | 262144 | - | - | - | sim | nao encontrado |
| ministral-3b-latest | 262144 | - | - | - | sim | nao encontrado |
| voxtral-small-latest | 32768 | - | - | - | sim | nao encontrado |

### groq (7 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| groq/compound | - | - | nao | - | - | nao encontrado |
| allam-2-7b | - | - | nao | - | - | nao encontrado |
| openai/gpt-oss-120b | - | - | - | - | - | nao encontrado |
| openai/gpt-oss-20b | - | - | - | - | - | nao encontrado |
| qwen/qwen3.6-27b | - | - | - | - | - | nao encontrado |
| qwen/qwen3.8-27b | - | - | - | - | - | nao encontrado |
| openai/gpt-oss-safeguard-20b | - | - | - | - | - | nao encontrado |

### cerebras (3 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| zai-glm-4.7 | - | - | - | - | - | nao encontrado |
| gemma-4-31b | - | - | - | - | - | nao encontrado |
| gpt-oss-120b | - | - | - | - | - | nao encontrado |

### together (7 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| meta-llama/Llama-3.3-70B-Instruct-Turbo-Free | - | - | - | - | - | nao encontrado |
| meta-llama/Llama-Vision-Free | - | - | - | - | - | nao encontrado |
| deepseek-ai/DeepSeek-R1-Distill-Llama-70B-Free | - | - | - | - | - | nao encontrado |
| meta-llama/Llama-3.3-70B-Instruct-Turbo | - | - | - | - | - | nao encontrado |
| deepseek-ai/DeepSeek-R1 | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3-235B-A22B | - | - | - | - | - | nao encontrado |
| meta-llama/Llama-4-Maverick-17B-128E-Instruct-FP8 | - | - | - | - | - | nao encontrado |

### fireworks (10 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| deepseek-v4-flash | - | - | sim | - | - | 0.14/0.28 |
| deepseek-v4-pro | - | - | sim | - | - | 0.435/0.87 |
| glm-5p1 | - | - | - | - | - | nao encontrado |
| gpt-oss-120b | - | - | - | - | - | nao encontrado |
| gpt-oss-20b | - | - | - | - | - | nao encontrado |
| kimi-k2p5 | - | - | - | - | - | nao encontrado |
| kimi-k2p6 | - | - | - | - | - | nao encontrado |
| minimax-m2p5 | - | - | - | - | - | nao encontrado |
| minimax-m2p7 | - | - | - | - | - | nao encontrado |
| qwen3p6-plus | - | - | - | - | - | nao encontrado |

### nvidia (8 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| meta/muse-glimmer-30b | - | - | - | - | - | nao encontrado |
| poolside/laguna-xs-2.1 | - | - | - | - | - | nao encontrado |
| google/gemma-4-31b-it | - | - | - | - | - | nao encontrado |
| google/diffusiongemma-26b-a4b-it | - | - | - | - | - | nao encontrado |
| nvidia/nemotron-3-ultra-550b-a55b | - | - | - | - | - | nao encontrado |
| nvidia/nemotron-3.5-super-vl-120b-a12b | - | - | - | sim | - | nao encontrado |
| nvidia/nemotron-3.5-lightning-30b-a3b | - | - | - | - | - | nao encontrado |
| nvidia/nemotron-3-nano-omni-30b-a3b-reasoning | - | - | sim | sim | - | nao encontrado |

### deepinfra (22 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| anthropic/claude-4-opus | - | - | - | - | - | nao encontrado |
| anthropic/claude-4-sonnet | - | - | - | - | - | nao encontrado |
| openai/gpt-oss-120b | - | - | - | - | - | nao encontrado |
| openai/gpt-oss-20b | - | - | - | - | - | nao encontrado |
| google/gemma-4-31B-it | - | - | - | - | - | nao encontrado |
| google/gemma-4-26B-A4B-it | - | - | - | - | - | nao encontrado |
| nvidia/NVIDIA-Nemotron-3-Super-120B-A12B | - | - | - | - | - | nao encontrado |
| nvidia/Nemotron-3-Nano-Omni-30B-A3B-Reasoning | - | - | - | - | - | nao encontrado |
| meta-llama/Llama-4-Maverick-17B-128E-Instruct-FP8 | - | - | - | - | - | nao encontrado |
| meta-llama/Llama-4-Scout-17B-16E-Instruct | - | - | - | - | - | nao encontrado |
| meta-llama/Llama-3.3-70B-Instruct-Turbo | - | - | - | - | - | nao encontrado |
| NousResearch/Hermes-3-Llama-3.1-405B | - | - | - | - | - | nao encontrado |
| deepseek-ai/DeepSeek-V4-Pro | - | - | - | - | - | nao encontrado |
| deepseek-ai/DeepSeek-V4-Flash | - | - | - | - | - | nao encontrado |
| zai-org/GLM-5.1 | - | - | - | - | - | nao encontrado |
| moonshotai/Kimi-K2.6 | - | - | - | - | - | nao encontrado |
| MiniMaxAI/MiniMax-M2.5 | - | - | - | - | - | 0.5/2 |
| Qwen/Qwen3.6-35B-A3B | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3.5-397B-A17B | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3.5-122B-A10B | - | - | - | - | - | nao encontrado |
| XiaomiMiMo/MiMo-V2.5-Pro | - | - | - | - | - | nao encontrado |
| XiaomiMiMo/MiMo-V2.5 | - | - | - | - | - | nao encontrado |

### novita (19 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| deepseek/deepseek-v4-pro | 1048576 | 393216 | sim | - | - | 0.435/0.87 |
| deepseek/deepseek-v4-flash | 1048576 | 393216 | sim | - | - | 0.14/0.28 |
| deepseek/deepseek-v3.2 | 163840 | 65536 | sim | - | - | nao encontrado |
| moonshotai/kimi-k3 | 1048576 | 1048576 | sim | sim | - | 3/15 |
| moonshotai/kimi-k2.7-code | 262144 | 262144 | sim | sim | - | 0.95/4 |
| moonshotai/kimi-k2.6 | 262144 | 262144 | sim | sim | - | 1/4 |
| zai-org/glm-5.2 | 1048576 | 131072 | sim | - | - | nao encontrado |
| zai-org/glm-5.1 | 204800 | 131072 | sim | - | - | nao encontrado |
| zai-org/glm-4.7 | 204800 | 131072 | sim | - | - | 0.75/3 |
| minimax/minimax-m3 | 1000000 | 131072 | sim | sim | - | nao encontrado |
| minimax/minimax-m2.7 | 204800 | 131072 | sim | - | - | nao encontrado |
| qwen/qwen3.7-max | 1000000 | 65536 | sim | - | - | nao encontrado |
| qwen/qwen3.6-plus | 1000000 | 65536 | sim | sim | - | nao encontrado |
| qwen/qwen3.5-397b-a17b | 262144 | 65536 | sim | sim | - | nao encontrado |
| qwen/qwen3-coder-480b-a35b-instruct | 262144 | 65536 | - | - | - | nao encontrado |
| xiaomimimo/mimo-v2.5-pro | 1048576 | 131072 | sim | - | - | nao encontrado |
| openai/gpt-oss-120b | 131072 | 32768 | sim | - | - | nao encontrado |
| google/gemma-4-31b-it | 262144 | 131072 | sim | sim | - | nao encontrado |
| meta-llama/llama-3.1-8b-instruct | 16384 | 16384 | - | - | - | nao encontrado |

### siliconflow (69 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| deepseek-ai/DeepSeek-V3.2 | - | - | - | - | - | nao encontrado |
| deepseek-ai/DeepSeek-V3.2-Exp | - | - | - | - | - | nao encontrado |
| deepseek-ai/DeepSeek-V3.1 | - | - | - | - | - | nao encontrado |
| deepseek-ai/DeepSeek-V3.1-Terminus | - | - | - | - | - | nao encontrado |
| deepseek-ai/DeepSeek-V3 | - | - | - | - | - | nao encontrado |
| deepseek-ai/DeepSeek-R1 | - | - | - | - | - | nao encontrado |
| deepseek-ai/deepseek-vl2 | - | - | - | - | - | nao encontrado |
| nex-agi/DeepSeek-V3.1-Nex-N1 | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3.6-35B-A3B | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3.6-27B | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3.5-397B-A17B | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3.5-122B-A10B | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3.5-35B-A3B | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3.5-27B | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3.5-9B | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3-Next-80B-A3B-Instruct | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3-Next-80B-A3B-Thinking | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3-235B-A22B | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3-235B-A22B-Instruct-2507 | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3-235B-A22B-Thinking-2507 | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3-32B | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3-30B-A3B-Instruct-2507 | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3-30B-A3B-Thinking-2507 | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3-14B | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3-8B | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3-Coder-480B-A35B-Instruct | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3-Coder-30B-A3B-Instruct | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3-Omni-30B-A3B-Instruct | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3-Omni-30B-A3B-Thinking | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3-Omni-30B-A3B-Captioner | - | - | - | - | - | nao encontrado |
| Qwen/Qwen2.5-72B-Instruct | - | - | - | - | - | nao encontrado |
| Qwen/Qwen2.5-72B-Instruct-128K | - | - | - | - | - | nao encontrado |
| Qwen/Qwen2.5-32B-Instruct | - | - | - | - | - | nao encontrado |
| Qwen/Qwen2.5-14B-Instruct | - | - | - | - | - | nao encontrado |
| Qwen/Qwen2.5-7B-Instruct | - | - | - | - | - | nao encontrado |
| Qwen/Qwen2.5-VL-7B-Instruct | - | - | - | - | - | nao encontrado |
| zai-org/GLM-5.1 | - | - | - | - | - | nao encontrado |
| zai-org/GLM-5 | - | - | - | - | - | nao encontrado |
| zai-org/GLM-5V-Turbo | - | - | - | - | - | nao encontrado |
| zai-org/GLM-4.7 | - | - | - | - | - | nao encontrado |
| zai-org/GLM-4.6 | - | - | - | - | - | nao encontrado |
| zai-org/GLM-4.6V | - | - | - | - | - | nao encontrado |
| zai-org/GLM-4.5 | - | - | - | - | - | nao encontrado |
| zai-org/GLM-4.5-Air | - | - | - | - | - | nao encontrado |
| zai-org/GLM-4.5V | - | - | - | - | - | nao encontrado |
| THUDM/GLM-4-32B-0414 | - | - | - | - | - | nao encontrado |
| THUDM/GLM-4-9B-0414 | - | - | - | - | - | nao encontrado |
| THUDM/GLM-Z1-32B-0414 | - | - | - | - | - | nao encontrado |
| THUDM/GLM-Z1-9B-0414 | - | - | - | - | - | nao encontrado |
| moonshotai/Kimi-K2.6 | - | - | - | - | - | nao encontrado |
| moonshotai/Kimi-K2.5 | - | - | - | - | - | nao encontrado |
| moonshotai/Kimi-K2-Instruct | - | - | - | - | - | nao encontrado |
| moonshotai/Kimi-K2-Instruct-0905 | - | - | - | - | - | nao encontrado |
| moonshotai/Kimi-K2-Thinking | - | - | - | - | - | nao encontrado |
| openai/gpt-oss-120b | - | - | - | - | - | nao encontrado |
| openai/gpt-oss-20b | - | - | - | - | - | nao encontrado |
| baidu/ERNIE-4.5-300B-A47B | - | - | - | - | - | nao encontrado |
| tencent/Hunyuan-A13B-Instruct | - | - | - | - | - | nao encontrado |
| tencent/Hunyuan-MT-7B | - | - | - | - | - | nao encontrado |
| tencent/Hy3-preview | - | - | - | - | - | nao encontrado |
| meta-llama/Meta-Llama-3.1-8B-Instruct | - | - | - | - | - | nao encontrado |
| MiniMaxAI/MiniMax-M2.5 | - | - | - | - | - | 0.5/2 |
| MiniMaxAI/MiniMax-M2.1 | - | - | - | - | - | 0.5/2 |
| inclusionAI/Ling-flash-2.0 | - | - | - | - | - | nao encontrado |
| inclusionAI/Ling-mini-2.0 | - | - | - | - | - | nao encontrado |
| inclusionAI/Ring-flash-2.0 | - | - | - | - | - | nao encontrado |
| google/gemma-4-31B-it | - | - | - | - | - | nao encontrado |
| google/gemma-4-26B-A4B-it | - | - | - | - | - | nao encontrado |
| ByteDance-Seed/Seed-OSS-36B-Instruct | - | - | - | - | - | nao encontrado |

### hyperbolic (8 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| Qwen/QwQ-32B | - | - | - | - | - | nao encontrado |
| deepseek-ai/DeepSeek-R1 | - | - | - | - | - | nao encontrado |
| deepseek-ai/DeepSeek-V3 | - | - | - | - | - | nao encontrado |
| meta-llama/Llama-3.3-70B-Instruct | - | - | - | - | - | nao encontrado |
| meta-llama/Llama-3.2-3B-Instruct | - | - | - | - | - | nao encontrado |
| Qwen/Qwen2.5-72B-Instruct | - | - | - | - | - | nao encontrado |
| Qwen/Qwen2.5-Coder-32B-Instruct | - | - | - | - | - | nao encontrado |
| NousResearch/Hermes-3-Llama-3.1-70B | - | - | - | - | - | nao encontrado |

### sambanova (7 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| MiniMax-M2.7 | - | - | - | - | - | 0.5/2 |
| DeepSeek-V3.2 | - | - | - | - | - | nao encontrado |
| Llama-4-Maverick-17B-128E-Instruct | - | - | - | - | - | nao encontrado |
| Meta-Llama-3.3-70B-Instruct | - | - | - | - | - | nao encontrado |
| gpt-oss-120b | - | - | - | - | - | nao encontrado |
| DeepSeek-V3.1 | - | - | - | - | - | nao encontrado |
| gemma-4-31B-it | - | - | - | - | - | nao encontrado |

### scaleway (6 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| qwen3-235b-a22b-instruct-2507 | - | - | - | - | - | nao encontrado |
| llama-3.1-70b-instruct | - | - | - | - | - | nao encontrado |
| llama-3.1-8b-instruct | - | - | - | - | - | nao encontrado |
| mistral-small-3.2-24b-instruct-2506 | - | - | - | - | - | nao encontrado |
| deepseek-v3-0324 | - | - | - | - | - | nao encontrado |
| gpt-oss-120b | - | - | - | - | - | nao encontrado |

### upstage (2 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| solar-pro3 | - | - | - | - | - | nao encontrado |
| solar-mini | - | - | - | - | - | nao encontrado |

### venice (1 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| venice-latest | - | - | - | - | - | nao encontrado |

### reka (3 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| reka-flash-3 | - | 8192 | sim | - | - | nao encontrado |
| reka-flash | - | - | - | - | - | nao encontrado |
| reka-edge-2603 | - | - | - | - | - | nao encontrado |

### ai21 (2 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| jamba-large-1.7 | - | - | - | - | - | nao encontrado |
| jamba-mini-2 | - | - | - | - | - | nao encontrado |

### liquid (1 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| liquid-lfm-40b | - | - | - | - | - | nao encontrado |

### plamo (1 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| plamo-3.0-prime | 262144 | 20000 | - | - | - | nao encontrado |

### publicai (7 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| swiss-ai/apertus-70b-instruct | - | - | - | - | - | nao encontrado |
| swiss-ai/Apertus-8B-Instruct-2509 | - | - | - | - | - | nao encontrado |
| aisingapore/Qwen-SEA-LION-v4-32B-IT | - | - | - | - | - | nao encontrado |
| aisingapore/Gemma-SEA-LION-v4-27B-IT | - | - | - | - | - | nao encontrado |
| allenai/Olmo-3-32B-Think | - | - | - | - | - | nao encontrado |
| allenai/Olmo-3-7B-Instruct | - | - | - | - | - | nao encontrado |
| utter-project/EuroLLM-22B-Instruct-2512 | - | - | - | - | - | nao encontrado |

### sarvam (2 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| sarvam-105b | 131072 | - | - | - | - | nao encontrado |
| sarvam-30b | 65536 | - | - | - | - | nao encontrado |

### typhoon (1 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| typhoon-v2.5-30b-a3b-instruct | 131072 | - | - | - | - | nao encontrado |

### maritalk (3 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| sabia-4 | - | - | - | - | - | nao encontrado |
| sabia-4-thinking | - | - | - | - | - | nao encontrado |
| sabiazinho-4 | - | - | - | - | - | nao encontrado |

### inference-net (3 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| meta-llama/Llama-3.3-70B-Instruct | - | - | - | - | - | nao encontrado |
| deepseek-ai/DeepSeek-R1 | - | - | - | - | - | nao encontrado |
| Qwen/Qwen2.5-72B-Instruct | - | - | - | - | - | nao encontrado |

### monsterapi (2 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| meta-llama/Meta-Llama-3.1-8B-Instruct | - | - | - | - | - | nao encontrado |
| meta-llama/Llama-3.3-70B-Instruct | - | - | - | - | - | nao encontrado |

### nscale (6 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| moonshotai/Kimi-K2.5 | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3-235B-A22B-Instruct-2507 | - | - | - | - | - | nao encontrado |
| openai/gpt-oss-120b | - | - | - | - | - | nao encontrado |
| openai/gpt-oss-20b | - | - | - | - | - | nao encontrado |
| meta-llama/Llama-4-Scout-17B-16E-Instruct | - | - | - | - | - | nao encontrado |
| meta-llama/Llama-3.3-70B-Instruct | - | - | - | - | - | nao encontrado |

### runcrate (6 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| deepseek-ai/DeepSeek-V4.1-Flash | - | - | - | - | - | nao encontrado |
| deepseek-ai/DeepSeek-V4.1-Flash-1017 | - | - | - | - | - | nao encontrado |
| zai-org/GLM-5.3 | - | - | - | - | - | nao encontrado |
| zai-org/GLM-5.3-Flash | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3.8-Flash | - | - | - | - | - | nao encontrado |
| openai/gpt-oss-120b | - | - | - | - | - | nao encontrado |

### synthetic (7 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| hf:openai/gpt-oss-120b | 131072 | 65536 | sim | - | sim | nao encontrado |
| hf:zai-org/GLM-5.2 | 524288 | 65536 | sim | - | sim | nao encontrado |
| hf:moonshotai/Kimi-K2.7-Code | 262144 | 65536 | sim | sim | sim | nao encontrado |
| hf:Qwen/Qwen3.6-27B | 262144 | 65536 | sim | sim | sim | nao encontrado |
| hf:MiniMaxAI/MiniMax-M3 | 262144 | 65536 | sim | sim | sim | 0.3/1.2 |
| hf:zai-org/GLM-4.7-Flash | 196608 | 65536 | sim | - | sim | nao encontrado |
| hf:nvidia/NVIDIA-Nemotron-3-Super-120B-A12B-NVFP4 | 262144 | 65536 | sim | - | sim | nao encontrado |

### wandb (3 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| openai/gpt-oss-120b | - | - | - | - | - | nao encontrado |
| Qwen/Qwen3-Coder-480B-A35B-Instruct | - | - | - | - | - | nao encontrado |
| deepseek-ai/DeepSeek-V3.1 | - | - | - | - | - | nao encontrado |

### baseten (16 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| zai-org/GLM-5.3 | 1048576 | - | - | - | - | nao encontrado |
| zai-org/GLM-5.3-Fast | 1048576 | - | - | - | - | nao encontrado |
| zai-org/GLM-5.3-Flash | 1048576 | - | - | - | - | nao encontrado |
| zai-org/GLM-5.2 | 1048576 | - | - | - | - | nao encontrado |
| zai-org/GLM-5.2-Fast | 1048576 | - | - | - | - | nao encontrado |
| zai-org/GLM-4.7 | 200000 | - | - | - | - | nao encontrado |
| deepseek-ai/DeepSeek-V4-Pro | 1048576 | - | - | - | - | nao encontrado |
| deepseek-ai/DeepSeek-V4-Pro-0813 | 1048576 | - | - | - | - | nao encontrado |
| deepseek-ai/DeepSeek-V4-Flash-0731 | 1048576 | - | - | - | - | nao encontrado |
| moonshotai/Kimi-K3 | 1048576 | - | - | - | - | nao encontrado |
| moonshotai/Kimi-K2.6 | 262144 | - | - | - | - | nao encontrado |
| moonshotai/Kimi-K2.7-Code | 262144 | - | - | - | - | nao encontrado |
| openai/gpt-oss-120b | 131072 | - | - | - | - | nao encontrado |
| nvidia/NVIDIA-Nemotron-3-Ultra-550B-A55B | 202000 | - | - | - | - | nao encontrado |
| thinkingmachines/inkling | 1048576 | - | - | - | - | nao encontrado |
| thinkingmachines/inkling-small | 1048576 | - | - | - | - | nao encontrado |

### friendliai (2 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| meta-llama-3.1-70b-instruct | - | - | - | - | - | nao encontrado |
| meta-llama-3.1-8b-instruct | - | - | - | - | - | nao encontrado |

### lambda-ai (3 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| deepseek-r1-671b | - | - | - | - | - | nao encontrado |
| llama3.3-70b-instruct-fp8 | - | - | - | - | - | nao encontrado |
| qwen25-coder-32b-instruct | - | - | - | - | - | nao encontrado |

### featherless-ai (2 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| featherless-ai/Qwerky-72B | - | - | - | - | - | nao encontrado |
| featherless-ai/Qwerky-QwQ-32B | - | - | - | - | - | nao encontrado |

### ollama-cloud (12 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| gpt-oss:20b | - | - | sim | - | - | nao encontrado |
| gpt-oss:120b | - | - | sim | - | - | nao encontrado |
| deepseek-v4-pro | - | - | sim | - | - | 0.435/0.87 |
| deepseek-v4-flash | - | - | sim | - | - | 0.14/0.28 |
| kimi-k2.6 | - | - | - | - | - | 1/4 |
| glm-5.1 | - | - | sim | - | - | nao encontrado |
| glm-5.2 | - | - | sim | - | - | nao encontrado |
| minimax-m3 | 1048576 | - | - | sim | - | nao encontrado |
| minimax-m2.7 | - | - | - | - | - | nao encontrado |
| gemma4:31b | - | - | - | - | - | nao encontrado |
| nemotron-3-super | - | - | - | - | - | nao encontrado |
| qwen3.5:397b | - | - | - | - | - | nao encontrado |

### opencode-zen (37 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| big-pickle | - | - | sim | - | - | nao encontrado |
| kimi-k2.7-code | - | - | - | - | - | 0.95/4 |
| qwen3.5-plus | - | - | - | nao | - | nao encontrado |
| qwen3.6-plus | - | - | sim | nao | - | nao encontrado |
| gpt-5.6-sol | - | 128000 | sim | - | - | 5/30 |
| gpt-5.6-terra | - | 128000 | sim | - | - | 2.5/15 |
| gpt-5.6-luna | - | 128000 | sim | - | - | 1/6 |
| gpt-5.4 | - | - | - | - | - | nao encontrado |
| gpt-5.4-mini | - | - | - | - | - | nao encontrado |
| gpt-5.4-nano | - | - | - | - | - | nao encontrado |
| gpt-5.3-codex-spark | - | - | - | - | - | 3/12 |
| gpt-5.1 | - | - | - | - | - | 1.25/10 |
| claude-fable-5 | - | - | - | - | - | 10/50 |
| claude-opus-5 | - | - | - | - | - | nao encontrado |
| claude-sonnet-5 | - | - | - | - | - | 2/10 |
| claude-haiku-4-5 | - | - | - | - | - | nao encontrado |
| gemini-3.7-flash | - | - | - | - | - | 1.5/7.5 |
| gemini-3.5-flash-lite | - | - | - | - | - | 0.3/2.5 |
| gemini-3.1-pro | - | - | - | - | - | nao encontrado |
| gemini-3-flash | - | - | - | - | - | 0.5/3 |
| grok-build-0.1 | - | - | - | - | - | nao encontrado |
| grok-4.6 | - | - | - | - | - | nao encontrado |
| muse-spark-1.2 | 1048576 | 131072 | sim | - | - | 1.25/4.25 |
| muse-spark-1.2-contributor-free | 1048576 | 131072 | sim | - | - | nao encontrado |
| muse-spark-1.3 | 1048576 | 131072 | sim | - | - | 1.25/4.25 |
| muse-spark-1.3-contributor-free | 1048576 | 131072 | sim | - | - | nao encontrado |
| deepseek-v4-pro | - | - | sim | - | - | 0.435/0.87 |
| deepseek-v4-flash | - | - | sim | - | - | 0.14/0.28 |
| glm-5.2 | - | - | sim | - | - | nao encontrado |
| minimax-m3 | 1048576 | - | - | sim | - | nao encontrado |
| kimi-k3 | - | - | sim | - | - | 3/15 |
| deepseek-v4.1-flash-free | 330000 | 128000 | sim | - | - | nao encontrado |
| mimo-v2.6-flash-free | 1048576 | 131072 | - | - | - | nao encontrado |
| hy3-free | 200000 | - | - | - | - | nao encontrado |
| nemotron-3-ultra-free | 1000000 | - | - | - | - | nao encontrado |
| nemotron-3.5-lightning-free | - | - | - | - | - | nao encontrado |
| laguna-s-2.1-free | - | - | - | - | - | nao encontrado |

### opencode-go (69 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| glm-5.2 | - | - | sim | - | - | nao encontrado |
| glm-5.2-high | - | - | sim | - | - | nao encontrado |
| glm-5.2-max | - | - | sim | - | - | nao encontrado |
| kimi-k2.7-code | - | - | - | - | - | 0.95/4 |
| qwen3.5-plus | - | - | - | nao | - | nao encontrado |
| qwen3.6-plus | - | - | sim | nao | - | nao encontrado |
| glm-5.1 | - | - | - | - | - | nao encontrado |
| glm-5 | - | - | - | - | - | 1/4 |
| kimi-k2.6 | - | - | - | - | - | 1/4 |
| kimi-k2.5 | - | - | - | - | - | 1.2/4.8 |
| kimi-k3 | - | - | sim | - | - | 3/15 |
| kimi-k3-max | - | - | sim | - | - | nao encontrado |
| kimi-k3.1-pro | 1048576 | 131072 | sim | sim | - | 2/10 |
| mimo-v2.5-pro | - | - | sim | - | - | nao encontrado |
| mimo-v2.5 | - | - | sim | - | - | nao encontrado |
| mimo-v2.5-high | - | - | sim | - | - | nao encontrado |
| mimo-v2.5-max | - | - | sim | - | - | nao encontrado |
| mimo-v2.6-flash-free | 1048576 | 131072 | - | - | - | nao encontrado |
| minimax-m3 | 1048576 | - | - | sim | - | nao encontrado |
| minimax-m2.7 | - | - | - | - | - | nao encontrado |
| minimax-m2.5 | 204800 | 131072 | - | - | - | 0.6/2.4 |
| qwen3.7-max | - | - | sim | nao | - | nao encontrado |
| qwen3.7-max-high | - | - | sim | nao | - | nao encontrado |
| qwen3.7-max-max | - | - | sim | nao | - | nao encontrado |
| qwen3.7-plus | 1000000 | 65536 | sim | nao | - | nao encontrado |
| qwen3.7-plus-high | - | - | sim | nao | - | nao encontrado |
| qwen3.7-plus-max | - | - | sim | nao | - | nao encontrado |
| qwen3.8-max | - | - | sim | nao | - | nao encontrado |
| qwen3.6-plus-high | - | - | sim | nao | - | nao encontrado |
| qwen3.6-plus-max | - | - | sim | nao | - | nao encontrado |
| hy3 | 256000 | - | sim | - | - | nao encontrado |
| hy3-none | 256000 | - | sim | - | - | nao encontrado |
| hy3-low | 256000 | - | sim | - | - | nao encontrado |
| hy3-high | 256000 | - | sim | - | - | nao encontrado |
| hy3-preview | - | - | - | - | - | nao encontrado |
| muse-spark-1.2-contributor | 1048576 | 131072 | sim | sim | - | 0.1/0.2 |
| muse-spark-1.2-contributor-minimal | 1048576 | 131072 | sim | sim | - | nao encontrado |
| muse-spark-1.2-contributor-low | 1048576 | 131072 | sim | sim | - | nao encontrado |
| muse-spark-1.2-contributor-medium | 1048576 | 131072 | sim | sim | - | nao encontrado |
| muse-spark-1.2-contributor-high | 1048576 | 131072 | sim | sim | - | nao encontrado |
| muse-spark-1.2-contributor-xhigh | 1048576 | 131072 | sim | sim | - | nao encontrado |
| muse-spark-1.3-contributor | 1048576 | 131072 | sim | sim | - | 0.1/0.2 |
| muse-spark-1.3-contributor-minimal | 1048576 | 131072 | sim | sim | - | nao encontrado |
| muse-spark-1.3-contributor-low | 1048576 | 131072 | sim | sim | - | nao encontrado |
| muse-spark-1.3-contributor-medium | 1048576 | 131072 | sim | sim | - | nao encontrado |
| muse-spark-1.3-contributor-high | 1048576 | 131072 | sim | sim | - | nao encontrado |
| muse-spark-1.3-contributor-xhigh | 1048576 | 131072 | sim | sim | - | nao encontrado |
| muse-spark-1.3-contributor-max | 1048576 | 131072 | sim | sim | - | nao encontrado |
| grok-4.5 | - | - | sim | - | - | nao encontrado |
| grok-4.5-low | - | - | sim | - | - | nao encontrado |
| grok-4.5-medium | - | - | sim | - | - | nao encontrado |
| grok-4.5-high | - | - | sim | - | - | nao encontrado |
| grok-4.6 | 500000 | 500000 | sim | - | - | nao encontrado |
| deepseek-v4-pro | - | - | sim | - | - | 0.435/0.87 |
| deepseek-v4-flash | - | - | sim | - | - | 0.14/0.28 |
| deepseek-v4-flash-vision-exp | - | - | sim | sim | - | nao encontrado |
| gpt-5.6-luna | - | 128000 | sim | - | - | 1/6 |
| gpt-6-luna | 1050000 | 128000 | sim | - | - | 0.1/0.5 |
| deepseek-v4.1-flash | 1000000 | 384000 | - | - | - | 0.14/0.28 |
| deepseek-flash | 1000000 | 384000 | - | - | - | 0.14/0.28 |
| claude-haiku-5-5 | 1000000 | 128000 | - | - | - | nao encontrado |
| grok-4.7 | 500000 | 500000 | - | - | - | nao encontrado |
| hy4-preview | 1024000 | 64000 | - | - | - | nao encontrado |
| longcat-2.0 | 1000000 | 131072 | - | - | - | nao encontrado |
| mimo-v2.6-flash | 1048576 | 131072 | - | - | - | nao encontrado |
| mimo-v2.6-pro | 1048576 | 131072 | - | - | - | nao encontrado |
| qwen3.8-flash | 1000000 | 131072 | - | - | - | nao encontrado |
| space-bunny | 1048576 | 524288 | - | - | - | nao encontrado |
| ox-alpha-free | - | - | sim | - | - | nao encontrado |

### kilo-gateway (6 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| kilo-auto/frontier | - | - | - | - | - | nao encontrado |
| kilo-auto/balanced | - | - | - | - | - | nao encontrado |
| kilo-auto/free | - | - | - | - | - | nao encontrado |
| nvidia/nemotron-3-super-120b-a12b:free | - | - | - | - | - | nao encontrado |
| minimax/minimax-m2.5:free | - | - | - | - | - | nao encontrado |
| arcee-ai/trinity-large-preview:free | - | - | - | - | - | nao encontrado |

### kilocode (15 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| openrouter/free | - | - | - | - | - | nao encontrado |
| openai/gpt-5.6-sol | - | - | - | - | - | 5/30 |
| openai/gpt-5.6-terra | - | - | - | - | - | 2.5/15 |
| openai/gpt-5.6-luna | - | - | - | - | - | 1/6 |
| anthropic/claude-opus-5 | - | - | - | - | - | nao encontrado |
| anthropic/claude-sonnet-5 | - | - | - | - | - | 2/10 |
| anthropic/claude-haiku-4.5 | - | - | - | - | - | 0.5/2.5 |
| google/gemini-3.1-pro-preview | - | - | - | - | - | nao encontrado |
| google/gemini-3.7-flash | - | - | - | - | - | 1.5/7.5 |
| google/gemini-3.5-flash-lite | - | - | - | - | - | 0.3/2.5 |
| qwen/qwen3.8-max | - | - | - | - | - | nao encontrado |
| qwen/qwen3.7-plus | - | - | - | - | - | nao encontrado |
| deepseek/deepseek-v4-pro-0813 | - | - | sim | - | - | nao encontrado |
| deepseek/deepseek-v4-flash-0731 | - | - | sim | - | - | nao encontrado |
| moonshotai/kimi-k3 | - | - | - | - | - | 3/15 |

### vercel-ai-gateway (5 modelos)

| modelo | ctx | max out | reasoning | visao | tools | preco in/out ($/1M) |
|---|---|---|---|---|---|---|
| openai/gpt-4.1 | - | - | - | - | - | 2.5/10 |
| anthropic/claude-4-sonnet | - | - | - | - | - | nao encontrado |
| google/gemini-2.5-pro | - | - | - | - | - | 2/12 |
| moonshotai/kimi-k2 | - | - | - | - | - | 1/4 |
| vercel/v0-1.5-md | - | - | - | - | - | nao encontrado |

### llmgateway

Nenhum modelo estatico na fonte (catalogo dinamico em runtime).

### requesty

Nenhum modelo estatico na fonte (catalogo dinamico em runtime).

## B.2 Tabela canonica de precos (9Router `pricing.js`)

Fonte unica com preco por modelo nas tres bases (129 IDs). Valores em USD por 1M de tokens.

| modelo | input $/1M | output $/1M | cached $/1M |
|---|---|---|---|
| MiniMax-M2.1 | 0.5 | 2 | 0.25 |
| MiniMax-M2.5 | 0.5 | 2 | 0.25 |
| MiniMax-M2.7 | 0.5 | 2 | 0.25 |
| MiniMax-M3 | 0.3 | 1.2 | 0.06 |
| auto | 2 | 8 | 1 |
| claude-3-5-sonnet-20241022 | 3 | 15 | 1.5 |
| claude-fable-5 | 10 | 50 | 1 |
| claude-haiku-4-5-20251001 | 1 | 5 | 0.1 |
| claude-haiku-4.5 | 0.5 | 2.5 | 0.05 |
| claude-opus-4-20250514 | 15 | 25 | 7.5 |
| claude-opus-4-5-20251101 | 5 | 25 | 0.5 |
| claude-opus-4-5-thinking | 5 | 25 | 0.5 |
| claude-opus-4-6 | 5 | 25 | 0.5 |
| claude-opus-4-6-thinking | 5 | 25 | 0.5 |
| claude-opus-4.1 | 5 | 25 | 0.5 |
| claude-opus-4.5 | 5 | 25 | 0.5 |
| claude-opus-4.6 | 5 | 25 | 0.5 |
| claude-opus-5-5 | 4 | 20 | 0.2 |
| claude-opus-5-5-high | 4 | 20 | 0.2 |
| claude-opus-5-5-low | 4 | 20 | 0.2 |
| claude-opus-5-5-medium | 4 | 20 | 0.2 |
| claude-opus-5-5-thinking | 4 | 20 | 0.2 |
| claude-sonnet-4 | 3 | 15 | 0.3 |
| claude-sonnet-4-20250514 | 3 | 15 | 1.5 |
| claude-sonnet-4-5-20250929 | 3 | 15 | 0.3 |
| claude-sonnet-4-6 | 3 | 15 | 0.3 |
| claude-sonnet-4.5 | 3 | 15 | 0.3 |
| claude-sonnet-4.6 | 3 | 15 | 0.3 |
| claude-sonnet-5 | 2 | 10 | 0.2 |
| claude-sonnet-5-5 | 2 | 10 | 0.2 |
| claude-sonnet-5-5-high | 2 | 10 | 0.2 |
| claude-sonnet-5-5-low | 2 | 10 | 0.2 |
| claude-sonnet-5-5-medium | 2 | 10 | 0.2 |
| claude-sonnet-5-5-thinking | 2 | 10 | 0.2 |
| coder-model | 1.5 | 6 | 0.75 |
| deepseek-chat | 0.14 | 0.28 | 0.0028 |
| deepseek-flash | 0.14 | 0.28 | 0.0028 |
| deepseek-r1 | 0.14 | 0.28 | 0.0028 |
| deepseek-reasoner | 0.14 | 0.28 | 0.0028 |
| deepseek-v3.2-chat | 0.14 | 0.28 | 0.0028 |
| deepseek-v3.2-reasoner | 0.14 | 0.28 | 0.0028 |
| deepseek-v4-flash | 0.14 | 0.28 | 0.0028 |
| deepseek-v4-pro | 0.435 | 0.87 | 0.003625 |
| deepseek-v4.1-flash | 0.14 | 0.28 | 0.0028 |
| gemini-2.5-flash | 0.3 | 2.5 | 0.03 |
| gemini-2.5-flash-lite | 0.15 | 1.25 | 0.015 |
| gemini-2.5-pro | 2 | 12 | 0.25 |
| gemini-3-flash | 0.5 | 3 | 0.03 |
| gemini-3-flash-agent | 0.5 | 3 | 0.03 |
| gemini-3-flash-preview | 0.5 | 3 | 0.03 |
| gemini-3-pro-preview | 2 | 12 | 0.25 |
| gemini-3.1-pro-high | 4 | 18 | 0.5 |
| gemini-3.1-pro-low | 2 | 12 | 0.25 |
| gemini-3.5-flash-extra-low | 0.5 | 3 | 0.03 |
| gemini-3.5-flash-high | 0.5 | 3 | 0.03 |
| gemini-3.5-flash-lite | 0.3 | 2.5 | 0.03 |
| gemini-3.5-flash-low | 0.5 | 3 | 0.03 |
| gemini-3.6-flash | 1.5 | 7.5 | 0.15 |
| gemini-3.6-flash-high | 1.5 | 7.5 | 0.15 |
| gemini-3.6-flash-low | 1.5 | 7.5 | 0.15 |
| gemini-3.6-flash-medium | 1.5 | 7.5 | 0.15 |
| gemini-3.7-flash | 1.5 | 7.5 | 0.15 |
| gemini-3.7-flash-high | 1.5 | 7.5 | 0.15 |
| gemini-3.7-flash-low | 1.5 | 7.5 | 0.15 |
| gemini-3.7-flash-medium | 1.5 | 7.5 | 0.15 |
| gemini-3.8-flash | 1.5 | 7.5 | 0.15 |
| gemini-3.8-flash-high | 1.5 | 7.5 | 0.15 |
| gemini-3.8-flash-low | 1.5 | 7.5 | 0.15 |
| gemini-3.8-flash-medium | 1.5 | 7.5 | 0.15 |
| gemini-pro-agent | 4 | 18 | 0.5 |
| glm-4.6 | 0.5 | 2 | 0.25 |
| glm-4.6v | 0.75 | 3 | 0.375 |
| glm-4.7 | 0.75 | 3 | 0.375 |
| glm-5 | 1 | 4 | 0.5 |
| gpt-3.5-turbo | 0.5 | 1.5 | 0.25 |
| gpt-4 | 2.5 | 10 | 1.25 |
| gpt-4-turbo | 10 | 30 | 5 |
| gpt-4.1 | 2.5 | 10 | 1.25 |
| gpt-4o | 2.5 | 10 | 1.25 |
| gpt-4o-mini | 0.15 | 0.6 | 0.075 |
| gpt-5 | 1.25 | 10 | 0.625 |
| gpt-5-codex | 1.25 | 10 | 0.625 |
| gpt-5-mini | 0.25 | 2 | 0.125 |
| gpt-5.1 | 1.25 | 10 | 0.625 |
| gpt-5.1-codex | 1.25 | 10 | 0.625 |
| gpt-5.1-codex-max | 8 | 32 | 4 |
| gpt-5.1-codex-mini | 1.5 | 6 | 0.75 |
| gpt-5.1-codex-mini-high | 2 | 8 | 1 |
| gpt-5.2 | 1.75 | 14 | 0.175 |
| gpt-5.2-codex | 1.75 | 14 | 0.175 |
| gpt-5.3-codex | 1.75 | 14 | 0.175 |
| gpt-5.3-codex-spark | 3 | 12 | 0.3 |
| gpt-5.6 | 2.5 | 15 | 0.25 |
| gpt-5.6-luna | 1 | 6 | 0.1 |
| gpt-5.6-sol | 5 | 30 | 0.5 |
| gpt-5.6-terra | 2.5 | 15 | 0.25 |
| gpt-6-astra | 10 | 50 | 1 |
| gpt-6-luna | 0.1 | 0.5 | 0.01 |
| gpt-6-sol | 2 | 10 | 0.2 |
| gpt-6.1-sol | 2 | 10 | 0.1 |
| gpt-6.1-terra | 1 | 5 | 0.05 |
| gpt-oss-120b-medium | 0.5 | 2 | 0.25 |
| grok-code-fast-1 | 0.5 | 2 | 0.25 |
| k3 | 3 | 15 | 0.3 |
| kimi-for-coding | 0.95 | 4 | 0.19 |
| kimi-for-coding-highspeed | 1.9 | 8 | 0.38 |
| kimi-k2 | 1 | 4 | 0.5 |
| kimi-k2-thinking | 1.5 | 6 | 0.75 |
| kimi-k2.5 | 1.2 | 4.8 | 0.6 |
| kimi-k2.5-thinking | 1.8 | 7.2 | 0.9 |
| kimi-k2.6 | 1 | 4 | 0.5 |
| kimi-k2.7-code | 0.95 | 4 | 0.19 |
| kimi-k2.7-code-highspeed | 1.9 | 8 | 0.38 |
| kimi-k3 | 3 | 15 | 0.3 |
| kimi-k3.1-pro | 2 | 10 | 0.2 |
| kimi-latest | 1 | 4 | 0.5 |
| minimax-m2.1 | 0.5 | 2 | 0.25 |
| minimax-m2.5 | 0.6 | 2.4 | 0.3 |
| muse-spark-1.1 | 1.25 | 4.25 | 0.15 |
| muse-spark-1.2 | 1.25 | 4.25 | 0.15 |
| muse-spark-1.2-contributor | 0.1 | 0.2 | 0.002 |
| muse-spark-1.3 | 1.25 | 4.25 | 0.15 |
| muse-spark-1.3-contributor | 0.1 | 0.2 | 0.002 |
| o1 | 15 | 60 | 7.5 |
| o1-mini | 3 | 12 | 1.5 |
| oswe-vscode-prime | 1 | 4 | 0.5 |
| qwen3-coder-flash | 0.5 | 2 | 0.25 |
| qwen3-coder-plus | 1 | 4 | 0.5 |
| vision-model | 1.5 | 6 | 0.75 |

PROVIDER_PRICING keys=2 -> gh, tokenrouter
