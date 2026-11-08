# DonStop IA — console administrativo

Frontend do console em **Next.js 16 + React 19 + TypeScript**, usando os tokens do `design-system/MASTER.md`.

## Estrutura

```text
admin/
├─ app/                 # App Router: shell do console e rotas
├─ components/console/  # sidebar, topbar, cards, chips, i18n provider
├─ lib/                 # i18n, catálogo de provedores e tipos
├─ locales/             # en, pt-PT, pt-BR, es, fr, de
├─ src/                 # runtime legado do mockup estático
├─ scripts/             # validação/build legado do guia
└─ tests/               # node:test do i18n, guia e fluxo de provedores
```

## Rotas

- `/dashboard`
- `/providers` — provedores configurados
- `/providers/catalog` — provedores preconfigurados
- `/providers/[id]` — detalhe, contas e links de conexão
- `/settings`
- `/login` ? entrada com chave de escopo `manage` quando `DONTSTOP_CONSOLE_AUTH=required`

## Admin API

O frontend l? provedores do gateway quando `DONTSTOP_GATEWAY_URL` e `DONTSTOP_ADMIN_API_KEY` est?o definidos no servidor Next. A chave precisa do escopo `manage`. Sem essas vari?veis, a UI usa o cat?logo mock de `lib/providers.ts`.

```bash
cp .env.example .env.local
```

## Autentica??o do console

O login valida a chave contra a Admin API e grava um cookie `httpOnly`, `SameSite=Strict`, por 8 horas. Com `DONTSTOP_CONSOLE_AUTH=required`, o `proxy.ts` redireciona rotas sem cookie para `/login`. Sem essa vari?vel, o console mant?m o modo local atual.

## Comandos

```bash
cd admin
npm run dev        # http://127.0.0.1:3000
npm run lint
npm run typecheck
npm test
npm run build
```

O mockup estático continua em `design-system/mockups/console.html` e pode ser servido pela raiz do repositório:

```bash
python -m http.server 8765 --bind 127.0.0.1
```
