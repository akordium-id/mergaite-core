package sequence

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

type ResetPolicy string

const (
	ResetNever   ResetPolicy = "never"
	ResetYearly  ResetPolicy = "yearly"
	ResetMonthly ResetPolicy = "monthly"
	ResetDaily   ResetPolicy = "daily"
)

func (r ResetPolicy) IsValid() bool {
	switch r {
	case ResetNever, ResetYearly, ResetMonthly, ResetDaily:
		return true
	default:
		return false
	}
}

// Sequence represents an auto-numbering rule and counter state for an entity type.
type Sequence struct {
	ID           shared.ID   `json:"id"`
	TenantID     shared.ID   `json:"tenant_id"`
	Code         string      `json:"code"`
	Name         string      `json:"name"`
	EntityType   string      `json:"entity_type"` // e.g. "document"
	SubType      string      `json:"sub_type"`    // e.g. "quotation", "invoice", "sales_order"
	Prefix       string      `json:"prefix"`
	Suffix       string      `json:"suffix"`
	Template     string      `json:"template"`
	Padding      int         `json:"padding"`
	StartValue   int64       `json:"start_value"`
	IncrementBy  int         `json:"increment_by"`
	CurrentValue int64       `json:"current_value"`
	ResetPolicy  ResetPolicy `json:"reset_policy"`
	LastNumber   string      `json:"last_number,omitempty"`
	LastResetAt  *time.Time  `json:"last_reset_at,omitempty"`
	IsActive     bool        `json:"is_active"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// ShouldReset checks if the counter should reset based on its reset policy and the given timestamp.
func (s *Sequence) ShouldReset(now time.Time) bool {
	if s.ResetPolicy == ResetNever || s.CurrentValue == 0 {
		return false
	}
	if s.LastResetAt == nil {
		return false
	}

	last := *s.LastResetAt
	switch s.ResetPolicy {
	case ResetYearly:
		return now.Year() != last.Year()
	case ResetMonthly:
		return now.Year() != last.Year() || now.Month() != last.Month()
	case ResetDaily:
		return now.Year() != last.Year() || now.YearDay() != last.YearDay()
	default:
		return false
	}
}

// NextValue computes the next numerical counter value and whether a policy reset occurred.
func (s *Sequence) NextValue(now time.Time) (nextVal int64, wasReset bool) {
	if s.ShouldReset(now) {
		return s.StartValue, true
	}

	if s.CurrentValue < s.StartValue {
		return s.StartValue, false
	}

	increment := int64(s.IncrementBy)
	if increment <= 0 {
		increment = 1
	}

	return s.CurrentValue + increment, false
}

var seqPatternRegex = regexp.MustCompile(`\{SEQ(?::(\d+))?\}`)

// FormatNumber formats a sequence counter according to the template pattern.
// Supported tokens:
//   - {PREFIX} -> Sequence Prefix
//   - {SUFFIX} -> Sequence Suffix
//   - {YYYY}   -> 4-digit Year (e.g. 2026)
//   - {YY}     -> 2-digit Year (e.g. 26)
//   - {MM}     -> 2-digit Month (01-12)
//   - {DD}     -> 2-digit Day (01-31)
//   - {SEQ}    -> Padded sequence using default padding
//   - {SEQ:N}  -> Padded sequence with N digits (e.g. {SEQ:5} -> 00042)
//   - Any dynamic token passed via extraTokens, e.g. {ORG}
func FormatNumber(template, prefix, suffix string, val int64, defaultPadding int, now time.Time, extraTokens map[string]string) string {
	if template == "" {
		template = "{PREFIX}/{YYYY}/{MM}/{SEQ:4}"
	}

	res := template
	res = strings.ReplaceAll(res, "{PREFIX}", prefix)
	res = strings.ReplaceAll(res, "{SUFFIX}", suffix)
	res = strings.ReplaceAll(res, "{YYYY}", now.Format("2006"))
	res = strings.ReplaceAll(res, "{YY}", now.Format("06"))
	res = strings.ReplaceAll(res, "{MM}", now.Format("01"))
	res = strings.ReplaceAll(res, "{DD}", now.Format("02"))

	for k, v := range extraTokens {
		tokenKey := "{" + k + "}"
		res = strings.ReplaceAll(res, tokenKey, v)
	}

	// Format {SEQ} or {SEQ:N}
	res = seqPatternRegex.ReplaceAllStringFunc(res, func(m string) string {
		padding := defaultPadding
		if padding <= 0 {
			padding = 4
		}

		match := seqPatternRegex.FindStringSubmatch(m)
		if len(match) > 1 && match[1] != "" {
			if n, err := strconv.Atoi(match[1]); err == nil && n > 0 {
				padding = n
			}
		}

		formatStr := fmt.Sprintf("%%0%dd", padding)
		return fmt.Sprintf(formatStr, val)
	})

	return res
}
