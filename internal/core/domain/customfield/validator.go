package customfield

import (
	"fmt"
	"regexp"
	"slices"
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

// ValidateValues validates an input values map against the provided definitions.
// It returns a sanitized map containing valid values and applies defaults for missing non-required fields.
func ValidateValues(definitions []Definition, rawValues map[string]any) (map[string]any, error) {
	if rawValues == nil {
		rawValues = make(map[string]any)
	}

	sanitized := make(map[string]any)

	for _, def := range definitions {
		if !def.IsActive {
			continue
		}

		val, exists := rawValues[def.Code]

		// Handle missing or nil value
		if !exists || val == nil {
			if def.IsRequired {
				if def.DefaultValue != nil {
					sanitized[def.Code] = def.DefaultValue
					continue
				}
				return nil, fmt.Errorf("%w: field '%s' (%s) is required", shared.ErrInvalidInput, def.Code, def.Name)
			}
			if def.DefaultValue != nil {
				sanitized[def.Code] = def.DefaultValue
			}
			continue
		}

		// Validate data type and constraints
		validatedVal, err := validateField(def, val)
		if err != nil {
			return nil, err
		}

		sanitized[def.Code] = validatedVal
	}

	return sanitized, nil
}

func validateField(def Definition, val any) (any, error) {
	switch def.DataType {
	case DataTypeText:
		str, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("%w: field '%s' must be a string", shared.ErrInvalidInput, def.Code)
		}
		if def.ValidationRules.Regex != "" {
			matched, err := regexp.MatchString(def.ValidationRules.Regex, str)
			if err != nil || !matched {
				return nil, fmt.Errorf("%w: field '%s' value does not match required format", shared.ErrInvalidInput, def.Code)
			}
		}
		return str, nil

	case DataTypeNumber:
		num, err := toFloat64(val)
		if err != nil {
			return nil, fmt.Errorf("%w: field '%s' must be a number", shared.ErrInvalidInput, def.Code)
		}
		if def.ValidationRules.Min != nil && num < *def.ValidationRules.Min {
			return nil, fmt.Errorf("%w: field '%s' value %v is less than minimum %v", shared.ErrInvalidInput, def.Code, num, *def.ValidationRules.Min)
		}
		if def.ValidationRules.Max != nil && num > *def.ValidationRules.Max {
			return nil, fmt.Errorf("%w: field '%s' value %v exceeds maximum %v", shared.ErrInvalidInput, def.Code, num, *def.ValidationRules.Max)
		}
		return num, nil

	case DataTypeBoolean:
		b, ok := val.(bool)
		if !ok {
			return nil, fmt.Errorf("%w: field '%s' must be a boolean", shared.ErrInvalidInput, def.Code)
		}
		return b, nil

	case DataTypeDate:
		str, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("%w: field '%s' must be a date string (YYYY-MM-DD)", shared.ErrInvalidInput, def.Code)
		}
		if _, err := time.Parse(time.DateOnly, str); err != nil {
			return nil, fmt.Errorf("%w: field '%s' must be in YYYY-MM-DD date format", shared.ErrInvalidInput, def.Code)
		}
		return str, nil

	case DataTypeSelect:
		str, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("%w: field '%s' must be a string option", shared.ErrInvalidInput, def.Code)
		}
		if !containsOption(def.Options, str) {
			return nil, fmt.Errorf("%w: field '%s' value '%s' is not in allowed options: %v", shared.ErrInvalidInput, def.Code, str, def.Options)
		}
		return str, nil

	case DataTypeMultiSelect:
		rawList, ok := val.([]any)
		if !ok {
			// Also support []string
			if strList, okStr := val.([]string); okStr {
				for _, s := range strList {
					if !containsOption(def.Options, s) {
						return nil, fmt.Errorf("%w: field '%s' option '%s' is not in allowed options: %v", shared.ErrInvalidInput, def.Code, s, def.Options)
					}
				}
				return strList, nil
			}
			return nil, fmt.Errorf("%w: field '%s' must be an array of string options", shared.ErrInvalidInput, def.Code)
		}

		result := make([]string, len(rawList))
		for i, item := range rawList {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("%w: field '%s' items must be strings", shared.ErrInvalidInput, def.Code)
			}
			if !containsOption(def.Options, s) {
				return nil, fmt.Errorf("%w: field '%s' option '%s' is not in allowed options: %v", shared.ErrInvalidInput, def.Code, s, def.Options)
			}
			result[i] = s
		}
		return result, nil

	case DataTypeJSON:
		return val, nil

	default:
		return val, nil
	}
}

func containsOption(options []string, target string) bool {
	return slices.Contains(options, target)
}

func toFloat64(val any) (float64, error) {
	switch v := val.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int32:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("not a number")
	}
}
