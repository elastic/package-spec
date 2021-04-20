package validator

import (
	"github.com/elastic/package-spec/code/go/internal/errors"
)

// ValidationErrors is an Error that contains a iterable collection of validation error messages.
type ValidationErrors errors.ValidationErrors
