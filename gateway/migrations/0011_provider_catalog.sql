CREATE TABLE provider_catalog (
    slug text PRIMARY KEY,
    display_name text NOT NULL,
    protocol text NOT NULL,
    api_version text,
    default_base_url text NOT NULL,
    models_path text,
    auth_type text NOT NULL,
    auth_header text NOT NULL,
    auth_scheme text NOT NULL,
    headers jsonb NOT NULL DEFAULT '{}'::jsonb,
    capabilities jsonb NOT NULL DEFAULT '{}'::jsonb,
    network jsonb NOT NULL DEFAULT '{}'::jsonb,
    quirks jsonb NOT NULL DEFAULT '[]'::jsonb,
    source_ref text NOT NULL DEFAULT '',
    catalog_version text NOT NULL,
    enabled boolean NOT NULL DEFAULT true,
    CONSTRAINT provider_catalog_protocol_check CHECK (protocol IN ('openai', 'anthropic'))
);

CREATE TABLE provider_catalog_models (
    slug text NOT NULL REFERENCES provider_catalog (slug) ON DELETE CASCADE,
    model_id text NOT NULL,
    display_name text NOT NULL DEFAULT '',
    context_window integer,
    max_output_tokens integer,
    supports_stream boolean NOT NULL DEFAULT true,
    supports_tools boolean NOT NULL DEFAULT false,
    supports_vision boolean NOT NULL DEFAULT false,
    supports_reasoning boolean NOT NULL DEFAULT false,
    input_price_per_million numeric(14, 8) NOT NULL DEFAULT 0,
    output_price_per_million numeric(14, 8) NOT NULL DEFAULT 0,
    cached_price_per_million numeric(14, 8) NOT NULL DEFAULT 0,
    pricing_source text NOT NULL DEFAULT 'unverified',
    PRIMARY KEY (slug, model_id)
);

ALTER TABLE providers
    ADD COLUMN IF NOT EXISTS catalog_slug text;

CREATE INDEX providers_catalog_slug_idx
    ON providers (tenant_id, catalog_slug);

INSERT INTO provider_catalog (slug, display_name, protocol, api_version, default_base_url, models_path, auth_type, auth_header, auth_scheme, headers, capabilities, network, quirks, source_ref, catalog_version)
VALUES
    ('openai', 'OpenAI', 'openai', NULL, 'https://api.openai.com/v1', '/models', 'api_key', 'Authorization', 'bearer',
     '{}'::jsonb,
     '{"stream": true, "tools": true, "vision": true, "reasoning": true, "embeddings": false, "images": false, "audio": false}'::jsonb,
     '{"region_field": null, "proxy_policy": "optional", "private_host": false}'::jsonb,
     '["supports_responses_api"]'::jsonb, 'omniroute@3.8.53+9router@0.5.112+allcode@1.27.6', '2026-11'),
    ('anthropic', 'Anthropic', 'anthropic', '2023-06-01', 'https://api.anthropic.com', '/v1/models', 'api_key', 'x-api-key', 'raw',
     '{"anthropic-version": "2023-06-01"}'::jsonb,
     '{"stream": true, "tools": true, "vision": true, "reasoning": true, "embeddings": false, "images": false, "audio": false}'::jsonb,
     '{"region_field": null, "proxy_policy": "optional", "private_host": false}'::jsonb,
     '["requires_x_api_key_header", "rejects_temperature_on_opus_4_7_plus"]'::jsonb, 'omniroute@3.8.53+9router@0.5.112+allcode@1.27.6', '2026-11'),
    ('alibaba', 'Alibaba Qwen (DashScope)', 'openai', NULL, 'https://dashscope-intl.aliyuncs.com/compatible-mode/v1', '/models', 'api_key', 'Authorization', 'bearer',
     '{}'::jsonb,
     '{"stream": true, "tools": true, "vision": true, "reasoning": true, "embeddings": false, "images": false, "audio": false}'::jsonb,
     '{"region_field": null, "proxy_policy": "optional", "private_host": false}'::jsonb,
     '["dashscope_compatible_mode", "cn_region_variant_exists"]'::jsonb, 'omniroute@3.8.53+9router@0.5.112+allcode@1.27.6', '2026-11'),
    ('cerebras', 'Cerebras', 'openai', NULL, 'https://api.cerebras.ai/v1', '/models', 'api_key', 'Authorization', 'bearer',
     '{}'::jsonb,
     '{"stream": true, "tools": true, "vision": false, "reasoning": true, "embeddings": false, "images": false, "audio": false}'::jsonb,
     '{"region_field": null, "proxy_policy": "optional", "private_host": false}'::jsonb,
     '["third_party_integration_header_optional"]'::jsonb, 'omniroute@3.8.53+9router@0.5.112+allcode@1.27.6', '2026-11'),
    ('cohere', 'Cohere', 'openai', NULL, 'https://api.cohere.com/compatibility/v1', '/models', 'api_key', 'Authorization', 'bearer',
     '{}'::jsonb,
     '{"stream": true, "tools": true, "vision": true, "reasoning": true, "embeddings": false, "images": false, "audio": false}'::jsonb,
     '{"region_field": null, "proxy_policy": "optional", "private_host": false}'::jsonb,
     '["compatibility_path", "9router_uses_api_cohere_ai"]'::jsonb, 'omniroute@3.8.53+9router@0.5.112+allcode@1.27.6', '2026-11'),
    ('groq', 'Groq', 'openai', NULL, 'https://api.groq.com/openai/v1', '/models', 'api_key', 'Authorization', 'bearer',
     '{}'::jsonb,
     '{"stream": true, "tools": true, "vision": false, "reasoning": true, "embeddings": false, "images": false, "audio": false}'::jsonb,
     '{"region_field": null, "proxy_policy": "optional", "private_host": false}'::jsonb,
     '[]'::jsonb, 'omniroute@3.8.53+9router@0.5.112+allcode@1.27.6', '2026-11'),
    ('mistral', 'Mistral', 'openai', NULL, 'https://api.mistral.ai/v1', '/models', 'api_key', 'Authorization', 'bearer',
     '{}'::jsonb,
     '{"stream": true, "tools": true, "vision": true, "reasoning": false, "embeddings": false, "images": false, "audio": false}'::jsonb,
     '{"region_field": null, "proxy_policy": "optional", "private_host": false}'::jsonb,
     '[]'::jsonb, 'omniroute@3.8.53+9router@0.5.112+allcode@1.27.6', '2026-11'),
    ('nvidia', 'NVIDIA NIM', 'openai', NULL, 'https://integrate.api.nvidia.com/v1', '/models', 'api_key', 'Authorization', 'bearer',
     '{}'::jsonb,
     '{"stream": true, "tools": true, "vision": false, "reasoning": true, "embeddings": false, "images": false, "audio": false}'::jsonb,
     '{"region_field": null, "proxy_policy": "optional", "private_host": false}'::jsonb,
     '["billing_origin_header_optional"]'::jsonb, 'omniroute@3.8.53+9router@0.5.112+allcode@1.27.6', '2026-11'),
    ('openrouter', 'OpenRouter', 'openai', NULL, 'https://openrouter.ai/api/v1', '/models', 'api_key', 'Authorization', 'bearer',
     '{}'::jsonb,
     '{"stream": true, "tools": true, "vision": true, "reasoning": true, "embeddings": false, "images": false, "audio": false}'::jsonb,
     '{"region_field": null, "proxy_policy": "optional", "private_host": false}'::jsonb,
     '["dynamic_catalog", "optional_http_referer_and_x_title"]'::jsonb, 'omniroute@3.8.53+9router@0.5.112+allcode@1.27.6', '2026-11'),
    ('perplexity', 'Perplexity', 'openai', NULL, 'https://api.perplexity.ai', NULL, 'api_key', 'Authorization', 'bearer',
     '{}'::jsonb,
     '{"stream": true, "tools": false, "vision": false, "reasoning": true, "embeddings": false, "images": false, "audio": false}'::jsonb,
     '{"region_field": null, "proxy_policy": "optional", "private_host": false}'::jsonb,
     '["no_models_endpoint", "search_grounded_models"]'::jsonb, 'omniroute@3.8.53+9router@0.5.112+allcode@1.27.6', '2026-11'),
    ('together', 'Together AI', 'openai', NULL, 'https://api.together.xyz/v1', '/models', 'api_key', 'Authorization', 'bearer',
     '{}'::jsonb,
     '{"stream": true, "tools": true, "vision": false, "reasoning": true, "embeddings": false, "images": false, "audio": false}'::jsonb,
     '{"region_field": null, "proxy_policy": "optional", "private_host": false}'::jsonb,
     '["open_weights_focus"]'::jsonb, 'omniroute@3.8.53+9router@0.5.112+allcode@1.27.6', '2026-11'),
    ('venice', 'Venice AI', 'openai', NULL, 'https://api.venice.ai/api/v1', '/models', 'api_key', 'Authorization', 'bearer',
     '{}'::jsonb,
     '{"stream": true, "tools": true, "vision": true, "reasoning": false, "embeddings": false, "images": false, "audio": false}'::jsonb,
     '{"region_field": null, "proxy_policy": "optional", "private_host": false}'::jsonb,
     '["uncensored_models_available"]'::jsonb, 'omniroute@3.8.53+9router@0.5.112+allcode@1.27.6', '2026-11'),
    ('vercel-ai-gateway', 'Vercel AI Gateway', 'openai', NULL, 'https://ai-gateway.vercel.sh/v1', '/models', 'api_key', 'Authorization', 'bearer',
     '{}'::jsonb,
     '{"stream": true, "tools": true, "vision": true, "reasoning": true, "embeddings": false, "images": false, "audio": false}'::jsonb,
     '{"region_field": null, "proxy_policy": "optional", "private_host": false}'::jsonb,
     '["namespaced_model_ids"]'::jsonb, 'omniroute@3.8.53+9router@0.5.112+allcode@1.27.6', '2026-11'),
    ('xai', 'xAI', 'openai', NULL, 'https://api.x.ai/v1', '/models', 'api_key', 'Authorization', 'bearer',
     '{}'::jsonb,
     '{"stream": true, "tools": true, "vision": true, "reasoning": true, "embeddings": false, "images": false, "audio": false}'::jsonb,
     '{"region_field": null, "proxy_policy": "optional", "private_host": false}'::jsonb,
     '["openai_chat_here_responses_in_allcode"]'::jsonb, 'omniroute@3.8.53+9router@0.5.112+allcode@1.27.6', '2026-11')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO provider_catalog_models (slug, model_id, display_name, context_window, max_output_tokens, supports_tools, supports_vision, supports_reasoning, input_price_per_million, output_price_per_million, pricing_source)
VALUES
    ('openai', 'gpt-4o', 'GPT-4o', 128000, 16384, true, true, false, 2.5, 10, '9router:pricing.js@0.5.112'),
    ('openai', 'gpt-4o-mini', 'GPT-4o Mini', 128000, 16384, true, true, false, 0.15, 0.6, '9router:pricing.js@0.5.112'),
    ('openai', 'gpt-4.1', 'GPT-4.1', 1047576, 32768, true, true, false, 2.5, 10, '9router:pricing.js@0.5.112'),
    ('openai', 'gpt-4.1-mini', 'GPT-4.1 Mini', 1047576, 32768, true, true, false, 0, 0, 'unverified'),

    ('anthropic', 'claude-sonnet-4-20250514', 'Claude Sonnet 4', 200000, 64000, true, true, true, 3, 15, '9router:pricing.js@0.5.112'),
    ('anthropic', 'claude-opus-4-20250514', 'Claude Opus 4', 200000, 64000, true, true, true, 15, 25, '9router:pricing.js@0.5.112'),
    ('anthropic', 'claude-3-5-sonnet-20241022', 'Claude 3.5 Sonnet', 200000, 64000, true, true, true, 3, 15, '9router:pricing.js@0.5.112'),

    ('alibaba', 'qwen3.5-plus', 'Qwen3.5 Plus', 1000000, 65536, true, true, true, 0.5, 2, '9router:pricing.js@0.5.112'),
    ('alibaba', 'qwen3-coder-plus', 'Qwen3 Coder Plus', 1000000, 64000, true, false, true, 1, 4, '9router:pricing.js@0.5.112'),
    ('alibaba', 'glm-5', 'GLM-5', 200000, 128000, true, false, true, 1, 4, '9router:pricing.js@0.5.112'),

    ('cerebras', 'gpt-oss-120b', 'GPT-OSS 120B', 131072, 32768, true, false, true, 0, 0, 'unverified'),
    ('cerebras', 'llama-3.3-70b', 'Llama 3.3 70B', 131072, 32768, true, false, false, 0, 0, 'unverified'),
    ('cerebras', 'qwen-3-32b', 'Qwen3 32B', 262144, 64000, true, false, true, 0.5, 2, '9router:pricing.js@0.5.112'),

    ('cohere', 'command-a-03-2025', 'Command A', 256000, 8192, true, false, true, 0, 0, 'unverified'),
    ('cohere', 'command-r-plus-08-2024', 'Command R+', 128000, 4096, true, false, false, 0, 0, 'unverified'),

    ('groq', 'llama-3.3-70b-versatile', 'Llama 3.3 70B Versatile', 131072, 32768, true, false, false, 0, 0, 'unverified'),
    ('groq', 'openai/gpt-oss-120b', 'GPT-OSS 120B', 131072, 32768, true, false, true, 0, 0, 'unverified'),
    ('groq', 'qwen/qwen3-32b', 'Qwen3 32B', 131072, 32768, true, false, true, 0, 0, 'unverified'),

    ('mistral', 'mistral-large-latest', 'Mistral Large', 262144, 32768, true, false, false, 0, 0, 'unverified'),
    ('mistral', 'mistral-small-latest', 'Mistral Small', 262144, 32768, true, true, false, 0, 0, 'unverified'),
    ('mistral', 'codestral-latest', 'Codestral', 262144, 32768, true, false, false, 0, 0, 'unverified'),

    ('nvidia', 'deepseek-ai/deepseek-v4-pro', 'DeepSeek V4 Pro', 1000000, 384000, true, false, true, 0, 0, 'unverified'),
    ('nvidia', 'z-ai/glm-5.2', 'GLM 5.2', 1000000, 131072, true, false, true, 1, 4, '9router:pricing.js@0.5.112'),
    ('nvidia', 'minimaxai/minimax-m3', 'MiniMax M3', 200000, 131072, true, false, true, 0.3, 1.2, '9router:pricing.js@0.5.112'),

    ('openrouter', 'auto', 'OpenRouter Auto', 200000, 64000, true, true, true, 2, 8, '9router:pricing.js@0.5.112'),

    ('perplexity', 'sonar-pro', 'Sonar Pro', 200000, 8192, false, false, true, 0, 0, 'unverified'),
    ('perplexity', 'sonar', 'Sonar', 128000, 8192, false, false, true, 0, 0, 'unverified'),

    ('together', 'meta-llama/Llama-3.3-70B-Instruct-Turbo', 'Llama 3.3 70B Turbo', 131072, 8192, true, false, false, 0, 0, 'unverified'),
    ('together', 'deepseek-ai/DeepSeek-R1', 'DeepSeek R1', 131072, 8192, false, false, true, 0, 0, 'unverified'),
    ('together', 'Qwen/Qwen3-235B-A22B', 'Qwen3 235B A22B', 262144, 32768, true, false, true, 0, 0, 'unverified'),

    ('venice', 'venice-latest', 'Venice Latest', 128000, 8192, true, true, false, 0, 0, 'unverified'),

    ('vercel-ai-gateway', 'openai/gpt-4.1', 'GPT-4.1 (gateway)', 1047576, 32768, true, true, false, 2.5, 10, '9router:pricing.js@0.5.112'),
    ('vercel-ai-gateway', 'anthropic/claude-4-sonnet', 'Claude Sonnet 4 (gateway)', 200000, 64000, true, true, true, 3, 15, '9router:pricing.js@0.5.112'),
    ('vercel-ai-gateway', 'google/gemini-2.5-pro', 'Gemini 2.5 Pro (gateway)', 1048576, 65536, true, true, true, 0, 0, 'unverified'),

    ('xai', 'grok-4.5', 'Grok 4.5', 256000, 64000, true, true, true, 0, 0, 'unverified'),
    ('xai', 'grok-4', 'Grok 4', 256000, 64000, true, true, true, 0, 0, 'unverified')
ON CONFLICT (slug, model_id) DO NOTHING;
