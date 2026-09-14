package product

import (
	"context"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

type ProductFilter struct {
	Type   *Type
	Status *Status
	Limit  int32
	Offset int32
}

// UnitRepository defines storage operations for Units and Conversions.
type UnitRepository interface {
	CreateUnit(ctx context.Context, unit *Unit) error
	GetUnitByID(ctx context.Context, tenantID, id shared.ID) (*Unit, error)
	GetUnitByCode(ctx context.Context, tenantID shared.ID, code string) (*Unit, error)
	ListUnits(ctx context.Context, tenantID shared.ID) ([]Unit, error)

	CreateConversion(ctx context.Context, conv *UnitConversion) error
	GetConversion(ctx context.Context, tenantID, fromID, toID shared.ID) (*UnitConversion, error)
	ListConversions(ctx context.Context, tenantID shared.ID) ([]UnitConversion, error)
}

// ProductRepository defines storage operations for Products and Variants.
type ProductRepository interface {
	CreateProduct(ctx context.Context, p *Product) error
	GetProductByID(ctx context.Context, tenantID, id shared.ID) (*Product, error)
	GetProductBySKU(ctx context.Context, tenantID shared.ID, sku string) (*Product, error)
	ListProducts(ctx context.Context, tenantID shared.ID, filter ProductFilter) ([]Product, int64, error)
	UpdateProduct(ctx context.Context, p *Product) error

	CreateVariant(ctx context.Context, v *ProductVariant) error
	ListVariants(ctx context.Context, tenantID, productID shared.ID) ([]ProductVariant, error)
	DeleteVariant(ctx context.Context, tenantID, id shared.ID) error
}
