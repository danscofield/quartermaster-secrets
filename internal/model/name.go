package model

import (
	"fmt"
	"regexp"
)

var secretNamePattern = regexp.MustCompile(`^[a-zA-Z0-9.]+$`)

// ValidateSecretName checks that name is non-empty and contains only
// letters, digits, and periods.
func ValidateSecretName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if !secretNamePattern.MatchString(name) {
		return fmt.Errorf("name must contain only letters, numbers, and periods")
	}
	return nil
}

// IsValidationError reports whether err is a client-side name validation error.
func IsValidationError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return msg == "name is required" || msg == "name must contain only letters, numbers, and periods"
}
