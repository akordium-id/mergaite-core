-- name: CreateUnit :one
INSERT INTO units (
    id,
    tenant_id,
    code,
    name,
    symbol,
    category,
    precision,
    status,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetUnitByID :one
SELECT * FROM units
WHERE tenant_id = $1 AND id = $2 LIMIT 1;

-- name: GetUnitByCode :one
SELECT * FROM units
WHERE tenant_id = $1 AND code = $2 LIMIT 1;

-- name: ListUnits :many
SELECT * FROM units
WHERE tenant_id = $1
ORDER BY code ASC;

-- name: CreateUnitConversion :one
INSERT INTO unit_conversions (
    id,
    tenant_id,
    from_unit_id,
    to_unit_id,
    factor,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetUnitConversion :one
SELECT * FROM unit_conversions
WHERE tenant_id = $1 AND from_unit_id = $2 AND to_unit_id = $3 LIMIT 1;

-- name: ListUnitConversions :many
SELECT uc.*, u1.code as from_code, u2.code as to_code
FROM unit_conversions uc
JOIN units u1 ON u1.id = uc.from_unit_id
JOIN units u2 ON u2.id = uc.to_unit_id
WHERE uc.tenant_id = $1
ORDER BY uc.created_at ASC;

-- name: CreateProduct :one
INSERT INTO products (
    id,
    tenant_id,
    type,
    sku,
    barcode,
    name,
    description,
    unit_id,
    status,
    metadata,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING *;

-- name: GetProductByID :one
SELECT p.*, u.code as unit_code, u.name as unit_name, u.symbol as unit_symbol
FROM products p
JOIN units u ON u.id = p.unit_id
WHERE p.tenant_id = $1 AND p.id = $2 LIMIT 1;

-- name: GetProductBySKU :one
SELECT p.*, u.code as unit_code, u.name as unit_name, u.symbol as unit_symbol
FROM products p
JOIN units u ON u.id = p.unit_id
WHERE p.tenant_id = $1 AND p.sku = $2 LIMIT 1;

-- name: ListProducts :many
SELECT p.*, u.code as unit_code, u.symbol as unit_symbol
FROM products p
JOIN units u ON u.id = p.unit_id
WHERE p.tenant_id = $1
  AND (sqlc.narg('type')::varchar IS NULL OR p.type = sqlc.narg('type'))
  AND (sqlc.narg('status')::varchar IS NULL OR p.status = sqlc.narg('status'))
ORDER BY p.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountProducts :one
SELECT COUNT(*) FROM products
WHERE tenant_id = $1
  AND (sqlc.narg('type')::varchar IS NULL OR type = sqlc.narg('type'))
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'));

-- name: UpdateProduct :one
UPDATE products
SET
    name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    barcode = COALESCE(sqlc.narg('barcode'), barcode),
    status = COALESCE(sqlc.narg('status'), status),
    metadata = COALESCE(sqlc.narg('metadata'), metadata),
    updated_at = NOW()
WHERE tenant_id = sqlc.arg('tenant_id') AND id = sqlc.arg('id')
RETURNING *;

-- name: CreateProductVariant :one
INSERT INTO product_variants (
    id,
    tenant_id,
    product_id,
    sku,
    name,
    attributes,
    status,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: ListProductVariants :many
SELECT * FROM product_variants
WHERE tenant_id = $1 AND product_id = $2
ORDER BY sku ASC;

-- name: DeleteProductVariant :exec
DELETE FROM product_variants
WHERE tenant_id = $1 AND id = $2;
