package shared_test

import (
	"testing"
	"time"

	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

func TestID_NewID(t *testing.T) {
	id1, err := shared.NewID()
	if err != nil {
		t.Fatalf("expected no error creating ID, got %v", err)
	}
	if id1 == shared.NilID() {
		t.Fatal("expected non-nil ID")
	}

	time.Sleep(2 * time.Millisecond)

	id2, err := shared.NewID()
	if err != nil {
		t.Fatalf("expected no error creating second ID, got %v", err)
	}

	// UUIDv7 is time-ordered: id1 should be lexically <= id2
	if id1.String() >= id2.String() {
		t.Fatalf("expected id1 (%s) < id2 (%s)", id1.String(), id2.String())
	}
}

func TestID_ParseID(t *testing.T) {
	id := shared.MustNewID()
	str := id.String()

	parsed, err := shared.ParseID(str)
	if err != nil {
		t.Fatalf("unexpected error parsing ID: %v", err)
	}
	if parsed != id {
		t.Fatalf("expected %v, got %v", id, parsed)
	}

	_, err = shared.ParseID("invalid-uuid-string")
	if err == nil {
		t.Fatal("expected error parsing invalid uuid string, got nil")
	}
}

func TestID_PgUUIDConversion(t *testing.T) {
	id := shared.MustNewID()
	pgUUID := shared.ToPgUUID(id)

	if !pgUUID.Valid {
		t.Fatal("expected pgUUID to be valid")
	}

	converted := shared.FromPgUUID(pgUUID)
	if converted != id {
		t.Fatalf("expected %v, got %v", id, converted)
	}
}
