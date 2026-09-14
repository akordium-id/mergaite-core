package document

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/akordium-id/mergaite-core/internal/core/domain/document"
	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

type CreateDocumentLineInput struct {
	ProductID   *shared.ID     `json:"product_id,omitempty"`
	Description string         `json:"description"`
	Quantity    float64        `json:"quantity"`
	UnitID      shared.ID      `json:"unit_id"`
	UnitPrice   shared.Money   `json:"unit_price"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type CreateDocumentCommand struct {
	OrganizationID shared.ID                 `json:"organization_id"`
	DocumentType   document.DocumentType     `json:"document_type"`
	DocumentNumber string                    `json:"document_number,omitempty"`
	DocumentDate   *time.Time                `json:"document_date,omitempty"`
	PartyID        *shared.ID                `json:"party_id,omitempty"`
	Currency       string                    `json:"currency,omitempty"`
	Notes          string                    `json:"notes,omitempty"`
	Metadata       map[string]any            `json:"metadata,omitempty"`
	Lines          []CreateDocumentLineInput `json:"lines,omitempty"`
	CreatedBy      *shared.ID                `json:"created_by,omitempty"`
}

type TransitionDocumentCommand struct {
	DocumentID   shared.ID       `json:"document_id"`
	TargetStatus document.Status `json:"target_status"`
	Reason       string          `json:"reason,omitempty"`
	ActorID      *shared.ID      `json:"actor_id,omitempty"`
}

type DocumentListResult struct {
	Items    []document.Document `json:"items"`
	Total    int64               `json:"total"`
	Page     int32               `json:"page"`
	PageSize int32               `json:"page_size"`
}

type Usecase interface {
	CreateDocument(ctx context.Context, cmd CreateDocumentCommand) (*document.Document, error)
	GetDocumentByID(ctx context.Context, id shared.ID) (*document.Document, error)
	GetDocumentByNumber(ctx context.Context, docType document.DocumentType, number string) (*document.Document, error)
	ListDocuments(ctx context.Context, page, pageSize int32, orgID *shared.ID, docType *document.DocumentType, status *document.Status) (*DocumentListResult, error)
	TransitionDocument(ctx context.Context, cmd TransitionDocumentCommand) (*document.Document, error)
}

type usecase struct {
	docRepo document.Repository
}

func NewUsecase(docRepo document.Repository) Usecase {
	return &usecase{
		docRepo: docRepo,
	}
}

func (u *usecase) CreateDocument(ctx context.Context, cmd CreateDocumentCommand) (*document.Document, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	if cmd.OrganizationID == shared.NilID() {
		return nil, fmt.Errorf("%w: organization_id is required", shared.ErrInvalidInput)
	}

	docType := cmd.DocumentType
	if docType == "" {
		return nil, fmt.Errorf("%w: document_type is required", shared.ErrInvalidInput)
	}

	docID, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	docNum := strings.TrimSpace(cmd.DocumentNumber)
	if docNum == "" {
		prefix := strings.ToUpper(string(docType))
		docNum = fmt.Sprintf("%s-%s-%s", prefix, time.Now().Format("20060102"), docID.String()[:8])
	}

	// Ensure document number uniqueness within tenant + doc_type
	existing, err := u.docRepo.GetDocumentByNumber(ctx, tenantID, docType, docNum)
	if err != nil && err != shared.ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: document with number '%s' already exists for type '%s'", shared.ErrAlreadyExists, docNum, docType)
	}

	now := time.Now().UTC()
	docDate := now
	if cmd.DocumentDate != nil && !cmd.DocumentDate.IsZero() {
		docDate = cmd.DocumentDate.UTC()
	}

	currency := strings.TrimSpace(cmd.Currency)
	if currency == "" {
		if len(cmd.Lines) > 0 && cmd.Lines[0].UnitPrice.Currency() != "" {
			currency = cmd.Lines[0].UnitPrice.Currency()
		} else {
			currency = "IDR"
		}
	}

	total := shared.ZeroMoney(currency)
	lines := make([]document.DocumentLine, len(cmd.Lines))
	for i, l := range cmd.Lines {
		if l.Quantity <= 0 {
			return nil, fmt.Errorf("%w: line %d quantity must be greater than zero", shared.ErrInvalidInput, i+1)
		}
		if l.UnitID == shared.NilID() {
			return nil, fmt.Errorf("%w: line %d unit_id is required", shared.ErrInvalidInput, i+1)
		}

		unitPrice := l.UnitPrice
		if unitPrice.Currency() == "" {
			unitPrice = shared.MustNewMoney(l.UnitPrice.Amount(), currency)
		} else if unitPrice.Currency() != currency {
			return nil, fmt.Errorf("%w: line %d currency '%s' does not match document currency '%s'", shared.ErrInvalidInput, i+1, unitPrice.Currency(), currency)
		}

		subtotal := unitPrice.MultiplyFloat(l.Quantity)
		newTotal, err := total.Add(subtotal)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		total = newTotal

		lineID, err := shared.NewID()
		if err != nil {
			return nil, err
		}

		meta := l.Metadata
		if meta == nil {
			meta = make(map[string]any)
		}

		lines[i] = document.DocumentLine{
			ID:          lineID,
			TenantID:    tenantID,
			DocumentID:  docID,
			LineNumber:  int32(i + 1),
			ProductID:   l.ProductID,
			Description: strings.TrimSpace(l.Description),
			Quantity:    l.Quantity,
			UnitID:      l.UnitID,
			UnitPrice:   unitPrice,
			Subtotal:    subtotal,
			Metadata:    meta,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	}

	metadata := cmd.Metadata
	if metadata == nil {
		metadata = make(map[string]any)
	}

	doc := &document.Document{
		ID:             docID,
		TenantID:       tenantID,
		OrganizationID: cmd.OrganizationID,
		DocumentType:   docType,
		DocumentNumber: docNum,
		DocumentDate:   docDate,
		PartyID:        cmd.PartyID,
		Status:         document.StatusDraft,
		TotalAmount:    total,
		Notes:          strings.TrimSpace(cmd.Notes),
		Metadata:       metadata,
		CreatedBy:      cmd.CreatedBy,
		Lines:          lines,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := u.docRepo.CreateDocument(ctx, doc); err != nil {
		return nil, err
	}

	if len(lines) > 0 {
		if err := u.docRepo.CreateLines(ctx, lines); err != nil {
			return nil, err
		}
	}

	return doc, nil
}

func (u *usecase) GetDocumentByID(ctx context.Context, id shared.ID) (*document.Document, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	return u.docRepo.GetDocumentByID(ctx, tenantID, id)
}

func (u *usecase) GetDocumentByNumber(ctx context.Context, docType document.DocumentType, number string) (*document.Document, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	return u.docRepo.GetDocumentByNumber(ctx, tenantID, docType, strings.TrimSpace(number))
}

func (u *usecase) ListDocuments(ctx context.Context, page, pageSize int32, orgID *shared.ID, docType *document.DocumentType, status *document.Status) (*DocumentListResult, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	filter := document.DocumentFilter{
		OrganizationID: orgID,
		DocumentType:   docType,
		Status:         status,
		Limit:          pageSize,
		Offset:         offset,
	}

	items, total, err := u.docRepo.ListDocuments(ctx, tenantID, filter)
	if err != nil {
		return nil, err
	}

	return &DocumentListResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (u *usecase) TransitionDocument(ctx context.Context, cmd TransitionDocumentCommand) (*document.Document, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	doc, err := u.docRepo.GetDocumentByID(ctx, tenantID, cmd.DocumentID)
	if err != nil {
		return nil, err
	}

	if err := document.ValidateTransition(doc.Status, cmd.TargetStatus); err != nil {
		return nil, err
	}

	if err := u.docRepo.UpdateDocumentStatus(ctx, tenantID, cmd.DocumentID, cmd.TargetStatus); err != nil {
		return nil, err
	}

	transID, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	trans := &document.DocumentTransition{
		ID:         transID,
		TenantID:   tenantID,
		DocumentID: cmd.DocumentID,
		FromStatus: doc.Status,
		ToStatus:   cmd.TargetStatus,
		Reason:     strings.TrimSpace(cmd.Reason),
		ActorID:    cmd.ActorID,
		CreatedAt:  time.Now().UTC(),
	}

	if err := u.docRepo.RecordTransition(ctx, trans); err != nil {
		return nil, err
	}

	// Refresh document state
	return u.docRepo.GetDocumentByID(ctx, tenantID, cmd.DocumentID)
}
