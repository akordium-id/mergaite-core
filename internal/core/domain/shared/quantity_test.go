package shared_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

func TestQuantity_Creation(t *testing.T) {
	q, err := shared.NewQuantity(12.5, "KG")
	if err != nil {
		t.Fatalf("unexpected error creating Quantity: %v", err)
	}

	if q.Value() != 12.5 {
		t.Errorf("expected value 12.5, got %f", q.Value())
	}
	if q.UnitCode() != "KG" {
		t.Errorf("expected unit code KG, got %s", q.UnitCode())
	}

	// Normalization
	q2, err := shared.NewQuantity(5, "pcs ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q2.UnitCode() != "PCS" {
		t.Errorf("expected normalized unit PCS, got %s", q2.UnitCode())
	}

	// Empty unit
	_, err = shared.NewQuantity(10, "")
	if !errors.Is(err, shared.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for empty unit, got %v", err)
	}
}

func TestQuantity_Operations(t *testing.T) {
	q1 := shared.MustNewQuantity(10.5, "KG")
	q2 := shared.MustNewQuantity(4.5, "KG")
	qM := shared.MustNewQuantity(5.0, "M")

	// Add
	sum, err := q1.Add(q2)
	if err != nil {
		t.Fatalf("unexpected add error: %v", err)
	}
	if sum.Value() != 15.0 {
		t.Errorf("expected sum 15.0, got %f", sum.Value())
	}

	// Add mismatch
	_, err = q1.Add(qM)
	if !errors.Is(err, shared.ErrUnitMismatch) {
		t.Errorf("expected ErrUnitMismatch, got %v", err)
	}

	// Subtract
	diff, err := q1.Subtract(q2)
	if err != nil {
		t.Fatalf("unexpected subtract error: %v", err)
	}
	if diff.Value() != 6.0 {
		t.Errorf("expected diff 6.0, got %f", diff.Value())
	}

	// Multiply
	prod := q2.Multiply(2.0)
	if prod.Value() != 9.0 {
		t.Errorf("expected prod 9.0, got %f", prod.Value())
	}

	// Divide
	div, err := q1.Divide(2.0)
	if err != nil {
		t.Fatalf("unexpected divide error: %v", err)
	}
	if div.Value() != 5.25 {
		t.Errorf("expected div 5.25, got %f", div.Value())
	}

	// Divide by zero
	_, err = q1.Divide(0)
	if !errors.Is(err, shared.ErrDivisionByZero) {
		t.Errorf("expected ErrDivisionByZero, got %v", err)
	}
}

func TestQuantity_JSONSerialization(t *testing.T) {
	original := shared.MustNewQuantity(250.75, "BOX")

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal quantity: %v", err)
	}

	var unmarshaled shared.Quantity
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal quantity: %v", err)
	}

	if !original.Equals(unmarshaled) {
		t.Errorf("expected unmarshaled quantity to equal original: %v vs %v", original, unmarshaled)
	}
}
