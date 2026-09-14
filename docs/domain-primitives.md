# Mergiate Core: Domain Primitives & Value Objects

This document explains the usage and safety guarantees of the core Domain Value Objects located in [`internal/core/domain/shared/`](file:///home/orin/code/archive/mergaite-core/internal/core/domain/shared/).

---

## 1. ID Primitive (`shared.ID`)

Mergiate Core uses **UUIDv7** for universal entity identification.

```go
import "github.com/akordium-id/mergaite-core/internal/core/domain/shared"

// Generate a new time-ordered UUIDv7
id, err := shared.NewID()
if err != nil {
    // handle error
}

// Or panic if system clock fails
id := shared.MustNewID()

// Parse from string representation
parsedID, err := shared.ParseID("01a09e91-c4d0-7d8a-b36e-14ad3d88f6a0")

// Interop with pgx / sqlc pgtype.UUID
pgUUID := shared.ToPgUUID(id)
domainID := shared.FromPgUUID(pgUUID)
```

---

## 2. Money Value Object (`shared.Money`)

`shared.Money` is an immutable Value Object that stores amounts in **minor currency units** (e.g., cents, Indonesian Rupiah without decimals) alongside an ISO-4217 currency code.

### Invariant Rules
- Amounts are represented as `int64` to prevent floating-point rounding errors (e.g., `0.1 + 0.2 != 0.3`).
- Arithmetic operations (`Add`, `Subtract`) between mismatched currencies return `shared.ErrCurrencyMismatch`.

### Code Examples
```go
// Creating Money instances
price, err := shared.NewMoney(150000, "IDR") // IDR 150,000
tax, _ := shared.NewMoney(16500, "IDR")      // 11% PPN

// Safe addition
total, err := price.Add(tax) // Returns IDR 166500
if err != nil {
    // ErrCurrencyMismatch if currencies differ
}

// Subtraction
discount := shared.MustNewMoney(10000, "IDR")
finalPrice, _ := total.Subtract(discount)

// Multiplication by quantity or percentage
multiplied := price.Multiply(3)         // 3 items
taxed := price.MultiplyFloat(1.11)      // 11% tax applied

// Division
portion, err := finalPrice.Divide(2)

// Currency checks
if price.Equals(tax) { ... }
if price.IsPositive() { ... }
```

### JSON Serialization
`shared.Money` marshals to a clean JSON object:
```json
{
  "amount": 150000,
  "currency": "IDR"
}
```

---

## 3. Quantity & Unit (`shared.Quantity` & `shared.Unit`)

`Quantity` represents an amount coupled with a Unit of Measure (`Unit` - UoM).

```go
// Defining a unit
kgUnit := shared.Unit{
    Code:   "KG",
    Name:   "Kilogram",
    Symbol: "kg",
}

// Creating quantities
q1 := shared.MustNewQuantity(10.5, "KG")
q2 := shared.MustNewQuantity(4.5, "KG")
qMeters := shared.MustNewQuantity(5.0, "M")

// Addition & Subtraction (enforces identical UnitCode)
sum, err := q1.Add(q2) // 15.00 KG
_, err = q1.Add(qMeters) // Returns shared.ErrUnitMismatch!

// Scalar multiplication and division
half, _ := q1.Divide(2.0) // 5.25 KG
doubled := q2.Multiply(2.0) // 9.00 KG
```

### JSON Serialization
```json
{
  "value": 15.0,
  "unit_code": "KG"
}
```

---

## 4. Tenant Context & Isolation

Multi-tenancy context is propagated through standard Go `context.Context`.

```go
// Inject tenant into request context
ctx = shared.WithTenantID(ctx, tenantID)

// Retrieve optional tenant
if id, ok := shared.GetTenantID(ctx); ok {
    // tenant present
}

// Enforce mandatory tenant (returns shared.ErrTenantRequired if absent)
tenantID, err := shared.RequireTenantID(ctx)
if err != nil {
    return err
}
```

---

## 5. Domain Error Handling

Standard domain errors in `shared/errors.go`:

| Domain Error | HTTP Mapping | Description |
|---|---|---|
| `shared.ErrNotFound` | `404 Not Found` | Requested entity does not exist |
| `shared.ErrAlreadyExists` | `409 Conflict` | Unique constraint violated (e.g. duplicate tenant code) |
| `shared.ErrInvalidInput` | `400 Bad Request` | Input fails domain invariants |
| `shared.ErrCurrencyMismatch` | `400 Bad Request` | Currency conflict in money calculation |
| `shared.ErrUnitMismatch` | `400 Bad Request` | UoM conflict in quantity calculation |
| `shared.ErrTenantRequired` | `400 Bad Request` | Request missing mandatory `X-Tenant-ID` header |
| `shared.ErrTenantSuspended` | `403 Forbidden` | Access blocked due to inactive tenant account |
