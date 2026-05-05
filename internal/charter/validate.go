package charter

import (
	"fmt"
	"strings"

	"github.com/invopop/jsonschema"
)

// ValidationError describes a single field-level validation failure.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation failures.
type ValidationErrors []ValidationError

func (errs ValidationErrors) Error() string {
	lines := make([]string, len(errs))
	for i, e := range errs {
		lines[i] = e.Error()
	}
	return strings.Join(lines, "\n")
}

// Validate checks a Charter for required fields and valid enum values, returning any violations.
func Validate(c *Charter) ValidationErrors {
	var errs ValidationErrors

	if c.SchemaVersion != "1" {
		errs = append(errs, ValidationError{Field: "schema_version", Message: "must be \"1\""})
	}
	if c.ID == "" {
		errs = append(errs, ValidationError{Field: "id", Message: "is required"})
	}
	if c.Goal == "" {
		errs = append(errs, ValidationError{Field: "goal", Message: "is required — every charter needs a one-sentence goal"})
	}
	if c.Source.Type == "" {
		errs = append(errs, ValidationError{Field: "source.type", Message: "is required"})
	}
	if len(c.AcceptanceCriteria) == 0 {
		errs = append(errs, ValidationError{Field: "acceptance_criteria", Message: "at least one criterion is required"})
	}
	for i, ac := range c.AcceptanceCriteria {
		if ac.Statement == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("acceptance_criteria[%d].statement", i),
				Message: "each criterion must have a statement",
			})
		}
		if ac.Verification != "test" && ac.Verification != "manual" && ac.Verification != "metric" && ac.Verification != "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("acceptance_criteria[%d].verification", i),
				Message: "must be test, manual, or metric",
			})
		}
	}
	switch c.Risk {
	case RiskLow, RiskMedium, RiskHigh, RiskCritical:
	default:
		errs = append(errs, ValidationError{Field: "risk", Message: "must be one of low, medium, high, critical"})
	}
	switch c.Status {
	case StatusDraft, StatusReady, StatusApproved, StatusArchived:
	default:
		errs = append(errs, ValidationError{Field: "status", Message: "must be one of draft, ready, approved, archived"})
	}
	for _, u := range c.Unknowns {
		if u.Blocking && c.Status != StatusDraft {
			errs = append(errs, ValidationError{
				Field:   "unknowns",
				Message: fmt.Sprintf("blocking unknown %q must be resolved before charter can leave draft status", u.Question),
			})
		}
	}
	if (c.Risk == RiskHigh || c.Risk == RiskCritical) && c.RollbackPlan == "" {
		errs = append(errs, ValidationError{
			Field:   "rollback_plan",
			Message: fmt.Sprintf("required for %s risk charters", c.Risk),
		})
	}

	return errs
}

// JSONSchema returns the JSON Schema representation of the Charter type.
func JSONSchema() *jsonschema.Schema {
	return jsonschema.Reflect(&Charter{})
}