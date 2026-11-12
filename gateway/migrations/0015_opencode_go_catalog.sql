INSERT INTO provider_catalog (
    slug, display_name, protocol, api_version, default_base_url, models_path,
    auth_type, auth_header, auth_scheme, headers, capabilities, network, quirks,
    source_ref, catalog_version, enabled
)
VALUES (
    'opencode-go',
    'OpenCode Go',
    'openai',
    NULL,
    'https://opencode.ai/inference/go/openai/v1',
    '/models',
    'api_key',
    'Authorization',
    'bearer',
    '{}'::jsonb,
    '{"stream": true, "tools": true, "vision": true, "reasoning": true, "embeddings": false, "images": false, "audio": false}'::jsonb,
    '{"region_field": null, "proxy_policy": "optional", "private_host": false}'::jsonb,
    '["requires_x_opencode_session"]'::jsonb,
    'opencode@2026-11',
    '2026-11',
    true
)
ON CONFLICT (slug) DO UPDATE SET
    display_name = excluded.display_name,
    protocol = excluded.protocol,
    default_base_url = excluded.default_base_url,
    models_path = excluded.models_path,
    auth_type = excluded.auth_type,
    auth_header = excluded.auth_header,
    auth_scheme = excluded.auth_scheme,
    capabilities = excluded.capabilities,
    network = excluded.network,
    quirks = excluded.quirks,
    source_ref = excluded.source_ref,
    catalog_version = excluded.catalog_version,
    enabled = excluded.enabled;
