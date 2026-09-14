-- 000013_create_service_accounts_and_api_keys.down.sql

DELETE FROM permissions WHERE code IN (
    'service_account:create',
    'service_account:read',
    'service_account:update',
    'service_account:delete',
    'api_key:create',
    'api_key:read',
    'api_key:revoke'
);

DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS service_accounts;
