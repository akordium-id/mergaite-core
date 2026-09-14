-- name: CreateAddress :one
INSERT INTO addresses (
    id,
    tenant_id,
    type,
    label,
    line1,
    line2,
    city,
    state,
    postal_code,
    country_code,
    latitude,
    longitude,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
) RETURNING *;

-- name: GetAddressByID :one
SELECT * FROM addresses
WHERE tenant_id = $1 AND id = $2 LIMIT 1;

-- name: LinkPartyAddress :exec
INSERT INTO party_addresses (
    tenant_id, party_id, address_id, address_type, is_primary, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6
) ON CONFLICT (party_id, address_id, address_type) DO UPDATE
SET is_primary = EXCLUDED.is_primary;

-- name: LinkOrganizationAddress :exec
INSERT INTO organization_addresses (
    tenant_id, organization_id, address_id, address_type, is_primary, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6
) ON CONFLICT (organization_id, address_id, address_type) DO UPDATE
SET is_primary = EXCLUDED.is_primary;

-- name: ListPartyAddresses :many
SELECT a.*, pa.is_primary, pa.address_type as link_type
FROM addresses a
JOIN party_addresses pa ON pa.address_id = a.id AND pa.tenant_id = a.tenant_id
WHERE a.tenant_id = $1 AND pa.party_id = $2
ORDER BY pa.is_primary DESC, a.created_at DESC;

-- name: ListOrganizationAddresses :many
SELECT a.*, oa.is_primary, oa.address_type as link_type
FROM addresses a
JOIN organization_addresses oa ON oa.address_id = a.id AND oa.tenant_id = a.tenant_id
WHERE a.tenant_id = $1 AND oa.organization_id = $2
ORDER BY oa.is_primary DESC, a.created_at DESC;

-- name: CreateContact :one
INSERT INTO contacts (
    id,
    tenant_id,
    party_id,
    organization_id,
    type,
    value,
    label,
    is_primary,
    verified_at,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: ListPartyContacts :many
SELECT * FROM contacts
WHERE tenant_id = $1 AND party_id = $2
ORDER BY is_primary DESC, created_at ASC;

-- name: ListOrganizationContacts :many
SELECT * FROM contacts
WHERE tenant_id = $1 AND organization_id = $2
ORDER BY is_primary DESC, created_at ASC;

-- name: DeleteContact :exec
DELETE FROM contacts
WHERE tenant_id = $1 AND id = $2;
