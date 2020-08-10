package validator

import (
	"github.com/elastic/package-spec/code/go/internal/validator"
)

// ValidateFromPath validates a package located at the given path against the
// appropriate specification and returns any errors.
func ValidateFromPath(packageRootPath string) error {
	// TODO: Noop for now. Implement actual validation.
	var errs validator.ValidationErrors
	return errs
}
