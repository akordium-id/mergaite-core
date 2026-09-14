package shared_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

func TestMoney_Creation(t *testing.T) {
	m, err := shared.NewMoney(100000, "IDR")
	if err != nil {
		t.Fatalf("unexpected error creating Money: %v", err)
	}

	if m.Amount() != 100000 {
		t.Errorf("expected amount 100000, got %d", m.Amount())
	}
	if m.Currency() != "IDR" {
		t.Errorf("expected currency IDR, got %s", m.Currency())
	}

	// Case normalization
	m2, err := shared.NewMoney(50, "usd ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m2.Currency() != "USD" {
		t.Errorf("expected normalized currency USD, got %s", m2.Currency())
	}

	// Invalid currency
	_, err = shared.NewMoney(100, "RUPIAH")
	if !errors.Is(err, shared.ErrInvalidCurrency) {
		t.Errorf("expected ErrInvalidCurrency, got %v", err)
	}
}

func TestMoney_Operations(t *testing.T) {
	m1 := shared.MustNewMoney(50000, "IDR")
	m2 := shared.MustNewMoney(25000, "IDR")
	mUSD := shared.MustNewMoney(10, "USD")

	// Add
	sum, err := m1.Add(m2)
	if err != nil {
		t.Fatalf("unexpected add error: %v", err)
	}
	if sum.Amount() != 75000 {
		t.Errorf("expected sum 75000, got %d", sum.Amount())
	}

	// Add currency mismatch
	_, err = m1.Add(mUSD)
	if !errors.Is(err, shared.ErrCurrencyMismatch) {
		t.Errorf("expected ErrCurrencyMismatch, got %v", err)
	}

	// Subtract
	diff, err := m1.Subtract(m2)
	if err != nil {
		t.Fatalf("unexpected subtract error: %v", err)
	}
	if diff.Amount() != 25000 {
		t.Errorf("expected diff 25000, got %d", diff.Amount())
	}

	// Multiply
	prod := m2.Multiply(3)
	if prod.Amount() != 75000 {
		t.Errorf("expected prod 75000, got %d", prod.Amount())
	}

	// MultiplyFloat
	taxed := m1.MultiplyFloat(1.11) // 11% PPN
	if taxed.Amount() != 55500 {
		t.Errorf("expected taxed 55500, got %d", taxed.Amount())
	}

	// Divide
	div, err := m1.Divide(2)
	if err != nil {
		t.Fatalf("unexpected divide error: %v", err)
	}
	if div.Amount() != 25000 {
		t.Errorf("expected div 25000, got %d", div.Amount())
	}

	// Division by zero
	_, err = m1.Divide(0)
	if !errors.Is(err, shared.ErrDivisionByZero) {
		t.Errorf("expected ErrDivisionByZero, got %v", err)
	}
}

func TestMoney_JSONSerialization(t *testing.T) {
	original := shared.MustNewMoney(1500000, "IDR")

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal money: %v", err)
	}

	var unmarshaled shared.Money
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal money: %v", err)
	}

	if !original.Equals(unmarshaled) {
		t.Errorf("expected unmarshaled money to equal original: %v vs %v", original, unmarshaled)
	}
}
