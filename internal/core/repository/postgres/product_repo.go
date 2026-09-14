package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akordium-id/mergaite-core/internal/core/domain/product"
	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
	"github.com/akordium-id/mergaite-core/internal/core/repository/postgres/sqlc"
)

type productRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewProductRepository creates a new PostgreSQL Product repository.
func NewProductRepository(pool *pgxpool.Pool) product.ProductRepository {
	return &productRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// NewUnitRepository creates a new PostgreSQL Unit repository.
func NewUnitRepository(pool *pgxpool.Pool) product.UnitRepository {
	return &productRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// ----------------------------------------------------------------------------
// UnitRepository Implementation
// ----------------------------------------------------------------------------

func (r *productRepository) CreateUnit(ctx context.Context, u *product.Unit) error {
	params := sqlc.CreateUnitParams{
		ID:        shared.ToPgUUID(u.ID),
		TenantID:  shared.ToPgUUID(u.TenantID),
		Code:      u.Code,
		Name:      u.Name,
		Symbol:    u.Symbol,
		Category:  u.Category,
		Precision: u.Precision,
		Status:    string(u.Status),
		CreatedAt: pgtype.Timestamptz{Time: u.CreatedAt, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: u.UpdatedAt, Valid: true},
	}

	row, err := r.queries.CreateUnit(ctx, params)
	if err != nil {
		return err
	}

	*u = *toDomainUnit(&row)
	return nil
}

func (r *productRepository) GetUnitByID(ctx context.Context, tenantID, id shared.ID) (*product.Unit, error) {
	row, err := r.queries.GetUnitByID(ctx, sqlc.GetUnitByIDParams{
		TenantID: shared.ToPgUUID(tenantID),
		ID:       shared.ToPgUUID(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return toDomainUnit(&row), nil
}

func (r *productRepository) GetUnitByCode(ctx context.Context, tenantID shared.ID, code string) (*product.Unit, error) {
	row, err := r.queries.GetUnitByCode(ctx, sqlc.GetUnitByCodeParams{
		TenantID: shared.ToPgUUID(tenantID),
		Code:     code,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return toDomainUnit(&row), nil
}

func (r *productRepository) ListUnits(ctx context.Context, tenantID shared.ID) ([]product.Unit, error) {
	rows, err := r.queries.ListUnits(ctx, shared.ToPgUUID(tenantID))
	if err != nil {
		return nil, err
	}

	units := make([]product.Unit, len(rows))
	for i, row := range rows {
		units[i] = *toDomainUnit(&row)
	}
	return units, nil
}

func (r *productRepository) CreateConversion(ctx context.Context, conv *product.UnitConversion) error {
	params := sqlc.CreateUnitConversionParams{
		ID:         shared.ToPgUUID(conv.ID),
		TenantID:   shared.ToPgUUID(conv.TenantID),
		FromUnitID: shared.ToPgUUID(conv.FromUnitID),
		ToUnitID:   shared.ToPgUUID(conv.ToUnitID),
		Factor:     floatToNumeric(&conv.Factor),
		CreatedAt:  pgtype.Timestamptz{Time: conv.CreatedAt, Valid: true},
		UpdatedAt:  pgtype.Timestamptz{Time: conv.UpdatedAt, Valid: true},
	}

	row, err := r.queries.CreateUnitConversion(ctx, params)
	if err != nil {
		return err
	}

	f := numericToFloat(row.Factor)
	var factor float64
	if f != nil {
		factor = *f
	}

	conv.ID = shared.FromPgUUID(row.ID)
	conv.Factor = factor
	conv.CreatedAt = row.CreatedAt.Time
	conv.UpdatedAt = row.UpdatedAt.Time
	return nil
}

func (r *productRepository) GetConversion(ctx context.Context, tenantID, fromID, toID shared.ID) (*product.UnitConversion, error) {
	row, err := r.queries.GetUnitConversion(ctx, sqlc.GetUnitConversionParams{
		TenantID:   shared.ToPgUUID(tenantID),
		FromUnitID: shared.ToPgUUID(fromID),
		ToUnitID:   shared.ToPgUUID(toID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}

	f := numericToFloat(row.Factor)
	var factor float64
	if f != nil {
		factor = *f
	}

	return &product.UnitConversion{
		ID:         shared.FromPgUUID(row.ID),
		TenantID:   shared.FromPgUUID(row.TenantID),
		FromUnitID: shared.FromPgUUID(row.FromUnitID),
		ToUnitID:   shared.FromPgUUID(row.ToUnitID),
		Factor:     factor,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}, nil
}

func (r *productRepository) ListConversions(ctx context.Context, tenantID shared.ID) ([]product.UnitConversion, error) {
	rows, err := r.queries.ListUnitConversions(ctx, shared.ToPgUUID(tenantID))
	if err != nil {
		return nil, err
	}

	convs := make([]product.UnitConversion, len(rows))
	for i, row := range rows {
		f := numericToFloat(row.Factor)
		var factor float64
		if f != nil {
			factor = *f
		}
		convs[i] = product.UnitConversion{
			ID:         shared.FromPgUUID(row.ID),
			TenantID:   shared.FromPgUUID(row.TenantID),
			FromUnitID: shared.FromPgUUID(row.FromUnitID),
			ToUnitID:   shared.FromPgUUID(row.ToUnitID),
			Factor:     factor,
			FromCode:   row.FromCode,
			ToCode:     row.ToCode,
			CreatedAt:  row.CreatedAt.Time,
			UpdatedAt:  row.UpdatedAt.Time,
		}
	}
	return convs, nil
}

// ----------------------------------------------------------------------------
// ProductRepository Implementation
// ----------------------------------------------------------------------------

func (r *productRepository) CreateProduct(ctx context.Context, p *product.Product) error {
	metadataJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return fmt.Errorf("%w: invalid product metadata json", shared.ErrInvalidInput)
	}

	var sku *string
	if p.SKU != "" {
		sku = &p.SKU
	}
	var barcode *string
	if p.Barcode != "" {
		barcode = &p.Barcode
	}
	var desc *string
	if p.Description != "" {
		desc = &p.Description
	}

	params := sqlc.CreateProductParams{
		ID:          shared.ToPgUUID(p.ID),
		TenantID:    shared.ToPgUUID(p.TenantID),
		Type:        string(p.Type),
		Sku:         sku,
		Barcode:     barcode,
		Name:        p.Name,
		Description: desc,
		UnitID:      shared.ToPgUUID(p.UnitID),
		Status:      string(p.Status),
		Metadata:    metadataJSON,
		CreatedAt:   pgtype.Timestamptz{Time: p.CreatedAt, Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: p.UpdatedAt, Valid: true},
	}

	row, err := r.queries.CreateProduct(ctx, params)
	if err != nil {
		return err
	}

	p.ID = shared.FromPgUUID(row.ID)
	p.CreatedAt = row.CreatedAt.Time
	p.UpdatedAt = row.UpdatedAt.Time
	return nil
}

func (r *productRepository) GetProductByID(ctx context.Context, tenantID, id shared.ID) (*product.Product, error) {
	row, err := r.queries.GetProductByID(ctx, sqlc.GetProductByIDParams{
		TenantID: shared.ToPgUUID(tenantID),
		ID:       shared.ToPgUUID(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}

	p := toDomainProductFromGetRow(&row)
	variants, err := r.ListVariants(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	p.Variants = variants

	return p, nil
}

func (r *productRepository) GetProductBySKU(ctx context.Context, tenantID shared.ID, sku string) (*product.Product, error) {
	row, err := r.queries.GetProductBySKU(ctx, sqlc.GetProductBySKUParams{
		TenantID: shared.ToPgUUID(tenantID),
		Sku:      &sku,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}

	p := toDomainProductFromGetSKURow(&row)
	variants, err := r.ListVariants(ctx, tenantID, p.ID)
	if err != nil {
		return nil, err
	}
	p.Variants = variants

	return p, nil
}

func (r *productRepository) ListProducts(ctx context.Context, tenantID shared.ID, filter product.ProductFilter) ([]product.Product, int64, error) {
	limit := max(filter.Limit, 20)
	offset := max(filter.Offset, 0)

	var typeStr *string
	if filter.Type != nil {
		s := string(*filter.Type)
		typeStr = &s
	}
	var statusStr *string
	if filter.Status != nil {
		s := string(*filter.Status)
		statusStr = &s
	}

	total, err := r.queries.CountProducts(ctx, sqlc.CountProductsParams{
		TenantID: shared.ToPgUUID(tenantID),
		Type:     typeStr,
		Status:   statusStr,
	})
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.queries.ListProducts(ctx, sqlc.ListProductsParams{
		TenantID: shared.ToPgUUID(tenantID),
		Limit:    limit,
		Offset:   offset,
		Type:     typeStr,
		Status:   statusStr,
	})
	if err != nil {
		return nil, 0, err
	}

	products := make([]product.Product, len(rows))
	for i, row := range rows {
		products[i] = *toDomainProductFromListRow(&row)
	}

	return products, total, nil
}

func (r *productRepository) UpdateProduct(ctx context.Context, p *product.Product) error {
	var metadataJSON []byte
	if p.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(p.Metadata)
		if err != nil {
			return fmt.Errorf("%w: invalid metadata json", shared.ErrInvalidInput)
		}
	}

	var barcode *string
	if p.Barcode != "" {
		barcode = &p.Barcode
	}
	var desc *string
	if p.Description != "" {
		desc = &p.Description
	}
	statusStr := string(p.Status)

	params := sqlc.UpdateProductParams{
		TenantID:    shared.ToPgUUID(p.TenantID),
		ID:          shared.ToPgUUID(p.ID),
		Name:        &p.Name,
		Description: desc,
		Barcode:     barcode,
		Status:      &statusStr,
		Metadata:    metadataJSON,
	}

	row, err := r.queries.UpdateProduct(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return shared.ErrNotFound
		}
		return err
	}

	p.UpdatedAt = row.UpdatedAt.Time
	return nil
}

func (r *productRepository) CreateVariant(ctx context.Context, v *product.ProductVariant) error {
	attrJSON, err := json.Marshal(v.Attributes)
	if err != nil {
		return fmt.Errorf("%w: invalid variant attributes json", shared.ErrInvalidInput)
	}

	params := sqlc.CreateProductVariantParams{
		ID:         shared.ToPgUUID(v.ID),
		TenantID:   shared.ToPgUUID(v.TenantID),
		ProductID:  shared.ToPgUUID(v.ProductID),
		Sku:        v.SKU,
		Name:       v.Name,
		Attributes: attrJSON,
		Status:     string(v.Status),
		CreatedAt:  pgtype.Timestamptz{Time: v.CreatedAt, Valid: true},
		UpdatedAt:  pgtype.Timestamptz{Time: v.UpdatedAt, Valid: true},
	}

	row, err := r.queries.CreateProductVariant(ctx, params)
	if err != nil {
		return err
	}

	v.ID = shared.FromPgUUID(row.ID)
	v.CreatedAt = row.CreatedAt.Time
	v.UpdatedAt = row.UpdatedAt.Time
	return nil
}

func (r *productRepository) ListVariants(ctx context.Context, tenantID, productID shared.ID) ([]product.ProductVariant, error) {
	rows, err := r.queries.ListProductVariants(ctx, sqlc.ListProductVariantsParams{
		TenantID:  shared.ToPgUUID(tenantID),
		ProductID: shared.ToPgUUID(productID),
	})
	if err != nil {
		return nil, err
	}

	variants := make([]product.ProductVariant, len(rows))
	for i, row := range rows {
		var attrs map[string]any
		if len(row.Attributes) > 0 {
			_ = json.Unmarshal(row.Attributes, &attrs)
		}
		variants[i] = product.ProductVariant{
			ID:         shared.FromPgUUID(row.ID),
			TenantID:   shared.FromPgUUID(row.TenantID),
			ProductID:  shared.FromPgUUID(row.ProductID),
			SKU:        row.Sku,
			Name:       row.Name,
			Attributes: attrs,
			Status:     product.Status(row.Status),
			CreatedAt:  row.CreatedAt.Time,
			UpdatedAt:  row.UpdatedAt.Time,
		}
	}
	return variants, nil
}

func (r *productRepository) DeleteVariant(ctx context.Context, tenantID, id shared.ID) error {
	return r.queries.DeleteProductVariant(ctx, sqlc.DeleteProductVariantParams{
		TenantID: shared.ToPgUUID(tenantID),
		ID:       shared.ToPgUUID(id),
	})
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

func toDomainUnit(row *sqlc.Unit) *product.Unit {
	return &product.Unit{
		ID:        shared.FromPgUUID(row.ID),
		TenantID:  shared.FromPgUUID(row.TenantID),
		Code:      row.Code,
		Name:      row.Name,
		Symbol:    row.Symbol,
		Category:  row.Category,
		Precision: row.Precision,
		Status:    product.UnitStatus(row.Status),
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}

func toDomainProductFromGetRow(row *sqlc.GetProductByIDRow) *product.Product {
	var metadata map[string]any
	if len(row.Metadata) > 0 {
		_ = json.Unmarshal(row.Metadata, &metadata)
	}
	return &product.Product{
		ID:          shared.FromPgUUID(row.ID),
		TenantID:    shared.FromPgUUID(row.TenantID),
		Type:        product.Type(row.Type),
		SKU:         strFromPtr(row.Sku),
		Barcode:     strFromPtr(row.Barcode),
		Name:        row.Name,
		Description: strFromPtr(row.Description),
		UnitID:      shared.FromPgUUID(row.UnitID),
		UnitCode:    row.UnitCode,
		UnitSymbol:  row.UnitSymbol,
		Status:      product.Status(row.Status),
		Metadata:    metadata,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
}

func toDomainProductFromGetSKURow(row *sqlc.GetProductBySKURow) *product.Product {
	var metadata map[string]any
	if len(row.Metadata) > 0 {
		_ = json.Unmarshal(row.Metadata, &metadata)
	}
	return &product.Product{
		ID:          shared.FromPgUUID(row.ID),
		TenantID:    shared.FromPgUUID(row.TenantID),
		Type:        product.Type(row.Type),
		SKU:         strFromPtr(row.Sku),
		Barcode:     strFromPtr(row.Barcode),
		Name:        row.Name,
		Description: strFromPtr(row.Description),
		UnitID:      shared.FromPgUUID(row.UnitID),
		UnitCode:    row.UnitCode,
		UnitSymbol:  row.UnitSymbol,
		Status:      product.Status(row.Status),
		Metadata:    metadata,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
}

func toDomainProductFromListRow(row *sqlc.ListProductsRow) *product.Product {
	var metadata map[string]any
	if len(row.Metadata) > 0 {
		_ = json.Unmarshal(row.Metadata, &metadata)
	}
	return &product.Product{
		ID:          shared.FromPgUUID(row.ID),
		TenantID:    shared.FromPgUUID(row.TenantID),
		Type:        product.Type(row.Type),
		SKU:         strFromPtr(row.Sku),
		Barcode:     strFromPtr(row.Barcode),
		Name:        row.Name,
		Description: strFromPtr(row.Description),
		UnitID:      shared.FromPgUUID(row.UnitID),
		UnitCode:    row.UnitCode,
		UnitSymbol:  row.UnitSymbol,
		Status:      product.Status(row.Status),
		Metadata:    metadata,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
}
