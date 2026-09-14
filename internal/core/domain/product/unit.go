package product

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

type UnitStatus string

const (
	UnitStatusActive   UnitStatus = "active"
	UnitStatusInactive UnitStatus = "inactive"
)

// Unit represents a Unit of Measure (UoM).
type Unit struct {
	ID        shared.ID  `json:"id"`
	TenantID  shared.ID  `json:"tenant_id"`
	Code      string     `json:"code"`   // e.g. "PCS", "KG", "BOX", "M"
	Name      string     `json:"name"`   // e.g. "Pieces", "Kilogram", "Box"
	Symbol    string     `json:"symbol"` // e.g. "pcs", "kg", "box"
	Category  string     `json:"category"`
	Precision int32      `json:"precision"`
	Status    UnitStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// UnitConversion defines a linear multiplier between two units: from_unit * factor = to_unit.
// (e.g. 1 BOX * 12 = 12 PCS)
type UnitConversion struct {
	ID         shared.ID `json:"id"`
	TenantID   shared.ID `json:"tenant_id"`
	FromUnitID shared.ID `json:"from_unit_id"`
	ToUnitID   shared.ID `json:"to_unit_id"`
	Factor     float64   `json:"factor"`
	FromCode   string    `json:"from_code,omitempty"`
	ToCode     string    `json:"to_code,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Convert converts a quantity amount between units using direct, inverse, or multi-hop conversion path.
func Convert(amount float64, fromID, toID shared.ID, conversions []UnitConversion) (float64, error) {
	if fromID == toID {
		return amount, nil
	}

	type edge struct {
		to     shared.ID
		factor float64
	}

	adj := make(map[shared.ID][]edge)
	for _, c := range conversions {
		if math.Abs(c.Factor) < 1e-9 {
			continue
		}
		// Direct edge: from * factor = to
		adj[c.FromUnitID] = append(adj[c.FromUnitID], edge{to: c.ToUnitID, factor: c.Factor})
		// Inverse edge: to * (1/factor) = from
		adj[c.ToUnitID] = append(adj[c.ToUnitID], edge{to: c.FromUnitID, factor: 1.0 / c.Factor})
	}

	type queueItem struct {
		node   shared.ID
		factor float64
	}

	queue := []queueItem{{node: fromID, factor: 1.0}}
	visited := map[shared.ID]bool{fromID: true}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr.node == toID {
			return amount * curr.factor, nil
		}

		for _, e := range adj[curr.node] {
			if !visited[e.to] {
				visited[e.to] = true
				queue = append(queue, queueItem{
					node:   e.to,
					factor: curr.factor * e.factor,
				})
			}
		}
	}

	return 0, fmt.Errorf("%w: no conversion factor found between units", shared.ErrUnitMismatch)
}

func NormalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}
