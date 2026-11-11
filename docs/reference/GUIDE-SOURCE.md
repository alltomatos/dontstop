# Origem do guia do usuário (local × GitHub)

> Decisão registrada em [`docs/05-ADRS.md`](../05-ADRS.md) (ADR-020).
> Complementa [`docs/user-guide/README.md`](../user-guide/README.md).

## 1. Problema

O guia era renderizado para `admin/public/guide/**` por `scripts/build-guide-html.js` e embutido na
imagem do admin. Qualquer ajuste de texto exigia rebuild e redeploy do container.

## 2. Decisão

O console passa a servir o guia por uma rota própria, `GET /guide/{locale}/{slug}`, que escolhe a
origem em runtime:

| `DONTSTOP_GUIDE_SOURCE` | Comportamento |
|---|---|
| `local` (padrão) | Lê o HTML embutido em `admin/public/guide/<locale>/<slug>.html`. É o modo de desenvolvimento/local. |
| `github` | Busca o Markdown em `raw.githubusercontent.com` no `ref` configurado, renderiza no servidor e guarda em cache curto. |

A URL pública não muda entre os modos, então links, âncoras (`#openrouter`), imagens e a abertura em
nova aba com `rel="noopener noreferrer"` continuam iguais. O `proxy.ts` mantém `guide` fora da
autenticação.

A rota é **server-side**: o navegador nunca fala com o GitHub, nenhum token é necessário e não há
CORS. O conteúdo do GitHub é público; nenhum segredo entra no HTML.

## 3. Configuração

| Variável | Padrão | Uso |
|---|---|---|
| `DONTSTOP_GUIDE_SOURCE` | `local` | `github` habilita a busca remota |
| `DONTSTOP_GUIDE_REPO` | `alltomatos/dontstop` | `owner/repo` |
| `DONTSTOP_GUIDE_REF` | `main` | branch, tag ou commit |
| `DONTSTOP_GUIDE_BASE_URL` | `https://raw.githubusercontent.com` | base do conteúdo cru (permite espelho interno) |
| `DONTSTOP_GUIDE_CACHE_TTL_SECONDS` | `60` | TTL do cache em memória |
| `DONTSTOP_GUIDE_TIMEOUT_MS` | `4000` | timeout do fetch |

`deploy/stack-vps.yml` define `DONTSTOP_GUIDE_SOURCE=github`. `docker-compose.yml` e `.env.example`
usam `local` por padrão e permitem sobrescrever por ambiente.

## 4. Fluxo e fallback

```
/guide/{locale}/{slug}
   ├─ sanitiza locale (6 suportados) e slug ([a-z0-9-])
   ├─ github: tenta locale → en
   │     ├─ 200 → renderiza (imagens absolutas no raw; links internos → /guide/...)
   │     ├─ 304 → reaproveita cache e estende TTL
   │     ├─ 404 → próximo locale
   │     └─ erro/timeout → usa cache expirado, se existir
   ├─ fallback: HTML local embutido (locale → en)
   └─ último recurso: página amigável "Guia indisponível" (HTTP 200)
```

Nunca há `404` nem tela quebrada: qualquer falha termina em conteúdo local ou em uma página de aviso
com status 200.

## 5. Cache e atualização

- Cache em memória por processo, chave `repo@ref:locale:slug`, TTL curto (60 s por padrão).
- Requisições condicionais (`If-None-Match`) aproveitam `304` e reduzem tráfego.
- `?refresh=1` força a atualização, com cooldown de 10 s por chave para evitar abuso.
- Respostas enviam `X-Dontstop-Guide-Source` (`github`, `github-cache`, `github-stale`,
  `local`, `local-fallback`, `unavailable`) e `X-Dontstop-Guide-Locale`.
- `Cache-Control: public, max-age=0, s-maxage=<ttl>, stale-while-revalidate=<2×ttl>`.

Como o cache é por processo, cada réplica pode ter até `ttl` de atraso — aceitável para documentação.

## 6. Imagens e links

- No modo GitHub, `../assets/<locale>/x.png` vira
  `https://raw.githubusercontent.com/<repo>/<ref>/docs/user-guide/assets/<locale>/x.png`.
- Links entre guias (`dashboard.md#secao`) viram `/guide/<locale>/dashboard#secao` e continuam
  passando pela rota com fallback.
- No modo local, as imagens continuam vindo de `/guide/assets/...` (cópia embutida).

## 7. Operação

- Atualizou o guia no `main`? Basta recarregar: em até `ttl` segundos (ou com `?refresh=1`) a mudança
  aparece para todos, sem rebuild.
- GitHub indisponível? O guia continua abrindo pela cópia local até o próximo sucesso.
- Rodar `npm run guide:build` (admin) continua necessário apenas para atualizar a cópia embutida usada
  em `local` e no fallback de produção. O fluxo de captura de screenshots não mudou.

## 8. Segurança e limites

- Apenas conteúdo público do repositório; nenhum token é lido nem enviado.
- Nenhuma variável `DONTSTOP_GUIDE_*` é exposta ao cliente (`NEXT_PUBLIC_*` não é usada).
- Slug é validado contra path traversal antes de qualquer leitura de arquivo.
- Limite de taxa do GitHub é respeitado por cache + requisições condicionais.
