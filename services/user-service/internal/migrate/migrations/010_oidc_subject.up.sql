-- Link local users to OIDC subject (per tenant).

ALTER TABLE users ADD COLUMN IF NOT EXISTS oidc_sub VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS users_tenant_oidc_sub_uidx
    ON users (tenant_id, oidc_sub)
    WHERE oidc_sub IS NOT NULL;
