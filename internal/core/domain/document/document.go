package document

import (
	"time"

	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

type DocumentType string

const (
	DocTypeSalesOrder    DocumentType = "sales_order"
	DocTypeInvoice       DocumentType = "invoice"
	DocTypePurchaseOrder DocumentType = "purchase_order"
	DocTypeQuotation     DocumentType = "quotation"
	DocTypeDeliveryNote  DocumentType = "delivery_note"
)

type Status string

const (
	StatusDraft     Status = "draft"
	StatusSubmitted Status = "submitted"
	StatusApproved  Status = "approved"
	StatusPosted    Status = "posted"
	StatusCompleted Status = "completed"
	StatusCancelled Status = "cancelled"
	StatusRejected  Status = "rejected"
)

// Document is the generic business document aggregate root (Sales Order, Invoice, PO, Quotation).
type Document struct {
	ID               shared.ID            `json:"id"`
	TenantID         shared.ID            `json:"tenant_id"`
	OrganizationID   shared.ID            `json:"organization_id"`
	OrganizationName string               `json:"organization_name,omitempty"`
	DocumentType     DocumentType         `json:"document_type"`
	DocumentNumber   string               `json:"document_number"`
	DocumentDate     time.Time            `json:"document_date"`
	PartyID          *shared.ID           `json:"party_id,omitempty"`
	PartyName        string               `json:"party_name,omitempty"`
	Status           Status               `json:"status"`
	TotalAmount      shared.Money         `json:"total_amount"`
	Notes            string               `json:"notes,omitempty"`
	Metadata         map[string]any       `json:"metadata,omitempty"`
	CreatedBy        *shared.ID           `json:"created_by,omitempty"`
	Lines            []DocumentLine       `json:"lines,omitempty"`
	Transitions      []DocumentTransition `json:"transitions,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}

// DocumentLine represents an individual line item in a document.
type DocumentLine struct {
	ID          shared.ID      `json:"id"`
	TenantID    shared.ID      `json:"tenant_id"`
	DocumentID  shared.ID      `json:"document_id"`
	LineNumber  int32          `json:"line_number"`
	ProductID   *shared.ID     `json:"product_id,omitempty"`
	ProductName string         `json:"product_name,omitempty"`
	Description string         `json:"description"`
	Quantity    float64        `json:"quantity"`
	UnitID      shared.ID      `json:"unit_id"`
	UnitCode    string         `json:"unit_code,omitempty"`
	UnitPrice   shared.Money   `json:"unit_price"`
	Subtotal    shared.Money   `json:"subtotal"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// DocumentTransition logs state changes in the document's lifecycle.
type DocumentTransition struct {
	ID         shared.ID `json:"id"`
	TenantID   shared.ID `json:"tenant_id"`
	DocumentID shared.ID `json:"document_id"`
	FromStatus Status    `json:"from_status"`
	ToStatus   Status    `json:"to_status"`
	Reason     string    `json:"reason,omitempty"`
	ActorID    *shared.ID`json:"actor_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
