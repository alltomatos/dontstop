-- ADR-024: contas novas herdam o default de proxy do provedor.
-- Apenas o default muda; linhas existentes preservam o modo atual.
ALTER TABLE provider_accounts
    ALTER COLUMN proxy_mode SET DEFAULT 'inherit';
