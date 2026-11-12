UPDATE provider_accounts pa
SET base_url = COALESCE(
    (
        SELECT pc.default_base_url
        FROM providers p
        JOIN provider_catalog pc ON pc.slug = p.catalog_slug
        WHERE p.id = pa.provider_id AND p.tenant_id = pa.tenant_id
    ),
    (
        SELECT peer.base_url
        FROM provider_accounts peer
        WHERE peer.provider_id = pa.provider_id
          AND peer.tenant_id = pa.tenant_id
          AND peer.base_url <> ''
        ORDER BY peer.priority, peer.created_at
        LIMIT 1
    ),
    ''
)
WHERE pa.base_url = '';
