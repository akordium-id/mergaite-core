package document

import (
	"fmt"
	"slices"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

var validTransitions = map[Status][]Status{
	StatusDraft: {
		StatusSubmitted,
		StatusCancelled,
	},
	StatusSubmitted: {
		StatusApproved,
		StatusRejected,
		StatusDraft, // returned for revision
	},
	StatusApproved: {
		StatusPosted,
		StatusCancelled,
	},
	StatusPosted: {
		StatusCompleted,
		StatusCancelled,
	},
	StatusCompleted: {},
	StatusCancelled: {},
	StatusRejected:  {},
}

// IsValidTransition checks whether a state transition from 'from' to 'to' is permitted.
func IsValidTransition(from, to Status) bool {
	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}
	return slices.Contains(allowed, to)
}

// AllowedTransitions returns the list of allowed target statuses from the current status.
func AllowedTransitions(from Status) []Status {
	return validTransitions[from]
}

// ValidateTransition returns an error if transitioning from 'from' to 'to' violates workflow rules.
func ValidateTransition(from, to Status) error {
	if from == to {
		return fmt.Errorf("%w: document is already in status '%s'", shared.ErrInvalidInput, to)
	}
	if !IsValidTransition(from, to) {
		return fmt.Errorf("%w: illegal state transition from '%s' to '%s'", shared.ErrInvalidInput, from, to)
	}
	return nil
}
