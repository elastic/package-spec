// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package errors

import "fmt"

// StructuredError generic validation error
type StructuredError struct {
	err  error
	code string
}

// NewStructuredError creates a generic validation error
func NewStructuredError(err error, code string) *StructuredError {
	return &StructuredError{
		err:  err,
		code: code,
	}
}

// NewStructuredErrorf creates a generic validation error with unassigned code
func NewStructuredErrorf(format string, a ...any) *StructuredError {
	return NewStructuredError(fmt.Errorf(format, a...), UnassignedCode)
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

// Unwrap returns the wrapped error
func (e *StructuredError) Unwrap() error {
	return e.err
}
