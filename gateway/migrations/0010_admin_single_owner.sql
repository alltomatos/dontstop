CREATE UNIQUE INDEX admin_users_single_owner_idx
    ON admin_users (tenant_id)
    WHERE role = 'owner';
