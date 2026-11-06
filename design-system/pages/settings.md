# Página — Configurações

> Override do [MASTER](../MASTER.md). Preferência de idioma, persistência e acesso ao guia.

## Objetivo

Permitir que o usuário troque o idioma da interface a qualquer momento, veja onde a preferência é armazenada e abra o guia da página.

## Wireframe

```
┌──────────┬──────────────────────────────────────────────────────────────────────┐
│ Sidebar  │ Configurações                                        [Ajuda]         │
│          ├──────────────────────────────────────────────────────────────────────┤
│ ...      │ ┌──────────────────────────────────────────────────────────────┐       │
│ Config.  │ │ Idioma                                                       │       │
│          │ │ Escolha o idioma da interface.                               │       │
│          │ │ ( ) English  ( ) Português (PT)  ( ) Português (BR)          │       │
│          │ │ ( ) Español  (•) Français        ( ) Deutsch                 │       │
│          │ └──────────────────────────────────────────────────────────────┘       │
│          │ ┌──────────────────────────────────────────────────────────────┐       │
│          │ │ Preferência                                                  │       │
│          │ │ A mudança é aplicada imediatamente e salva neste navegador.  │       │
│          │ │ Armazenado como dontstop.locale.                             │       │
│          │ │ Traduções ausentes usam o inglês como fallback.              │       │
│          │ │ [Abrir guia do usuário]                                      │       │
│          │ └──────────────────────────────────────────────────────────────┘       │
└──────────┴──────────────────────────────────────────────────────────────────────┘
```

## Componentes

1. **Grade de idiomas**: `fieldset` + `legend`; cartões com `input[type=radio]`; nomes nativos; seleção atual destacada.
2. **Preferência**: texto sobre aplicação imediata, chave `dontstop.locale` e fallback para inglês.
3. **Ajuda**: link contextual para `docs/user-guide/<locale>/settings.md`, com fallback para inglês e índice.
4. **Onboarding** (primeiro acesso): mesmo componente de grade, com idioma detectado pré-selecionado e botão **Continuar**; sem fechar por `Esc` até confirmar.

## Comportamento

- Trocar o idioma chama `setLocale`, atualiza `<html lang>`, re-renderiza textos, `aria-label`, `title`, placeholders e o link de ajuda **sem reload**.
- A escolha é salva em `localStorage` (`dontstop.locale`, string BCP-47) e usada nas próximas visitas.
- O onboarding não reaparece quando a chave existe.
- O layout permanece estável em `de`/`fr`; a grade vira uma coluna abaixo de 900 px.

## Acessibilidade

- Foco visível nos cartões (`:focus-within`).
- Região `aria-live="polite"` anuncia a mudança de idioma.
- `aria-label`/`title` do link de ajuda traduzidos.
- `fieldset`/`legend` e navegação por teclado.

## Lazy loading

- A página não carrega dados de backend; apenas preferência local e catálogo de locales já usado pelo console.
