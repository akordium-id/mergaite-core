package organization

import (
	"fmt"
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

type Type string

const (
	TypeCompany    Type = "company"
	TypeBranch     Type = "branch"
	TypeDepartment Type = "department"
	TypeLocation   Type = "location"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusArchived Status = "archived"
)

// Organization represents a legal entity, branch, department, or location within a Tenant.
type Organization struct {
	ID        shared.ID      `json:"id"`
	TenantID  shared.ID      `json:"tenant_id"`
	ParentID  *shared.ID     `json:"parent_id,omitempty"`
	Code      string         `json:"code"`
	Name      string         `json:"name"`
	LegalName string         `json:"legal_name,omitempty"`
	Type      Type           `json:"type"`
	Status    Status         `json:"status"`
	Settings  map[string]any `json:"settings,omitempty"`
	Children  []Organization `json:"children,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// CheckCircularReference verifies that setting newParentID on targetID does not create a cycle.
func CheckCircularReference(orgs []Organization, targetID shared.ID, newParentID *shared.ID) error {
	if newParentID == nil || *newParentID == shared.NilID() {
		return nil
	}

	if *newParentID == targetID {
		return fmt.Errorf("%w: organization cannot be its own parent", shared.ErrInvalidInput)
	}

	// Build map parent lookup
	parentMap := make(map[shared.ID]*shared.ID)
	for _, org := range orgs {
		parentMap[org.ID] = org.ParentID
	}

	// Traverse upwards from newParentID to see if targetID is an ancestor
	current := newParentID
	visited := make(map[shared.ID]bool)

	for current != nil && *current != shared.NilID() {
		if *current == targetID {
			return fmt.Errorf("%w: circular hierarchy detected (target is an ancestor of new parent)", shared.ErrInvalidInput)
		}
		if visited[*current] {
			return fmt.Errorf("%w: circular reference detected in existing hierarchy", shared.ErrInvalidInput)
		}
		visited[*current] = true
		current = parentMap[*current]
	}

	return nil
}

// BuildTree converts a flat slice of organizations into a nested tree based on ParentID.
func BuildTree(orgs []Organization) []Organization {
	childrenMap := make(map[shared.ID][]Organization)
	orgMap := make(map[shared.ID]Organization)
	var rootIDs []shared.ID

	for _, o := range orgs {
		orgMap[o.ID] = o
		if o.ParentID == nil || *o.ParentID == shared.NilID() {
			rootIDs = append(rootIDs, o.ID)
		} else {
			childrenMap[*o.ParentID] = append(childrenMap[*o.ParentID], o)
		}
	}

	var buildNode func(id shared.ID) Organization
	buildNode = func(id shared.ID) Organization {
		node := orgMap[id]
		rawChildren := childrenMap[id]
		node.Children = make([]Organization, len(rawChildren))
		for i, c := range rawChildren {
			node.Children[i] = buildNode(c.ID)
		}
		return node
	}

	roots := make([]Organization, len(rootIDs))
	for i, rid := range rootIDs {
		roots[i] = buildNode(rid)
	}

	return roots
}
