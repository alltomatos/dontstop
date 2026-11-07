CREATE TABLE mcp_settings (
    tenant_id uuid PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    enabled boolean NOT NULL DEFAULT false,
    transport text NOT NULL DEFAULT 'stdio',
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT mcp_settings_transport_valid CHECK (transport IN ('stdio', 'sse', 'streamable-http'))
);

CREATE TABLE mcp_tool_settings (
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    tool_id text NOT NULL,
    enabled boolean NOT NULL DEFAULT true,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, tool_id)
);
