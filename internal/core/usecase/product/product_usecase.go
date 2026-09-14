package product

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/akordium-id/mergaite-core/internal/core/domain/product"
	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

type CreateUnitCommand struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Symbol    string `json:"symbol"`
	Category  string `json:"category,omitempty"`
	Precision int32  `json:"precision"`
}

type CreateConversionCommand struct {
	FromUnitID shared.ID `json:"from_unit_id"`
	ToUnitID   shared.ID `json:"to_unit_id"`
	Factor     float64   `json:"factor"`
}

type ConvertQuantityCommand struct {
	Amount     float64   `json:"amount"`
	FromUnitID shared.ID `json:"from_unit_id"`
	ToUnitID   shared.ID `json:"to_unit_id"`
}

type CreateProductCommand struct {
	Type        product.Type   `json:"type"`
	SKU         string         `json:"sku,omitempty"`
	Barcode     string         `json:"barcode,omitempty"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	UnitID      shared.ID      `json:"unit_id"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type UpdateProductCommand struct {
	ID          shared.ID      `json:"id"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Barcode     string         `json:"barcode,omitempty"`
	Status      product.Status `json:"status,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type CreateVariantCommand struct {
	ProductID  shared.ID      `json:"product_id"`
	SKU        string         `json:"sku"`
	Name       string         `json:"name"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

type ProductListResult struct {
	Items    []product.Product `json:"items"`
	Total    int64             `json:"total"`
	Page     int32             `json:"page"`
	PageSize int32             `json:"page_size"`
}

type Usecase interface {
	CreateUnit(ctx context.Context, cmd CreateUnitCommand) (*product.Unit, error)
	GetUnitByID(ctx context.Context, id shared.ID) (*product.Unit, error)
	GetUnitByCode(ctx context.Context, code string) (*product.Unit, error)
	ListUnits(ctx context.Context) ([]product.Unit, error)

	CreateConversion(ctx context.Context, cmd CreateConversionCommand) (*product.UnitConversion, error)
	ConvertQuantity(ctx context.Context, cmd ConvertQuantityCommand) (float64, error)

	CreateProduct(ctx context.Context, cmd CreateProductCommand) (*product.Product, error)
	GetProductByID(ctx context.Context, id shared.ID) (*product.Product, error)
	GetProductBySKU(ctx context.Context, sku string) (*product.Product, error)
	ListProducts(ctx context.Context, page, pageSize int32, productType *product.Type, status *product.Status) (*ProductListResult, error)
	UpdateProduct(ctx context.Context, cmd UpdateProductCommand) (*product.Product, error)

	CreateVariant(ctx context.Context, cmd CreateVariantCommand) (*product.ProductVariant, error)
	ListVariants(ctx context.Context, productID shared.ID) ([]product.ProductVariant, error)
	DeleteVariant(ctx context.Context, id shared.ID) error
}

type usecase struct {
	unitRepo    product.UnitRepository
	productRepo product.ProductRepository
}

func NewUsecase(unitRepo product.UnitRepository, productRepo product.ProductRepository) Usecase {
	return &usecase{
		unitRepo:    unitRepo,
		productRepo: productRepo,
	}
}

// ----------------------------------------------------------------------------
// Unit Operations
// ----------------------------------------------------------------------------

func (u *usecase) CreateUnit(ctx context.Context, cmd CreateUnitCommand) (*product.Unit, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	code := product.NormalizeCode(cmd.Code)
	name := strings.TrimSpace(cmd.Name)
	symbol := strings.TrimSpace(cmd.Symbol)

	if code == "" || name == "" {
		return nil, fmt.Errorf("%w: unit code and name are required", shared.ErrInvalidInput)
	}
	if symbol == "" {
		symbol = strings.ToLower(code)
	}

	category := strings.TrimSpace(cmd.Category)
	if category == "" {
		category = "quantity"
	}

	id, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	unit := &product.Unit{
		ID:        id,
		TenantID:  tenantID,
		Code:      code,
		Name:      name,
		Symbol:    symbol,
		Category:  category,
		Precision: cmd.Precision,
		Status:    product.UnitStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := u.unitRepo.CreateUnit(ctx, unit); err != nil {
		return nil, err
	}

	return unit, nil
}

func (u *usecase) GetUnitByID(ctx context.Context, id shared.ID) (*product.Unit, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	return u.unitRepo.GetUnitByID(ctx, tenantID, id)
}

func (u *usecase) GetUnitByCode(ctx context.Context, code string) (*product.Unit, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	return u.unitRepo.GetUnitByCode(ctx, tenantID, product.NormalizeCode(code))
}

func (u *usecase) ListUnits(ctx context.Context) ([]product.Unit, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	return u.unitRepo.ListUnits(ctx, tenantID)
}

func (u *usecase) CreateConversion(ctx context.Context, cmd CreateConversionCommand) (*product.UnitConversion, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	if cmd.FromUnitID == cmd.ToUnitID {
		return nil, fmt.Errorf("%w: from and to units cannot be identical", shared.ErrInvalidInput)
	}
	if cmd.Factor <= 0 {
		return nil, fmt.Errorf("%w: conversion factor must be positive", shared.ErrInvalidInput)
	}

	// Verify both units exist
	if _, err := u.unitRepo.GetUnitByID(ctx, tenantID, cmd.FromUnitID); err != nil {
		return nil, fmt.Errorf("from unit: %w", err)
	}
	if _, err := u.unitRepo.GetUnitByID(ctx, tenantID, cmd.ToUnitID); err != nil {
		return nil, fmt.Errorf("to unit: %w", err)
	}

	id, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	conv := &product.UnitConversion{
		ID:         id,
		TenantID:   tenantID,
		FromUnitID: cmd.FromUnitID,
		ToUnitID:   cmd.ToUnitID,
		Factor:     cmd.Factor,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := u.unitRepo.CreateConversion(ctx, conv); err != nil {
		return nil, err
	}

	return conv, nil
}

func (u *usecase) ConvertQuantity(ctx context.Context, cmd ConvertQuantityCommand) (float64, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return 0, err
	}

	conversions, err := u.unitRepo.ListConversions(ctx, tenantID)
	if err != nil {
		return 0, err
	}

	return product.Convert(cmd.Amount, cmd.FromUnitID, cmd.ToUnitID, conversions)
}

// ----------------------------------------------------------------------------
// Product Operations
// ----------------------------------------------------------------------------

func (u *usecase) CreateProduct(ctx context.Context, cmd CreateProductCommand) (*product.Product, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: product name is required", shared.ErrInvalidInput)
	}

	sku := strings.TrimSpace(cmd.SKU)
	if sku != "" {
		existing, err := u.productRepo.GetProductBySKU(ctx, tenantID, sku)
		if err != nil && err != shared.ErrNotFound {
			return nil, err
		}
		if existing != nil {
			return nil, fmt.Errorf("%w: product with SKU '%s' already exists", shared.ErrAlreadyExists, sku)
		}
	}

	// Verify unit exists
	unit, err := u.unitRepo.GetUnitByID(ctx, tenantID, cmd.UnitID)
	if err != nil {
		return nil, fmt.Errorf("unit: %w", err)
	}

	prodType := cmd.Type
	if prodType == "" {
		prodType = product.TypeGoods
	}

	id, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	metadata := cmd.Metadata
	if metadata == nil {
		metadata = make(map[string]any)
	}

	p := &product.Product{
		ID:          id,
		TenantID:    tenantID,
		Type:        prodType,
		SKU:         sku,
		Barcode:     strings.TrimSpace(cmd.Barcode),
		Name:        name,
		Description: strings.TrimSpace(cmd.Description),
		UnitID:      unit.ID,
		UnitCode:    unit.Code,
		UnitSymbol:  unit.Symbol,
		Status:      product.StatusActive,
		Metadata:    metadata,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := u.productRepo.CreateProduct(ctx, p); err != nil {
		return nil, err
	}

	return p, nil
}

func (u *usecase) GetProductByID(ctx context.Context, id shared.ID) (*product.Product, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	return u.productRepo.GetProductByID(ctx, tenantID, id)
}

func (u *usecase) GetProductBySKU(ctx context.Context, sku string) (*product.Product, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	return u.productRepo.GetProductBySKU(ctx, tenantID, strings.TrimSpace(sku))
}

func (u *usecase) ListProducts(ctx context.Context, page, pageSize int32, productType *product.Type, status *product.Status) (*ProductListResult, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	filter := product.ProductFilter{
		Type:   productType,
		Status: status,
		Limit:  pageSize,
		Offset: offset,
	}

	items, total, err := u.productRepo.ListProducts(ctx, tenantID, filter)
	if err != nil {
		return nil, err
	}

	return &ProductListResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (u *usecase) UpdateProduct(ctx context.Context, cmd UpdateProductCommand) (*product.Product, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	p, err := u.productRepo.GetProductByID(ctx, tenantID, cmd.ID)
	if err != nil {
		return nil, err
	}

	if cmd.Name != "" {
		p.Name = strings.TrimSpace(cmd.Name)
	}
	if cmd.Description != "" {
		p.Description = strings.TrimSpace(cmd.Description)
	}
	if cmd.Barcode != "" {
		p.Barcode = strings.TrimSpace(cmd.Barcode)
	}
	if cmd.Status != "" {
		p.Status = cmd.Status
	}
	if cmd.Metadata != nil {
		p.Metadata = cmd.Metadata
	}

	if err := u.productRepo.UpdateProduct(ctx, p); err != nil {
		return nil, err
	}

	return p, nil
}

func (u *usecase) CreateVariant(ctx context.Context, cmd CreateVariantCommand) (*product.ProductVariant, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	sku := strings.TrimSpace(cmd.SKU)
	name := strings.TrimSpace(cmd.Name)
	if sku == "" || name == "" {
		return nil, fmt.Errorf("%w: variant sku and name are required", shared.ErrInvalidInput)
	}

	// Verify parent product exists
	if _, err := u.productRepo.GetProductByID(ctx, tenantID, cmd.ProductID); err != nil {
		return nil, fmt.Errorf("parent product: %w", err)
	}

	id, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	attributes := cmd.Attributes
	if attributes == nil {
		attributes = make(map[string]any)
	}

	v := &product.ProductVariant{
		ID:         id,
		TenantID:   tenantID,
		ProductID:  cmd.ProductID,
		SKU:        sku,
		Name:       name,
		Attributes: attributes,
		Status:     product.StatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := u.productRepo.CreateVariant(ctx, v); err != nil {
		return nil, err
	}

	return v, nil
}

func (u *usecase) ListVariants(ctx context.Context, productID shared.ID) ([]product.ProductVariant, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	return u.productRepo.ListVariants(ctx, tenantID, productID)
}

func (u *usecase) DeleteVariant(ctx context.Context, id shared.ID) error {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return err
	}
	return u.productRepo.DeleteVariant(ctx, tenantID, id)
}
