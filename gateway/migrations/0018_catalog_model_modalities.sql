ALTER TABLE provider_catalog_models
    ADD COLUMN IF NOT EXISTS supports_text boolean NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS supports_audio boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS supports_video boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS supports_embeddings boolean NOT NULL DEFAULT false;

UPDATE provider_models pm
SET capabilities = capabilities || jsonb_build_object(
    'text_input', pcm.supports_text,
    'text_output', pcm.supports_text,
    'image_input', pcm.supports_vision,
    'audio_input', pcm.supports_audio,
    'video_input', pcm.supports_video,
    'image_output', false,
    'audio_output', false,
    'video_output', false,
    'vision', pcm.supports_vision,
    'audio', pcm.supports_audio,
    'embeddings', pcm.supports_embeddings,
    'metadata_source', 'catalog:provider_catalog_models'
)
FROM providers p
JOIN provider_catalog_models pcm ON pcm.slug = p.catalog_slug
WHERE pm.provider_id = p.id
  AND pm.tenant_id = p.tenant_id
  AND pm.model_id = pcm.model_id
  AND coalesce(pm.capabilities->>'metadata_source', '') <> 'provider:/models';
