// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package specerrors

import (
	"fmt"
	"strings"
)

// ValidationError is the interface that every validation error must implement.
type ValidationError interface {
	error

	// Code returns a unique identifier of this error.
	Code() string
}

// ValidationPathError is the interface that validation errors related to paths must implement.
type ValidationPathError interface { // TODO no validation error using this interface yet
	// File returns the file path where the error was raised.
	File() string
}

// ValidationSeverityError is the interface that validation errors related to severities must implement.
type ValidationSeverityError interface { // TODO no validation error using this interface yet
	// File returns the file path where the error was raised.
	Severity() string
}

// ValidationErrors is an error that contains an iterable collection of validation error messages.
type ValidationErrors []ValidationError

// Collect filters the validation errors using the function given as a parameter.
func (ve ValidationErrors) Collect(collect func(elem ValidationError) bool) (ValidationErrors, ValidationErrors) {
	var errs ValidationErrors
	var filtered ValidationErrors

	for _, item := range ve {
		if collect(item) {
			errs = append(errs, item)
		} else {
			filtered = append(filtered, item)
		}
	}
	return errs, filtered
}

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "found 0 validation errors"
	}

	var message strings.Builder
	errorWord := "errors"
	if len(ve) == 1 {
		errorWord = "error"
	}
	fmt.Fprintf(&message, "found %v validation %v:\n", len(ve), errorWord)
	for idx, err := range ve {
		fmt.Fprintf(&message, "%4d. %v\n", idx+1, err)
	}

	return message.String()
}

// Append adds more validation errors.
func (ve *ValidationErrors) Append(moreErrs ValidationErrors) {
	if len(moreErrs) == 0 {
		return
	}

	errs := *ve
	errs = append(errs, moreErrs...)

	*ve = errs
}
