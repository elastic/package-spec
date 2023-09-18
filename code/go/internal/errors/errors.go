// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package errors

import (
	"fmt"
	"strings"
)

// ValidationError is the interface that every validation error must implement.
type ValidationError interface {
	error

	// Severity returns the validation severity of the error, the higher the pickier. 0 is mandatory.
	Severity() int

	// Code returns a unique identifier of this error.
	Code() string
}

// ValidationPathError is the interface that validation errors related to paths must implement.
type ValidationPathError interface {
	// File returns the file path where the error was raised.
	File() string
}

// StructuredError generic validation error
type StructuredError struct {
	err      error
	filePath string
	code     string // TODO : generate constants and types for each kind of error/code
	severity int
}

// Constant values to assign a severity to each validation error
const (
	Info     int = iota
	Medium       = iota
	Critical     = iota
)

// DanglingObjectIDError validation error for dangling object IDs
type DanglingObjectIDError struct {
	objectID   string
	objectType string
	filePath   string
}

func (e *DanglingObjectIDError) Error() string {
	return fmt.Sprintf("file \"%s\" is invalid: dangling reference found: %s (%s)", e.filePath, e.objectID, e.objectType)
}

func (e *DanglingObjectIDError) File() string {
	return e.filePath
}

func (e *DanglingObjectIDError) Code() string {
	return "DanglingObjectIDError"
}

func (e *DanglingObjectIDError) Severity() int {
	return Critical
}

// NewDanglingObjectIDError creates a new validation error for dangling object IDs
func NewDanglingObjectIDError(objectID, objectType, filePath string) *DanglingObjectIDError {
	return &DanglingObjectIDError{
		objectID:   objectID,
		objectType: objectType,
		filePath:   filePath,
	}
}

// JsonSchemaError validation error for dangling object IDs
type JsonSchemaError struct {
	field       string
	description string
	filePath    string
}

func (e *JsonSchemaError) Error() string {
	return fmt.Sprintf("field %s: %s", e.field, e.description)
}

func (e *JsonSchemaError) File() string {
	return e.filePath
}

func (e *JsonSchemaError) Code() string {
	return "JsonSchemaError"
}

func (e *JsonSchemaError) Severity() int {
	return Critical
}

// NewJsonSchemaError creates a new validation error for JSON schema issues
func NewJsonSchemaError(filePath, field, description string) *JsonSchemaError {
	return &JsonSchemaError{
		field:       field,
		description: description,
		filePath:    filePath,
	}
}

// NewStructuredError creates a generic validation error
func NewStructuredError(err error, filePath, code string, level int) *StructuredError {
	return &StructuredError{
		err:      err,
		filePath: filePath,
		code:     code,
		severity: level,
	}
}

func (e *StructuredError) Error() string {
	return e.err.Error()
}

func (e *StructuredError) Code() string {
	return e.code
}

func (e *StructuredError) Severity() int {
	return e.severity
}

func (e *StructuredError) File() string {
	return e.filePath
}

// var _ ValidationError = StructuredError{}

// ValidationErrors is an error that contains an iterable collection of validation error messages.
type ValidationErrors []ValidationError

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
