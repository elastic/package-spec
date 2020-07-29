package pkg

import (
	"../internal/epkg"
)

// ValidationErrors is an Error that contains a iterable collection of validation error messages.
type ValidationErrors epkg.ValidationErrors

// Validate validates a given package against the spec and returns any errors.
func Validate(packageRootPath string) error {
	p, err := epkg.NewFromPath(packageRootPath)
	if err != nil {
		return err
	}

	return p.Validate()
}