-- ADR-025: catálogo AgentRouter atualizado com os modelos do console.
-- Disponibilidade é dinâmica (402/503) e não é codificada aqui.
INSERT INTO provider_catalog_models (
    slug, model_id, display_name, context_window, max_output_tokens,
    supports_stream, supports_tools, supports_vision, supports_reasoning,
    input_price_per_million, output_price_per_million, pricing_source
)
VALUES
    ('agentrouter', 'gpt-6.1sol', 'GPT 6.1 Sol', 200000, 8192, true, true, false, false, 0, 0, 'unverified'),
    ('agentrouter', 'gpt-6-astra', 'GPT 6 Astra', 200000, 8192, true, true, false, false, 0, 0, 'unverified'),
    ('agentrouter', 'claude-opus-5.5', 'Claude Opus 5.5', 200000, 8192, true, true, false, false, 0, 0, 'unverified')
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

-- Propaga os novos modelos para providers AgentRouter já conectados.
INSERT INTO provider_models (tenant_id, provider_id, model_id, display_name, active, capabilities)
SELECT p.tenant_id, p.id, m.model_id, m.display_name, true,
       jsonb_build_object(
           'text_input', true,
           'text_output', true,
           'image_input', false,
           'audio_input', false,
           'video_input', false,
           'image_output', false,
           'audio_output', false,
           'video_output', false,
           'stream', true,
           'tools', true,
           'metadata_source', 'catalog:provider_catalog_models'
       )
FROM providers p
JOIN provider_catalog_models m ON m.slug = p.catalog_slug
WHERE p.kind = 'agentrouter'
  AND m.model_id IN ('gpt-6.1sol', 'gpt-6-astra', 'claude-opus-5.5')
  AND NOT EXISTS (
      SELECT 1 FROM provider_models pm
      WHERE pm.provider_id = p.id AND pm.tenant_id = p.tenant_id AND pm.model_id = m.model_id
  );
