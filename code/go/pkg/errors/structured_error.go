package errors

import "fmt"

// StructuredError generic validation error
type StructuredError struct {
	err  error
	code string // TODO : generate constants and types for each kind of error/code
}

// NewStructuredError creates a generic validation error
func NewStructuredError(err error, code string) *StructuredError {
	return &StructuredError{
		err:  err,
		code: code,
	}
}

// Error returns the message error
func (e *StructuredError) Error() string {
	if e.code == "" {
		return e.err.Error()
	}
	return fmt.Sprintf("%s (%s)", e.err.Error(), e.code)
}

// Code returns a unique code assigned to this error
func (e *StructuredError) Code() string {
	return e.code
}
