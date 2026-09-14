package document_test

import (
	"errors"
	"testing"

	"github.com/akordium-id/mergaite-core/internal/core/domain/document"
	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

func TestValidateTransition(t *testing.T) {
	tests := []struct {
		name      string
		from      document.Status
		to        document.Status
		expectErr bool
	}{
		// Valid transitions
		{
			name:      "Draft to Submitted is valid",
			from:      document.StatusDraft,
			to:        document.StatusSubmitted,
			expectErr: false,
		},
		{
			name:      "Draft to Cancelled is valid",
			from:      document.StatusDraft,
			to:        document.StatusCancelled,
			expectErr: false,
		},
		{
			name:      "Submitted to Approved is valid",
			from:      document.StatusSubmitted,
			to:        document.StatusApproved,
			expectErr: false,
		},
		{
			name:      "Submitted to Rejected is valid",
			from:      document.StatusSubmitted,
			to:        document.StatusRejected,
			expectErr: false,
		},
		{
			name:      "Submitted back to Draft (revision) is valid",
			from:      document.StatusSubmitted,
			to:        document.StatusDraft,
			expectErr: false,
		},
		{
			name:      "Approved to Posted is valid",
			from:      document.StatusApproved,
			to:        document.StatusPosted,
			expectErr: false,
		},
		{
			name:      "Posted to Completed is valid",
			from:      document.StatusPosted,
			to:        document.StatusCompleted,
			expectErr: false,
		},
		{
			name:      "Approved to Cancelled is valid",
			from:      document.StatusApproved,
			to:        document.StatusCancelled,
			expectErr: false,
		},

		// Invalid transitions
		{
			name:      "Same state transition should fail",
			from:      document.StatusDraft,
			to:        document.StatusDraft,
			expectErr: true,
		},
		{
			name:      "Draft directly to Posted is illegal",
			from:      document.StatusDraft,
			to:        document.StatusPosted,
			expectErr: true,
		},
		{
			name:      "Draft directly to Completed is illegal",
			from:      document.StatusDraft,
			to:        document.StatusCompleted,
			expectErr: true,
		},
		{
			name:      "Completed to Draft is illegal (terminal state)",
			from:      document.StatusCompleted,
			to:        document.StatusDraft,
			expectErr: true,
		},
		{
			name:      "Cancelled to Approved is illegal (terminal state)",
			from:      document.StatusCancelled,
			to:        document.StatusApproved,
			expectErr: true,
		},
		{
			name:      "Rejected to Approved is illegal (terminal state)",
			from:      document.StatusRejected,
			to:        document.StatusApproved,
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := document.ValidateTransition(tc.from, tc.to)
			if tc.expectErr {
				if err == nil {
					t.Errorf("expected error for transition %s -> %s, got nil", tc.from, tc.to)
				} else if !errors.Is(err, shared.ErrInvalidInput) {
					t.Errorf("expected ErrInvalidInput, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for transition %s -> %s: %v", tc.from, tc.to, err)
				}
			}
		})
	}
}

func TestAllowedTransitions(t *testing.T) {
	allowedDraft := document.AllowedTransitions(document.StatusDraft)
	if len(allowedDraft) != 2 {
		t.Errorf("expected 2 allowed transitions from Draft, got %d", len(allowedDraft))
	}

	allowedCompleted := document.AllowedTransitions(document.StatusCompleted)
	if len(allowedCompleted) != 0 {
		t.Errorf("expected 0 allowed transitions from Completed (terminal), got %d", len(allowedCompleted))
	}
}
