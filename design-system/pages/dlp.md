# Página — DLP e Políticas

> Override do [MASTER](../MASTER.md). Foco: governança, redaction, bloqueio e auditoria sem expor PII.

## Objetivo

Configurar regras de DLP, testar redaction/bloqueio e auditar decisões por tenant, rota e modelo.

## Wireframe (desktop ≥ 1024px)

```
┌──────────┬──────────────────────────────────────────────────────────────────────┐
│ Sidebar  │ Título: DLP e Políticas                    [+ Nova regra] [Testar]   │
│          ├──────────────────────────────────────────────────────────────────────┤
│ Dashboard│ Abas: [Regras] [Políticas] [Eventos] [Auditoria]                      │
│ Providers│ ┌──────────────────────────────────────────────────────────────┐       │
│ Cache    │ │ Regra              Tipo      Ação     Severidade  Status     │       │
│ DLP      │ │ CPF/CNPJ           PII       Redact   Alta        ● Ativa    │       │
│ Custo    │ │ Chaves AWS         Secret    Block    Crítica     ● Ativa    │       │
│ Observ.  │ │ Prompt injection   Injection Audit    Média       ◐ Audit    │       │
│          │ └──────────────────────────────────────────────────────────────┘       │
│          │ ┌──────────────────── Teste de regra ──────────────────────────┐       │
│          │ │ Texto: [Cole um prompt.......]                               │       │
│          │ │ Resultado: 2 achados (CPF, e-mail) → ação: REDACT            │       │
│          │ │ Prévia: "Meu CPF é ***.***.***-** e e-mail ***@***.com"      │       │
│          │ │ [Executar teste] [Salvar como caso de teste]                 │       │
│          │ └──────────────────────────────────────────────────────────────┘       │
│          │ ┌──────────────────────────────────────────────────────────────┐       │
│          │ │ Eventos recentes (sem PII)                                   │       │
│          │ │ 10:45 block  AWS key  tenant A  rota openai  request 8f2a     │       │
│          │ │ 10:44 redact CPF      tenant A  rota claude  request 7c1b     │       │
│          │ └──────────────────────────────────────────────────────────────┘       │
└──────────┴──────────────────────────────────────────────────────────────────────┘
```

## Componentes

1. **Lista de regras** com tipo, ação, severidade, status e último hit.
2. **Editor de regra**: detector, padrão, ação (`allow/redact/block/audit`), escopo e exceções.
3. **Teste de regra** com prévia redigida e destaque dos spans detectados.
4. **Eventos** com request-id, tenant, rota, detector, ação e hash — nunca o dado bruto.
5. **Auditoria** com filtros por período/ator/ação e exportação.
6. **Políticas** com prioridade, escopo e simulação (“o que aconteceria”).

## Dados e APIs

- `GET/POST/PATCH /admin/v1/dlp/rules`
- `POST /admin/v1/dlp/test`
- `GET /admin/v1/dlp/events`
- `GET /admin/v1/audit/events`
- `GET/PATCH /admin/v1/policies`

## Estados

- **Loading:** skeleton na lista; teste com spinner.
- **Vazio:** “Nenhuma regra ativa” + “Criar regra”.
- **Erro:** falha do detector mostra “degradado” e fallback de regex.
- **Perigo:** criar regra `block` exige revisão e confirmação.

## Lazy loading

- Editor de regras e testador em chunks separados.
- Eventos paginados/virtualizados.
- Exportação de auditoria assíncrona com job e notificação.

## Segurança e a11y

- Nunca renderizar PII bruta; incluso no testador, mascarar por padrão.
- Ações destrutivas com confirmação; `block` com aviso de impacto.
- Cores + ícones + texto para severidade.
- Labels, foco e leitura de tabela acessíveis.
