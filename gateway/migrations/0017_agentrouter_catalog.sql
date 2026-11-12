ALTER TABLE provider_catalog
    DROP CONSTRAINT IF EXISTS provider_catalog_protocol_check;

ALTER TABLE provider_catalog
    ADD CONSTRAINT provider_catalog_protocol_check
    CHECK (protocol IN ('openai', 'anthropic', 'agentrouter'));

INSERT INTO provider_catalog (
    slug, display_name, protocol, api_version, default_base_url, models_path,
    auth_type, auth_header, auth_scheme, headers, capabilities, network, quirks,
    source_ref, catalog_version, enabled
)
VALUES (
    'agentrouter',
    'AgentRouter',
    'agentrouter',
    '2023-06-01',
    'https://agentrouter.org',
    NULL,
    'api_key',
    'x-api-key',
    'raw',
    '{"anthropic-version": "2023-06-01"}'::jsonb,
    '{"stream": true, "tools": true, "vision": false, "reasoning": false, "embeddings": false, "images": false, "audio": false, "prompt_translation": true}'::jsonb,
    '{"region_field": null, "proxy_policy": "optional", "private_host": false}'::jsonb,
    '["waf_fingerprint", "requires_prompt_translation", "static_models_only"]'::jsonb,
    'allcode@agentrouter-relay+9router@0.5.112',
    '2026-11',
    true
)
ON CONFLICT (slug) DO UPDATE SET
    display_name = excluded.display_name,
    protocol = excluded.protocol,
    api_version = excluded.api_version,
    default_base_url = excluded.default_base_url,
    models_path = excluded.models_path,
    auth_type = excluded.auth_type,
    auth_header = excluded.auth_header,
    auth_scheme = excluded.auth_scheme,
    headers = excluded.headers,
    capabilities = excluded.capabilities,
    network = excluded.network,
    quirks = excluded.quirks,
    source_ref = excluded.source_ref,
    catalog_version = excluded.catalog_version,
    enabled = excluded.enabled;

INSERT INTO provider_catalog_models (
    slug, model_id, display_name, context_window, max_output_tokens,
    supports_stream, supports_tools, supports_vision, supports_reasoning,
    input_price_per_million, output_price_per_million, pricing_source
)
VALUES
    ('agentrouter', 'gpt-5.6-sol', 'GPT 5.6 Sol', 200000, 8192, true, true, false, false, 0, 0, 'unverified'),
    ('agentrouter', 'deepseek-v4-flash', 'DeepSeek V4 Flash', 200000, 8192, true, true, false, false, 0, 0, 'unverified'),
    ('agentrouter', 'glm-5.3', 'GLM 5.3', 200000, 8192, true, true, false, false, 0, 0, 'unverified'),
    ('agentrouter', 'claude-opus-4-8', 'Claude Opus 4.8', 200000, 8192, true, true, false, false, 0, 0, 'unverified'),
    ('agentrouter', 'claude-opus-5', 'Claude Opus 5', 200000, 8192, true, true, false, false, 0, 0, 'unverified')
ON CONFLICT (slug, model_id) DO UPDATE SET
    display_name = excluded.display_name,
    context_window = excluded.context_window,
    max_output_tokens = excluded.max_output_tokens,
    supports_stream = excluded.supports_stream,
    supports_tools = excluded.supports_tools,
    supports_vision = excluded.supports_vision,
    supports_reasoning = excluded.supports_reasoning,
    input_price_per_million = excluded.input_price_per_million,
    output_price_per_million = excluded.output_price_per_million,
    pricing_source = excluded.pricing_source;
