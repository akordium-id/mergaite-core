package product

import (
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

type Type string

const (
	TypeGoods   Type = "goods"   // Physical item (stockable)
	TypeService Type = "service" // Service / non-physical
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusArchived Status = "archived"
)

// Product represents a goods or service item cataloged in Mergiate.
type Product struct {
	ID          shared.ID        `json:"id"`
	TenantID    shared.ID        `json:"tenant_id"`
	Type        Type             `json:"type"`
	SKU         string           `json:"sku,omitempty"`
	Barcode     string           `json:"barcode,omitempty"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	UnitID      shared.ID        `json:"unit_id"`
	UnitCode    string           `json:"unit_code,omitempty"`
	UnitSymbol  string           `json:"unit_symbol,omitempty"`
	Status      Status           `json:"status"`
	Metadata    map[string]any   `json:"metadata,omitempty"`
	Variants    []ProductVariant `json:"variants,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// ProductVariant represents a distinct variant (e.g. size, color) belonging to a parent Product.
type ProductVariant struct {
	ID         shared.ID      `json:"id"`
	TenantID   shared.ID      `json:"tenant_id"`
	ProductID  shared.ID      `json:"product_id"`
	SKU        string         `json:"sku"`
	Name       string         `json:"name"`
	Attributes map[string]any `json:"attributes,omitempty"`
	Status     Status         `json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}
