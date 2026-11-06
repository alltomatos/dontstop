# QA — P0-α E1.12

> Gate de credenciais de provider e resolução de endpoint.
> Data: 2026-11-06. Issue: #19.

## Escopo testado

- AES-256-GCM roundtrip para credencial de provider.
- Chave mestra de 32 bytes (base64) e rejeição de tamanho inválido.
- Detecção de ciphertext adulterado e chave errada.
- Migração `base_url`/`credential_key_version`.
- `Candidate -> provider.Account` com API key apenas em memória.
- CLI `encrypt-secret` sem imprimir plaintext.
- Ausência de plaintext em erros/logs de teste.

## Fora de escopo

- KMS/Vault e rotação multi-versão de chave (P1).
- `/v1/chat/completions` e streaming (E1.5).
- Audit log persistido da leitura de credencial.
- Allowlist de upstream/SSRF (hardening).

## Casos executados

| ID | Caso | Resultado |
|---|---|---|
| QA-31 | `Encrypt`/`Decrypt` roundtrip | PASS |
| QA-32 | Chave errada falha | PASS |
| QA-33 | Ciphertext adulterado falha | PASS |
| QA-34 | Chave com tamanho inválido falha | PASS |
| QA-35 | `MASTER_KEY` é carregada da configuração | PASS |
| QA-36 | Migração adiciona `base_url` e `credential_key_version` | PASS |
| QA-37 | Repositório carrega endpoint e ciphertext/versão | PASS |
| QA-38 | Resolver decripta credencial e preserva `base_url` | PASS |
| QA-39 | Falha de decrypt não vaza plaintext | PASS |
| QA-40 | CLI emite apenas ciphertext base64 | PASS |
| QA-41 | `go test -race -p 1 ./...` com Postgres/Redis | PASS |
| QA-42 | `golangci-lint`, `go vet`, `go build` | PASS |
| QA-43 | `govulncheck` com Go 1.26.10 | PASS, 0 vulnerabilidades |

## Auditoria de segurança

- **Criptografia:** AES-256-GCM, nonce aleatório de 12 bytes, tag autenticada e envelope versionado.
- **Chave:** 32 bytes; em produção deve vir de Docker secret/KMS, nunca de arquivo versionado.
- **Plaintext:** só existe em memória durante a chamada; erros não incluem o valor.
- **Integridade:** tamper e chave errada retornam erro sem detalhes sensíveis.
- **Rotação:** `credential_key_version` preparado para rotação; decriptação usa a chave mestra atual.

## Riscos residuais

- Uma única chave mestra no P0; rotação/rewrap e KMS ficam para P1.
- CLI local pode deixar plaintext no histórico se usado incorretamente; usar stdin e ambiente controlado.
- Sem audit event para leitura/decrypt de credencial nesta slice.
