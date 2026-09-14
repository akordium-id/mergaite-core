package organization_test

import (
	"testing"

	"github.com/akordium-id/mergaite-core/internal/core/domain/organization"
	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

func TestOrganization_CheckCircularReference(t *testing.T) {
	tenantID := shared.MustNewID()

	idA := shared.MustNewID()
	idB := shared.MustNewID()
	idC := shared.MustNewID()
	idD := shared.MustNewID()

	// A (Holding)
	//  └── B (PT Sub)
	//       └── C (Surabaya Branch)
	// D (Independent Branch)
	orgs := []organization.Organization{
		{ID: idA, TenantID: tenantID, ParentID: nil, Name: "Holding A"},
		{ID: idB, TenantID: tenantID, ParentID: &idA, Name: "Sub B"},
		{ID: idC, TenantID: tenantID, ParentID: &idB, Name: "Branch C"},
		{ID: idD, TenantID: tenantID, ParentID: nil, Name: "Branch D"},
	}

	// Case 1: Valid new parent (D under A)
	if err := organization.CheckCircularReference(orgs, idD, &idA); err != nil {
		t.Fatalf("expected valid hierarchy, got %v", err)
	}

	// Case 2: Self-parent (A under A)
	if err := organization.CheckCircularReference(orgs, idA, &idA); err == nil {
		t.Fatal("expected error for self-parent, got nil")
	}

	// Case 3: Circular reference (making A a child of C)
	if err := organization.CheckCircularReference(orgs, idA, &idC); err == nil {
		t.Fatal("expected error for circular hierarchy (A under descendant C), got nil")
	}

	// Case 4: Circular reference (making B a child of C)
	if err := organization.CheckCircularReference(orgs, idB, &idC); err == nil {
		t.Fatal("expected error for circular hierarchy (B under descendant C), got nil")
	}
}

func TestOrganization_BuildTree(t *testing.T) {
	tenantID := shared.MustNewID()
	idRoot := shared.MustNewID()
	idChild1 := shared.MustNewID()
	idChild2 := shared.MustNewID()
	idGrandchild := shared.MustNewID()

	orgs := []organization.Organization{
		{ID: idRoot, TenantID: tenantID, Name: "Root Org", Code: "root"},
		{ID: idChild1, TenantID: tenantID, ParentID: &idRoot, Name: "Child 1", Code: "c1"},
		{ID: idChild2, TenantID: tenantID, ParentID: &idRoot, Name: "Child 2", Code: "c2"},
		{ID: idGrandchild, TenantID: tenantID, ParentID: &idChild1, Name: "Grandchild 1", Code: "gc1"},
	}

	tree := organization.BuildTree(orgs)

	if len(tree) != 1 {
		t.Fatalf("expected 1 root organization, got %d", len(tree))
	}
	if tree[0].ID != idRoot {
		t.Fatalf("expected root ID %v, got %v", idRoot, tree[0].ID)
	}
	if len(tree[0].Children) != 2 {
		t.Fatalf("expected root to have 2 children, got %d", len(tree[0].Children))
	}

	// Find child 1
	var foundChild1 *organization.Organization
	for _, c := range tree[0].Children {
		if c.ID == idChild1 {
			foundChild1 = &c
			break
		}
	}
	if foundChild1 == nil {
		t.Fatal("expected to find child 1")
	}
	if len(foundChild1.Children) != 1 {
		t.Fatalf("expected child 1 to have 1 child, got %d", len(foundChild1.Children))
	}
	if foundChild1.Children[0].ID != idGrandchild {
		t.Fatalf("expected grandchild ID %v, got %v", idGrandchild, foundChild1.Children[0].ID)
	}
}
