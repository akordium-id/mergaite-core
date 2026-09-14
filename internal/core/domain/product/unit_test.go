package product_test

import (
	"math"
	"testing"

	"github.com/akordium-id/mergaite-core/internal/core/domain/product"
	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

func TestConvert(t *testing.T) {
	tenantID, _ := shared.NewID()
	boxID, _ := shared.NewID()
	packID, _ := shared.NewID()
	pcsID, _ := shared.NewID()
	kgID, _ := shared.NewID()
	gID, _ := shared.NewID()
	palletID, _ := shared.NewID()

	conversions := []product.UnitConversion{
		// 1 Pallet = 10 Box
		{
			ID:         mustID(),
			TenantID:   tenantID,
			FromUnitID: palletID,
			ToUnitID:   boxID,
			Factor:     10.0,
		},
		// 1 Box = 12 Pack
		{
			ID:         mustID(),
			TenantID:   tenantID,
			FromUnitID: boxID,
			ToUnitID:   packID,
			Factor:     12.0,
		},
		// 1 Pack = 5 PCS
		{
			ID:         mustID(),
			TenantID:   tenantID,
			FromUnitID: packID,
			ToUnitID:   pcsID,
			Factor:     5.0,
		},
		// 1 KG = 1000 G
		{
			ID:         mustID(),
			TenantID:   tenantID,
			FromUnitID: kgID,
			ToUnitID:   gID,
			Factor:     1000.0,
		},
	}

	tests := []struct {
		name      string
		amount    float64
		fromID    shared.ID
		toID      shared.ID
		expected  float64
		expectErr bool
	}{
		{
			name:     "Identity conversion (same unit)",
			amount:   42.0,
			fromID:   boxID,
			toID:     boxID,
			expected: 42.0,
		},
		{
			name:     "Direct conversion: 2 Box to Pack (factor 12)",
			amount:   2.0,
			fromID:   boxID,
			toID:     packID,
			expected: 24.0,
		},
		{
			name:     "Inverse conversion: 24 Pack to Box (factor 1/12)",
			amount:   24.0,
			fromID:   packID,
			toID:     boxID,
			expected: 2.0,
		},
		{
			name:     "Multi-hop conversion: 1 Pallet to PCS (10 * 12 * 5 = 600 PCS)",
			amount:   1.0,
			fromID:   palletID,
			toID:     pcsID,
			expected: 600.0,
		},
		{
			name:     "Multi-hop inverse conversion: 600 PCS to Pallet",
			amount:   600.0,
			fromID:   pcsID,
			toID:     palletID,
			expected: 1.0,
		},
		{
			name:     "Direct weight conversion: 2.5 KG to G",
			amount:   2.5,
			fromID:   kgID,
			toID:     gID,
			expected: 2500.0,
		},
		{
			name:      "Disconnected conversion: Pallet to KG",
			amount:    1.0,
			fromID:    palletID,
			toID:      kgID,
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := product.Convert(tc.amount, tc.fromID, tc.toID, conversions)
			if tc.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if math.Abs(res-tc.expected) > 1e-6 {
				t.Errorf("expected %f, got %f", tc.expected, res)
			}
		})
	}
}

func mustID() shared.ID {
	id, _ := shared.NewID()
	return id
}
