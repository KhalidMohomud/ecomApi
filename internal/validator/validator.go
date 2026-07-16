// Package validator turns the errors gin's request binding produces
// into the plain []string the API's error envelope expects
// (utils.ErrorResponse.Errors), instead of letting each handler
// format go-playground/validator errors by hand.
package validator

import (
	"errors"
	"fmt"

	govalidator "github.com/go-playground/validator/v10"
)

// FormatValidationErrors converts the error returned by
// c.ShouldBindJSON into a slice of human-readable messages, one per
// invalid field.
//
// It's written defensively: if err isn't a govalidator.ValidationErrors
// (e.g. the request body was malformed JSON, not just invalid
// values), it falls back to the raw error message instead of
// panicking or returning an empty slice.
func FormatValidationErrors(err error) []string {
	var ve govalidator.ValidationErrors
	if errors.As(err, &ve) {
		messages := make([]string, 0, len(ve))
		for _, fieldErr := range ve {
			messages = append(messages, formatFieldError(fieldErr))
		}
		return messages
	}
	return []string{err.Error()}
}

// formatFieldError renders one failed validation rule as a sentence.
// Only the tags actually used by this project's DTOs are handled by
// name; anything else falls through to a generic message rather than
// growing this switch preemptively.
func formatFieldError(fe govalidator.FieldError) string {
	field := fe.Field()

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, fe.Param())
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}
