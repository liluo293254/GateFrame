DROP INDEX IF EXISTS users_tenant_oidc_sub_uidx;
ALTER TABLE users DROP COLUMN IF EXISTS oidc_sub;
