package pkg

import "fmt"

// ValidationErrors is an Error that contains a iterable collection of validation error messages.
type ValidationErrors []string

func (ve ValidationErrors) Error() string {
	return fmt.Sprintf("found %v validation errors", len(ve))
}

// Validate validates a given package against the spec and returns any errors.
func Validate() ValidationErrors {
	return nil
}